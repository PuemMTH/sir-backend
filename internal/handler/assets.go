package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/syumai/workers/cloudflare/r2"

	"github.com/puemmth/sir-backend/internal/middleware"
	"github.com/puemmth/sir-backend/internal/store"
	"github.com/puemmth/sir-backend/internal/token"
)

const maxAssetSize = 10 << 20 // 10 MB

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

		asset, err := s.CreateUserAsset(r.Context(), assetID, claims.Sub, header.Filename, r2Key, mimeType, header.Size)
		if err != nil {
			_ = bucket.Delete(r2Key)
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

// AssetDetail handles DELETE /api/assets/{id}.
func AssetDetail(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	assetID := parts[3]

	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	switch r.Method {
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
