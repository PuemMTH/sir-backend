package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	BaseURL      = getEnv("BASE_URL", "https://sir.puem.me")
	RedirectURI  = "http://127.0.0.1:8080/callback"
	ClientID     = getEnv("CLIENT_ID", "test-client")
	ClientSecret = getEnv("CLIENT_SECRET", "test-secret")
)

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
