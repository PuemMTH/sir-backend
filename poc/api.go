package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type result struct {
	label  string
	status int
	body   string
}

func fetchData(accessToken, refreshToken string) tea.Cmd {
	return func() tea.Msg {
		var results []result

		// ── GET /api/me ──────────────────────────────────────────────────────
		status, body, _ := doRequest("GET", "/api/me", accessToken, nil)
		results = append(results, result{"GET /api/me", status, prettyJSON(body)})

		// extract own user ID for the next call
		var me struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(body), &me)

		// ── GET /api/users/{id} ───────────────────────────────────────────────
		if me.ID != "" {
			status, body, _ = doRequest("GET", "/api/users/"+me.ID, accessToken, nil)
			results = append(results, result{"GET /api/users/" + me.ID, status, prettyJSON(body)})
		}

		// ── GET /api/admin/users ──────────────────────────────────────────────
		status, body, _ = doRequest("GET", "/api/admin/users", accessToken, nil)
		results = append(results, result{"GET /api/admin/users", status, prettyJSON(body)})

		// ── POST /api/admin/users ─────────────────────────────────────────────
		demoEmail := fmt.Sprintf("demo-%d@example.com", time.Now().Unix())
		payload, _ := json.Marshal(map[string]string{
			"email":    demoEmail,
			"password": "demo-pass",
			"role":     "user",
		})
		status, body, _ = doRequest("POST", "/api/admin/users", accessToken, bytes.NewReader(payload))
		results = append(results, result{"POST /api/admin/users", status, prettyJSON(body)})

		// ── GET /api/notes ────────────────────────────────────────────────────
		status, body, _ = doRequest("GET", "/api/notes", accessToken, nil)
		results = append(results, result{"GET /api/notes", status, prettyJSON(body)})

		// ── POST /api/notes ───────────────────────────────────────────────────
		notePayload, _ := json.Marshal(map[string]string{
			"title":   "Demo Note",
			"content": "Created by the POC TUI client.",
		})
		status, body, _ = doRequest("POST", "/api/notes", accessToken, bytes.NewReader(notePayload))
		results = append(results, result{"POST /api/notes", status, prettyJSON(body)})

		// extract created note ID
		var note struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(body), &note)

		// ── GET /api/notes/{id} ───────────────────────────────────────────────
		if note.ID != "" {
			status, body, _ = doRequest("GET", "/api/notes/"+note.ID, accessToken, nil)
			results = append(results, result{"GET /api/notes/" + note.ID, status, prettyJSON(body)})

			// ── DELETE /api/notes/{id} ────────────────────────────────────────
			status, body, _ = doRequest("DELETE", "/api/notes/"+note.ID, accessToken, nil)
			deleteBody := body
			if status == http.StatusNoContent {
				deleteBody = `"note deleted"`
			}
			results = append(results, result{"DELETE /api/notes/" + note.ID, status, deleteBody})
		}

		// ── POST /oauth/revoke ────────────────────────────────────────────────
		if refreshToken != "" {
			revokePayload, _ := json.Marshal(map[string]string{"token": refreshToken})
			status, body, _ = doRequest("POST", "/oauth/revoke", "", bytes.NewReader(revokePayload))
			revokeBody := body
			if status == http.StatusOK && body == "" {
				revokeBody = `"token revoked"`
			}
			results = append(results, result{"POST /oauth/revoke", status, revokeBody})
		}

		return dataFetchedMsg{results: results}
	}
}

// doRequest performs an HTTP request, returning status code and body.
func doRequest(method, path, token string, body io.Reader) (int, string, error) {
	req, err := http.NewRequest(method, BaseURL+path, body)
	if err != nil {
		return 0, "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw), nil
}

func prettyJSON(raw string) string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	out, _ := json.MarshalIndent(v, "", "  ")
	return string(out)
}
