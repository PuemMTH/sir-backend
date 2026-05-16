package handler

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"syscall/js"

	"github.com/syumai/workers/cloudflare"
	"github.com/syumai/workers/cloudflare/r2"

	"github.com/sir-labs/sir-api/internal/middleware"
	"github.com/sir-labs/sir-api/internal/store"
)

// jsAwaitPromise blocks until the JS Promise resolves or rejects.
func jsAwaitPromise(promise js.Value) (js.Value, error) {
	resCh := make(chan js.Value, 1)
	errCh := make(chan error, 1)
	var then, catch js.Func
	then = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer then.Release()
		resCh <- args[0]
		return js.Undefined()
	})
	catch = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer catch.Release()
		errCh <- errors.New(args[0].Call("toString").String())
		return js.Undefined()
	})
	promise.Call("then", then).Call("catch", catch)
	select {
	case v := <-resCh:
		return v, nil
	case e := <-errCh:
		return js.Value{}, e
	}
}

// upstreamPost sends a JSON POST using the global fetch function invoked via
// js.Value.Invoke (not .Call), which preserves the correct `this` binding and
// avoids the "Illegal invocation" panic in Cloudflare Go/WASM Workers.
// syumai/workers' fetch package uses namespace.Call("fetch", ...) which loses
// the `this` reference — Invoke calls the function directly as a free function.
func upstreamPost(url string, body []byte) (int, []byte, error) {
	// Build headers and init objects
	headersClass := js.Global().Get("Headers")
	headersObj := headersClass.New()
	headersObj.Call("set", "Content-Type", "application/json")

	objectClass := js.Global().Get("Object")
	initObj := objectClass.New()
	initObj.Set("method", "POST")
	initObj.Set("headers", headersObj)
	initObj.Set("body", string(body))

	// Invoke fetch as a free function — NOT via .Call which loses `this`
	globalFetch := js.Global().Get("fetch")
	if globalFetch.IsUndefined() {
		return 0, nil, errors.New("fetch not available")
	}
	promise := globalFetch.Invoke(url, initObj)

	// Await the response promise
	jsRes, err := jsAwaitPromise(promise)
	if err != nil {
		return 0, nil, err
	}

	status := jsRes.Get("status").Int()

	// Await response.arrayBuffer()
	bufPromise := jsRes.Call("arrayBuffer")
	jsBuf, err := jsAwaitPromise(bufPromise)
	if err != nil {
		return 0, nil, err
	}

	uint8ArrayClass := js.Global().Get("Uint8Array")
	uint8Array := uint8ArrayClass.New(jsBuf)
	length := uint8Array.Get("length").Int()
	result := make([]byte, length)
	js.CopyBytesToGo(result, uint8Array)

	return status, result, nil
}

// bodyAsString reads resp body as string (used for non-200 upstream errors).
func bodyAsString(b []byte) string {
	return strings.TrimSpace(string(b))
}

// cfCacheGet checks Cloudflare's Cache API for a cached PDF. Returns nil on miss or error.
func cfCacheGet(cacheURL string) []byte {
	cacheObj := js.Global().Get("caches").Get("default")
	if cacheObj.IsUndefined() {
		return nil
	}
	cacheReq := js.Global().Get("Request").New(cacheURL)
	result, err := jsAwaitPromise(cacheObj.Call("match", cacheReq))
	if err != nil || result.IsUndefined() || result.IsNull() {
		return nil
	}
	jsBuf, err := jsAwaitPromise(result.Call("arrayBuffer"))
	if err != nil {
		return nil
	}
	uint8Arr := js.Global().Get("Uint8Array").New(jsBuf)
	length := uint8Arr.Get("length").Int()
	if length == 0 {
		return nil
	}
	data := make([]byte, length)
	js.CopyBytesToGo(data, uint8Arr)
	return data
}

// cfCachePut stores a PDF in Cloudflare's Cache API.
func cfCachePut(cacheURL string, data []byte) {
	cfCachePutWithContentType(cacheURL, data, "application/pdf")
}

func cfCachePutWithContentType(cacheURL string, data []byte, contentType string) {
	cacheObj := js.Global().Get("caches").Get("default")
	if cacheObj.IsUndefined() {
		return
	}
	uint8Arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(uint8Arr, data)

	headers := js.Global().Get("Headers").New()
	headers.Call("set", "Content-Type", contentType)
	headers.Call("set", "Cache-Control", "public, max-age=86400")

	initObj := js.Global().Get("Object").New()
	initObj.Set("status", 200)
	initObj.Set("headers", headers)

	resp := js.Global().Get("Response").New(uint8Arr.Get("buffer"), initObj)
	cacheReq := js.Global().Get("Request").New(cacheURL)
	jsAwaitPromise(cacheObj.Call("put", cacheReq, resp)) //nolint
}

// assetFile is a file entry sent to the upstream compile server.
type assetFile struct {
	Name    string `json:"name"`
	Content string `json:"content"` // base64-encoded
}

// Compile handles POST /api/compile.
//
// Flow:
//  1. Fetch the user's uploaded assets from D1 (metadata only).
//  2. Compute MD5(engine + ":" + source + asset IDs) as the cache key.
//  3. Look up the hash in the pdf_cache D1 table.
//  4. Cache hit  → stream the PDF directly from R2 (X-Cache: HIT).
//  5. Cache miss → fetch asset contents from R2, proxy everything to the
//     compile server, store the resulting PDF in R2, record the hash in D1,
//     then return the PDF (X-Cache: MISS).
//
// Compile errors from the upstream server are forwarded verbatim to the caller.
func Compile(w http.ResponseWriter, r *http.Request) {
	// Recover from any panic so CORS headers (set by the outer middleware) are
	// always flushed before the connection closes.
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[compile] PANIC: %v", rec)
			middleware.WriteError(w, "internal_error", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Source string `json:"source"`
		Engine string `json:"engine"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Source == "" {
		middleware.WriteError(w, "invalid_request: source required", http.StatusBadRequest)
		return
	}
	if req.Engine == "" {
		req.Engine = "lualatex"
	}

	claims := middleware.ClaimsFromCtx(r.Context())

	// ── 1. Open store ─────────────────────────────────────────────────────────
	log.Printf("[compile] opening store...")
	s, err := store.Open()
	if err != nil {
		log.Printf("[compile] store.Open failed: %v", err)
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()
	log.Printf("[compile] store opened")

	// ── 2. Fetch user asset metadata (IDs only for cache key) ─────────────────
	ctx := context.Background()
	assets, err := s.ListUserAssets(ctx, claims.Sub)
	if err != nil {
		log.Printf("[compile] ListUserAssets failed: %v", err)
		// Non-fatal — compile without assets
		assets = nil
	}

	// ── 3. Compute cache key (source + engine + asset IDs) ───────────────────
	h := md5.New()
	h.Write([]byte(req.Engine + ":" + req.Source))
	for _, a := range assets {
		h.Write([]byte(a.ID))
	}
	sourceHash := hex.EncodeToString(h.Sum(nil))
	cfCacheURL := "https://cache.internal/pdf/" + sourceHash

	// ── 4a. Cloudflare edge cache (fastest path) ──────────────────────────────
	if pdf := cfCacheGet(cfCacheURL); len(pdf) > 0 {
		log.Printf("[compile] CF cache hit (%d bytes)", len(pdf))
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("X-Cache", "CF-HIT")
		w.Write(pdf)
		return
	}

	// ── 4. Cache lookup ───────────────────────────────────────────────────────
	log.Printf("[compile] querying pdf_cache...")
	cached, err := s.GetPDFCache(ctx, sourceHash)
	if err != nil {
		log.Printf("[compile] GetPDFCache failed: %v", err)
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	log.Printf("[compile] cache lookup done, hit=%v", cached != nil)

	if cached != nil {
		bucket, bucketErr := r2.NewBucket("LATEX_BUCKET")
		if bucketErr == nil {
			obj, getErr := bucket.Get(cached.R2Key)
			if getErr == nil && obj != nil {
				pdf, readErr := io.ReadAll(obj.Body)
				if readErr == nil {
					cfCachePut(cfCacheURL, pdf)
					w.Header().Set("Content-Type", "application/pdf")
					w.Header().Set("X-Cache", "HIT")
					w.Write(pdf)
					return
				}
			}
		}
		// R2 object gone (stale record) — fall through and recompile
	}

	// ── 5. Cache miss — fetch asset contents from R2 ──────────────────────────
	log.Printf("[compile] cache miss, fetching assets from R2...")
	var files []assetFile
	if len(assets) > 0 {
		bucket, bucketErr := r2.NewBucket("LATEX_BUCKET")
		if bucketErr == nil {
			for _, asset := range assets {
				obj, getErr := bucket.Get(asset.R2Key)
				if getErr != nil || obj == nil {
					continue
				}
				data, readErr := io.ReadAll(obj.Body)
				if readErr != nil {
					continue
				}
				files = append(files, assetFile{
					Name:    asset.Name,
					Content: base64.StdEncoding.EncodeToString(data),
				})
			}
		}
	}
	log.Printf("[compile] fetched %d asset(s), proxying to upstream...", len(files))

	// ── 6. Proxy to compile server ────────────────────────────────────────────
	compileURL, _ := s.GetSetting(ctx, "compile_url")
	if compileURL == "" {
		compileURL = cloudflare.Getenv("COMPILE_URL")
	}
	if compileURL == "" {
		middleware.WriteError(w, "compile_service_not_configured", http.StatusInternalServerError)
		return
	}

	bodyBytes, _ := json.Marshal(map[string]any{
		"source": req.Source,
		"engine": req.Engine,
		"files":  files,
	})

	log.Printf("[compile] sending request to %s", compileURL)
	statusCode, pdfBytes, err := upstreamPost(compileURL, bodyBytes)
	if err != nil {
		log.Printf("[compile] upstream request failed: %v", err)
		middleware.WriteError(w, "compile_service_unavailable", http.StatusBadGateway)
		return
	}
	log.Printf("[compile] upstream responded with status %d, %d bytes", statusCode, len(pdfBytes))

	// Forward non-200 responses (e.g. 422 compile error) verbatim
	if statusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(pdfBytes)
		return
	}

	// ── 7. Store PDF in R2 and record in D1 (best-effort) ────────────────────
	r2Key := fmt.Sprintf("pdf/%s.pdf", sourceHash)
	bucket, bucketErr := r2.NewBucket("LATEX_BUCKET")
	if bucketErr == nil {
		_, putErr := bucket.Put(r2Key, io.NopCloser(bytes.NewReader(pdfBytes)), &r2.PutOptions{
			HTTPMetadata: r2.HTTPMetadata{ContentType: "application/pdf"},
		})
		if putErr == nil {
			_ = s.CreatePDFCache(ctx, sourceHash, r2Key, req.Engine)
		}
	}

	// ── 8. Return PDF ─────────────────────────────────────────────────────────
	cfCachePut(cfCacheURL, pdfBytes)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("X-Cache", "MISS")
	w.Write(pdfBytes)
}
