package main

import (
	"encoding/json"
	"net/http"

	"github.com/syumai/workers"

	"github.com/sir-labs/sir-api/internal/handler"
	"github.com/sir-labs/sir-api/internal/middleware"
)

func main() {
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/docs/openapi.json", handler.DocsJSON)
	mux.HandleFunc("/api/docs", handler.DocsUI)

	// OAuth 2.0 Authorization Code Flow (Loopback Interface Redirection — RFC 8252)
	mux.HandleFunc("/oauth/authorize", handler.Authorize)
	mux.HandleFunc("/oauth/token", handler.Token)
	mux.HandleFunc("/oauth/revoke", handler.Revoke)

	// Initial setup: creates first admin user + default client (runs once)
	mux.HandleFunc("/setup", handler.Setup)

	// Public: self-registration
	mux.HandleFunc("/register", handler.Register)

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

	mux.Handle("/api/compile", middleware.Chain(
		http.HandlerFunc(handler.Compile),
		middleware.AuthMiddleware,
	))

	mux.Handle("/api/latex-files", middleware.Chain(
		http.HandlerFunc(handler.LatexFiles),
		middleware.AuthMiddleware,
	))

	mux.Handle("/api/latex-files/", middleware.Chain(
		http.HandlerFunc(handler.LatexFileDetail),
		middleware.AuthMiddleware,
	))

	mux.Handle("/api/assets", middleware.Chain(
		http.HandlerFunc(handler.Assets),
		middleware.AuthMiddleware,
	))

	mux.Handle("/api/assets/", middleware.Chain(
		http.HandlerFunc(handler.AssetDetail),
		middleware.AuthMiddleware,
	))

	// Protected: any authenticated user
	mux.Handle("/api/users/", middleware.Chain(
		http.HandlerFunc(handler.GetUser),
		middleware.AuthMiddleware,
	))

	// Protected: any authenticated user (list + create)
	mux.Handle("/api/admin/users", middleware.Chain(
		http.HandlerFunc(handler.AdminUsers),
		middleware.AuthMiddleware,
	))

	mux.Handle("/api/admin/users/", middleware.Chain(
		http.HandlerFunc(handler.AdminUserDetail),
		middleware.AuthMiddleware,
	))

	mux.Handle("/api/admin/logs", middleware.Chain(
		http.HandlerFunc(handler.AdminLogs),
		middleware.AuthMiddleware,
	))

	workers.Serve(middleware.CORSMiddleware(mux))
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
