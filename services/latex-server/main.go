package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const fontsDir = "/usr/local/share/fonts/api"

var allowedFontExts = map[string]bool{
	".ttf":   true,
	".otf":   true,
	".woff":  true,
	".woff2": true,
}

var allowedEngines = map[string]bool{
	"lualatex": true,
	"pdflatex": true,
	"xelatex":  true,
}

type fileEntry struct {
	Name    string `json:"name"`    // original filename (e.g. "photo.jpg")
	Content string `json:"content"` // base64-encoded file content
}

type compileRequest struct {
	Source string      `json:"source"`
	Engine string      `json:"engine"`
	Files  []fileEntry `json:"files"` // optional assets written to the temp dir
}

type errorResponse struct {
	Error string `json:"error"`
	Log   string `json:"log,omitempty"`
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func compileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var req compileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}

	req.Source = strings.TrimSpace(req.Source)
	if req.Source == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "source must not be empty"})
		return
	}
	if req.Engine == "" {
		req.Engine = "lualatex"
	}
	if !allowedEngines[req.Engine] {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "unsupported engine: " + req.Engine})
		return
	}

	tmpDir, err := os.MkdirTemp("", "latex-*")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create temp dir"})
		return
	}
	defer os.RemoveAll(tmpDir)

	texPath := filepath.Join(tmpDir, "document.tex")
	pdfPath := filepath.Join(tmpDir, "document.pdf")
	logPath := filepath.Join(tmpDir, "document.log")

	if err := os.WriteFile(texPath, []byte(req.Source), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to write source"})
		return
	}

	// Write uploaded assets (images, etc.) into the same temp dir so LaTeX can reference them by name.
	for _, f := range req.Files {
		name := filepath.Base(f.Name)
		if name == "" || name == "." {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(f.Content)
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(tmpDir, name), data, 0o644)
	}

	args := []string{
		"-interaction=nonstopmode",
		"-halt-on-error",
		"-output-directory", tmpDir,
		texPath,
	}

	// Run twice so cross-references, TOC, and labels resolve correctly.
	var lastStderr []byte
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		cmd := exec.CommandContext(ctx, req.Engine, args...)
		lastStderr, _ = cmd.CombinedOutput()
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "compilation timed out"})
			return
		}
	}

	if _, err := os.Stat(pdfPath); err == nil {
		pdf, err := os.ReadFile(pdfPath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to read PDF"})
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `inline; filename="document.pdf"`)
		w.WriteHeader(http.StatusOK)
		w.Write(pdf)
		return
	}

	logContent := string(lastStderr)
	if data, err := os.ReadFile(logPath); err == nil {
		logContent = string(data)
	}

	writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
		Error: "compilation failed",
		Log:   logContent,
	})
}

func uploadFontHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "request too large or invalid multipart"})
		return
	}

	file, header, err := r.FormFile("font")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "font file required (field: font)"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedFontExts[ext] {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "unsupported font type: " + ext})
		return
	}

	name := filepath.Base(header.Filename)

	if err := os.MkdirAll(fontsDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create fonts dir"})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to read font"})
		return
	}

	if err := os.WriteFile(filepath.Join(fontsDir, name), data, 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to save font"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	exec.CommandContext(ctx, "fc-cache", "-f", fontsDir).Run()

	writeJSON(w, http.StatusOK, map[string]string{"name": name})
}

func listFontsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	entries, err := os.ReadDir(fontsDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []string{})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list fonts"})
		return
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	writeJSON(w, http.StatusOK, names)
}

func deleteFontHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	name := filepath.Base(r.PathValue("name"))
	if name == "" || name == "." {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "font name required"})
		return
	}

	ext := strings.ToLower(filepath.Ext(name))
	if !allowedFontExts[ext] {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid font name"})
		return
	}

	path := filepath.Join(fontsDir, name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "font not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to delete font"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	exec.CommandContext(ctx, "fc-cache", "-f", fontsDir).Run()

	writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", cors(healthHandler))
	mux.HandleFunc("/compile", cors(compileHandler))
	mux.HandleFunc("/fonts", cors(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			uploadFontHandler(w, r)
		case http.MethodGet:
			listFontsHandler(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		}
	}))
	mux.HandleFunc("/fonts/{name}", cors(deleteFontHandler))

	log.Println("latex-server listening on :80")
	if err := http.ListenAndServe(":80", mux); err != nil {
		log.Fatal(err)
	}
}
