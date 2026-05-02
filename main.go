package main

import (
	"encoding/json"
	"net/http"

	"github.com/syumai/workers"

	"github.com/puemmth/sir-backend/internal/handler"
	"github.com/puemmth/sir-backend/internal/middleware"
)

func main() {
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/health", handleHealth)

	// OAuth 2.0 Authorization Code Flow (Loopback Interface Redirection — RFC 8252)
	mux.HandleFunc("/oauth/authorize", handler.Authorize)
	mux.HandleFunc("/oauth/token", handler.Token)
	mux.HandleFunc("/oauth/revoke", handler.Revoke)

	// Initial setup: creates first admin user + default client (runs once)
	mux.HandleFunc("/setup", handler.Setup)

	// Protected: any authenticated user
	mux.Handle("/api/me", middleware.Chain(
		http.HandlerFunc(handler.Me),
		middleware.AuthMiddleware,
	))

	mux.Handle("/api/notes", middleware.Chain(
		http.HandlerFunc(handler.Notes),
		middleware.AuthMiddleware,
	))

	mux.Handle("/api/notes/", middleware.Chain(
		http.HandlerFunc(handler.NoteDetail),
		middleware.AuthMiddleware,
	))

	// Protected: admin only
	mux.Handle("/api/admin/users", middleware.Chain(
		http.HandlerFunc(handler.AdminUsers),
		middleware.AuthMiddleware,
		middleware.RequireRole("admin"),
	))

	workers.Serve(mux)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"name":    "sir-backend",
		"version": "1.1.0",
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
