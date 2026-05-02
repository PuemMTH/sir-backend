package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/puemmth/sir-backend/internal/middleware"
	"github.com/puemmth/sir-backend/internal/store"
)

// Notes handles GET /api/notes (list) and POST /api/notes (create).
func Notes(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	switch r.Method {
	case http.MethodGet:
		notes, err := s.ListNotes(r.Context(), claims.Sub)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notes)

	case http.MethodPost:
		var req struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
			middleware.WriteError(w, "invalid_request: title required", http.StatusBadRequest)
			return
		}

		note, err := s.CreateNote(r.Context(), claims.Sub, req.Title, req.Content)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(note)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// NoteDetail handles GET and DELETE for a specific note.
func NoteDetail(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	
	// Simple path parsing: /api/notes/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	noteID := parts[3]

	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	switch r.Method {
	case http.MethodGet:
		note, err := s.GetNote(r.Context(), claims.Sub, noteID)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		if note == nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(note)

	case http.MethodDelete:
		if err := s.DeleteNote(r.Context(), claims.Sub, noteID); err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
