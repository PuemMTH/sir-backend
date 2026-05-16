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

// LatexFiles handles GET /api/latex-files (list) and POST /api/latex-files (create).
func LatexFiles(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	switch r.Method {
	case http.MethodGet:
		files, err := s.ListLatexFiles(r.Context(), claims.Sub)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)

	case http.MethodPost:
		var req struct {
			Name    string `json:"name"`
			Content string `json:"content"`
			Engine  string `json:"engine"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			middleware.WriteError(w, "invalid_request: name required", http.StatusBadRequest)
			return
		}
		if req.Engine == "" {
			req.Engine = "lualatex"
		}

		fileID, err := token.RandomString(12)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		r2Key := fmt.Sprintf("latex/%s/%s.tex", claims.Sub, fileID)

		bucket, err := r2.NewBucket("LATEX_BUCKET")
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		_, err = bucket.Put(r2Key, io.NopCloser(strings.NewReader(req.Content)), &r2.PutOptions{
			HTTPMetadata: r2.HTTPMetadata{ContentType: "text/plain; charset=utf-8"},
		})
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}

		file, err := s.CreateLatexFile(r.Context(), fileID, claims.Sub, req.Name, r2Key, req.Engine)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(file)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// LatexFileDetail handles GET, PUT, DELETE for /api/latex-files/{id}.
func LatexFileDetail(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	fileID := parts[3]

	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	switch r.Method {
	case http.MethodGet:
		file, err := s.GetLatexFile(r.Context(), claims.Sub, fileID)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		if file == nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		bucket, err := r2.NewBucket("LATEX_BUCKET")
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		obj, err := bucket.Get(file.R2Key)
		if err != nil || obj == nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		content, err := io.ReadAll(obj.Body)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":         file.ID,
			"user_id":    file.UserID,
			"name":       file.Name,
			"engine":     file.Engine,
			"content":    string(content),
			"created_at": file.CreatedAt,
			"updated_at": file.UpdatedAt,
		})

	case http.MethodPut:
		var req struct {
			Name    *string `json:"name"`
			Content *string `json:"content"`
			Engine  *string `json:"engine"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, "invalid_request", http.StatusBadRequest)
			return
		}

		file, err := s.GetLatexFile(r.Context(), claims.Sub, fileID)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		if file == nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		if req.Name != nil {
			file.Name = *req.Name
		}
		if req.Engine != nil {
			file.Engine = *req.Engine
		}
		if req.Content != nil {
			bucket, err := r2.NewBucket("LATEX_BUCKET")
			if err != nil {
				middleware.WriteError(w, "server_error", http.StatusInternalServerError)
				return
			}
			_, err = bucket.Put(file.R2Key, io.NopCloser(strings.NewReader(*req.Content)), &r2.PutOptions{
				HTTPMetadata: r2.HTTPMetadata{ContentType: "text/plain; charset=utf-8"},
			})
			if err != nil {
				middleware.WriteError(w, "server_error", http.StatusInternalServerError)
				return
			}
		}

		updated, err := s.UpdateLatexFile(r.Context(), file)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)

	case http.MethodDelete:
		file, err := s.GetLatexFile(r.Context(), claims.Sub, fileID)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		if file == nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		bucket, err := r2.NewBucket("LATEX_BUCKET")
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		_ = bucket.Delete(file.R2Key)

		if err := s.DeleteLatexFile(r.Context(), claims.Sub, fileID); err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
