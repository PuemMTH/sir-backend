package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/puemmth/sir-backend/internal/middleware"
	"github.com/puemmth/sir-backend/internal/model"
	"github.com/puemmth/sir-backend/internal/store"
	"github.com/puemmth/sir-backend/internal/token"
)

// Me returns the authenticated user's profile.
func Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":    claims.Sub,
		"email": claims.Email,
		"role":  claims.Role,
		"scope": claims.Scope,
	})
}

// AdminUsers handles GET /api/admin/users and POST /api/admin/users.
func AdminUsers(w http.ResponseWriter, r *http.Request) {
	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	switch r.Method {
	case http.MethodGet:
		users, err := s.ListUsers(r.Context())
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		type userView struct {
			ID        string `json:"id"`
			Email     string `json:"email"`
			Role      string `json:"role"`
			CreatedAt int64  `json:"created_at"`
		}
		out := make([]userView, len(users))
		for i, u := range users {
			out[i] = userView{ID: u.ID, Email: u.Email, Role: u.Role, CreatedAt: u.CreatedAt}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)

	case http.MethodPost:
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
			middleware.WriteError(w, "invalid_request: email and password required", http.StatusBadRequest)
			return
		}
		if req.Role != "admin" && req.Role != "user" {
			req.Role = "user"
		}

		hash, salt, err := token.HashPassword(req.Password)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}
		id, err := token.RandomString(16)
		if err != nil {
			middleware.WriteError(w, "server_error", http.StatusInternalServerError)
			return
		}

		u := model.User{
			ID:           id,
			Email:        req.Email,
			PasswordHash: hash,
			Salt:         salt,
			Role:         req.Role,
			CreatedAt:    time.Now().Unix(),
		}
		if err := s.CreateUser(r.Context(), u); err != nil {
			middleware.WriteError(w, "conflict: email already exists", http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id":    u.ID,
			"email": u.Email,
			"role":  u.Role,
		})

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
