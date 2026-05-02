package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	tea "github.com/charmbracelet/bubbletea"
)

func fetchData(token string) tea.Cmd {
	return func() tea.Msg {
		me, err := fetchProtected(token, "/api/me")
		if err != nil {
			return errMsg{fmt.Errorf("failed to fetch /api/me: %v", err)}
		}

		users, err := fetchProtected(token, "/api/admin/users")
		if err != nil {
			return errMsg{fmt.Errorf("failed to fetch /api/admin/users: %v", err)}
		}

		notes, err := fetchProtected(token, "/api/notes")
		if err != nil {
			return errMsg{fmt.Errorf("failed to fetch /api/notes: %v", err)}
		}

		return dataFetchedMsg{
			me:    prettyJSON(me),
			users: prettyJSON(users),
			notes: prettyJSON(notes),
		}
	}
}

func fetchProtected(token, path string) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, BaseURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status: %s, body: %s", resp.Status, string(body))
	}
	return string(body), nil
}

func prettyJSON(raw string) string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	out, _ := json.MarshalIndent(v, "", "  ")
	return string(out)
}
