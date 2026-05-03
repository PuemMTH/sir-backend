package handler

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/syumai/workers/cloudflare"
	"github.com/syumai/workers/cloudflare/r2"

	"github.com/puemmth/sir-backend/internal/middleware"
	"github.com/puemmth/sir-backend/internal/store"
)

// compileClient is a dedicated HTTP client for proxying to the compile server.
// Using a separate client (not http.DefaultClient) with an explicit timeout avoids
// context-lifetime conflicts with the incoming CF request context.
var compileClient = &http.Client{Timeout: 120 * time.Second}

// Compile handles POST /api/compile.
//
// Flow:
//  1. Compute MD5(engine + ":" + source) as the cache key.
//  2. Look up the hash in the pdf_cache D1 table.
//  3. Cache hit  → stream the PDF directly from R2 (X-Cache: HIT).
//  4. Cache miss → proxy to the compile server, store the resulting PDF in R2,
//     record the hash in D1, then return the PDF (X-Cache: MISS).
//
// Compile errors from the upstream server are forwarded verbatim to the caller.
func Compile(w http.ResponseWriter, r *http.Request) {
	// Recover from any panic so CORS headers (set by the outer middleware) are
	// always flushed before the connection closes.
	defer func() {
		if rec := recover(); rec != nil {
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

	// ── 1. Compute cache key ──────────────────────────────────────────────────
	h := md5.New()
	h.Write([]byte(req.Engine + ":" + req.Source))
	sourceHash := hex.EncodeToString(h.Sum(nil))

	// ── 2. Open store ─────────────────────────────────────────────────────────
	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	// ── 3. Cache lookup ───────────────────────────────────────────────────────
	// Use a detached context so D1/R2 ops are not tied to the CF request lifetime.
	ctx := context.Background()

	cached, err := s.GetPDFCache(ctx, sourceHash)
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}

	if cached != nil {
		bucket, bucketErr := r2.NewBucket("LATEX_BUCKET")
		if bucketErr == nil {
			obj, getErr := bucket.Get(cached.R2Key)
			if getErr == nil && obj != nil {
				// Cache hit — serve from R2
				w.Header().Set("Content-Type", "application/pdf")
				w.Header().Set("X-Cache", "HIT")
				io.Copy(w, obj.Body)
				return
			}
		}
		// R2 object gone (stale record) — fall through and recompile
	}

	// ── 4. Cache miss — proxy to compile server ───────────────────────────────
	compileURL := cloudflare.Getenv("COMPILE_URL")
	if compileURL == "" {
		middleware.WriteError(w, "compile_service_not_configured", http.StatusInternalServerError)
		return
	}

	bodyBytes, _ := json.Marshal(map[string]string{
		"source": req.Source,
		"engine": req.Engine,
	})

	// Use context.Background() — NOT r.Context() — for the outbound request.
	// Passing the CF incoming-request context to a net/http outbound call can
	// trigger a WASM trap in syumai/workers because the CF context has a
	// non-standard Done channel implementation.
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, compileURL, bytes.NewReader(bodyBytes))
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	upstreamReq.Header.Set("Content-Type", "application/json")

	resp, err := compileClient.Do(upstreamReq)
	if err != nil {
		middleware.WriteError(w, "compile_service_unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}

	// Forward non-200 responses (e.g. 422 compile error) verbatim
	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(pdfBytes)
		return
	}

	// ── 5. Store PDF in R2 and record in D1 (best-effort) ────────────────────
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

	// ── 6. Return PDF ─────────────────────────────────────────────────────────
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("X-Cache", "MISS")
	w.Write(pdfBytes)
}
