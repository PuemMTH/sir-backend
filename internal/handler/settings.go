package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sir-labs/sir-api/internal/middleware"
	"github.com/sir-labs/sir-api/internal/store"
)

// Settings handles GET and PUT /api/admin/settings (admin only).
func Settings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getSettings(w, r)
	case http.MethodPut:
		putSettings(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func getSettings(w http.ResponseWriter, r *http.Request) {
	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	settings, err := s.ListSettings(context.Background())
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func putSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" {
		middleware.WriteError(w, "invalid_request: key and value required", http.StatusBadRequest)
		return
	}

	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	if err := s.UpsertSetting(context.Background(), req.Key, req.Value); err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": req.Key, "value": req.Value})
}
