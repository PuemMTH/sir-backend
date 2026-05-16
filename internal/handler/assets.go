package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/syumai/workers/cloudflare/r2"

	"github.com/sir-labs/sir-api/internal/middleware"
	"github.com/sir-labs/sir-api/internal/store"
	"github.com/sir-labs/sir-api/internal/token"
)

const maxAssetSize = 10 << 20    // 10 MB
const maxThumbnailSize = 1 << 20 // 1 MB

// Assets handles GET /api/assets (list) and POST /api/assets (upload).
func Assets(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	switch r.Method {
	case http.MethodGet:
		assets, err := s.ListUserAssets(r.Context(), claims.Sub)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(assets)

	case http.MethodPost:
		if err := r.ParseMultipartForm(maxAssetSize); err != nil {
			middleware.WriteError(w, "invalid_request: file too large or bad multipart form", http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			middleware.WriteError(w, "invalid_request: file field required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		if header.Size > maxAssetSize {
			middleware.WriteError(w, "file_too_large: max 10 MB", http.StatusRequestEntityTooLarge)
			return
		}

		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		assetID, err := token.RandomString(12)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		r2Key := fmt.Sprintf("assets/%s/%s", claims.Sub, assetID)
		thumbnailR2Key := ""

		bucket, err := r2.NewBucket("LATEX_BUCKET")
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		_, err = bucket.Put(r2Key, file, &r2.PutOptions{
			HTTPMetadata: r2.HTTPMetadata{ContentType: mimeType},
		})
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}

		thumbnail, thumbnailHeader, thumbnailErr := r.FormFile("thumbnail")
		if thumbnailErr == nil {
			defer thumbnail.Close()
			if thumbnailHeader.Size <= maxThumbnailSize {
				thumbnailMimeType := thumbnailHeader.Header.Get("Content-Type")
				if thumbnailMimeType == "" {
					thumbnailMimeType = "image/webp"
				}
				thumbnailR2Key = fmt.Sprintf("assets/%s/%s.preview", claims.Sub, assetID)
				_, err = bucket.Put(thumbnailR2Key, thumbnail, &r2.PutOptions{
					HTTPMetadata: r2.HTTPMetadata{ContentType: thumbnailMimeType},
				})
				if err != nil {
					thumbnailR2Key = ""
				}
			}
		}

		asset, err := s.CreateUserAsset(r.Context(), assetID, claims.Sub, header.Filename, r2Key, thumbnailR2Key, mimeType, header.Size)
		if err != nil {
			_ = bucket.Delete(r2Key)
			if thumbnailR2Key != "" {
				_ = bucket.Delete(thumbnailR2Key)
			}
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(asset)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// AssetDetail handles GET /api/assets/{id}/content, GET /api/assets/{id}/preview, and DELETE /api/assets/{id}.
func AssetDetail(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	assetID := parts[3]
	var action string
	if len(parts) >= 5 {
		action = parts[4]
	}

	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	switch r.Method {
	case http.MethodGet:
		if action != "content" && action != "preview" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		asset, err := s.GetUserAsset(r.Context(), claims.Sub, assetID)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		if asset == nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		r2Key := asset.R2Key
		contentType := asset.MimeType
		if action == "preview" {
			if asset.ThumbnailR2Key == "" {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			r2Key = asset.ThumbnailR2Key
			contentType = "image/webp"
		}

		cacheURL := fmt.Sprintf("https://cache.internal/assets/%s/%s/%s", claims.Sub, assetID, action)
		if data := cfCacheGet(cacheURL); len(data) > 0 {
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Cache-Control", "private, max-age=86400")
			w.Header().Set("X-Cache", "CF-HIT")
			w.Write(data)
			return
		}

		bucket, err := r2.NewBucket("LATEX_BUCKET")
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		obj, err := bucket.Get(r2Key)
		if err != nil || obj == nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		data, err := io.ReadAll(obj.Body)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		cfCachePutWithContentType(cacheURL, data, contentType)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "private, max-age=86400")
		w.Header().Set("X-Cache", "MISS")
		w.Write(data)

	case http.MethodDelete:
		asset, err := s.GetUserAsset(r.Context(), claims.Sub, assetID)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		if asset == nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		bucket, err := r2.NewBucket("LATEX_BUCKET")
		if err == nil {
			_ = bucket.Delete(asset.R2Key)
			if asset.ThumbnailR2Key != "" {
				_ = bucket.Delete(asset.ThumbnailR2Key)
			}
		}

		if err := s.DeleteUserAsset(r.Context(), claims.Sub, assetID); err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
