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

	// Protected: any authenticated user
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

	workers.Serve(middleware.CORSMiddleware(mux))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"name":    "sir-api",
		"version": "2.0.0",
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
