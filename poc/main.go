package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/skratchdot/open-golang/open"
)

var (
	BaseURL      = getEnv("BASE_URL", "https://sir.puem.me")
	RedirectURI  = "http://127.0.0.1:8080/callback"
	ClientID     = getEnv("CLIENT_ID", "test-client")
	ClientSecret = getEnv("CLIENT_SECRET", "test-secret")
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

type state int

const (
	stateInit state = iota
	stateWaiting
	stateFetching
	stateDone
	stateError
)

type model struct {
	state       state
	err         error
	spinner     spinner.Model
	accessToken string
	meData      string
	notesData   string
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		state:   stateInit,
		spinner: s,
	}
}

type authSuccessMsg string
type dataFetchedMsg struct {
	me    string
	notes string
}
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if m.state == stateInit && msg.String() == "enter" {
			m.state = stateWaiting
			return m, tea.Batch(m.spinner.Tick, startAuthFlow())
		}
	case authSuccessMsg:
		m.state = stateFetching
		m.accessToken = string(msg)
		return m, fetchData(string(msg))
	case dataFetchedMsg:
		m.state = stateDone
		m.meData = msg.me
		m.notesData = msg.notes
		return m, tea.Quit
	case errMsg:
		m.state = stateError
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1)
	infoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("63"))
)

func (m model) View() string {
	s := "\n" + titleStyle.Render("OAuth 2.0 TUI Client") + "\n\n"

	switch m.state {
	case stateInit:
		s += "Press " + lipgloss.NewStyle().Bold(true).Render("Enter") + " to open your browser and authenticate.\n"
		s += "Press 'q' or 'ctrl+c' to quit.\n"
	case stateWaiting:
		s += m.spinner.View() + " Waiting for browser authorization (Check your browser)...\n"
	case stateFetching:
		s += m.spinner.View() + " Exchanging code & fetching protected data...\n"
	case stateDone:
		s += infoStyle.Render("✔ Authentication Successful!") + "\n\n"

		meBox := boxStyle.Render(fmt.Sprintf("User Info (/api/me):\n\n%s", lipgloss.NewStyle().Foreground(lipgloss.Color("228")).Render(m.meData)))
		notesBox := boxStyle.Render(fmt.Sprintf("Notes (/api/notes):\n\n%s", lipgloss.NewStyle().Foreground(lipgloss.Color("120")).Render(m.notesData)))

		s += lipgloss.JoinHorizontal(lipgloss.Top, meBox, "   ", notesBox) + "\n"
	case stateError:
		s += errorStyle.Render(fmt.Sprintf("✖ Error: %v", m.err)) + "\n"
	}

	return s + "\n"
}

func startAuthFlow() tea.Cmd {
	return func() tea.Msg {
		state := "random-state-123"
		authURL := fmt.Sprintf("%s/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s&state=%s&scope=openid",
			BaseURL, ClientID, url.QueryEscape(RedirectURI), state)

		if err := open.Run(authURL); err != nil {
			return errMsg{fmt.Errorf("failed to open browser: %v", err)}
		}

		codeChan := make(chan string)
		errChan := make(chan error)
		srv := &http.Server{Addr: ":8080"}

		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			receivedState := r.URL.Query().Get("state")
			errStr := r.URL.Query().Get("error")

			if errStr != "" {
				http.Error(w, "Authorization failed: "+errStr, http.StatusBadRequest)
				errChan <- fmt.Errorf("authorization failed: %s", errStr)
				return
			}
			if receivedState != state {
				http.Error(w, "Invalid state", http.StatusBadRequest)
				errChan <- fmt.Errorf("invalid state")
				return
			}
			if code == "" {
				http.Error(w, "Code not found", http.StatusBadRequest)
				errChan <- fmt.Errorf("code not found")
				return
			}

			fmt.Fprintf(w, "<h1>Authorization Successful</h1><p>You can close this window and return to the terminal.</p>")
			codeChan <- code
		})

		go func() {
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				errChan <- fmt.Errorf("server error: %v", err)
			}
		}()

		select {
		case code := <-codeChan:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			srv.Shutdown(ctx)

			// Exchange code for token
			tokenResp, err := exchangeCode(code)
			if err != nil {
				return errMsg{fmt.Errorf("failed to exchange code: %v", err)}
			}
			return authSuccessMsg(tokenResp.AccessToken)
		case err := <-errChan:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			srv.Shutdown(ctx)
			return errMsg{err}
		}
	}
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func exchangeCode(code string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", RedirectURI)
	data.Set("client_id", ClientID)
	data.Set("client_secret", ClientSecret)

	resp, err := http.PostForm(BaseURL+"/oauth/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad status: %s, body: %s", resp.Status, string(body))
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func fetchData(token string) tea.Cmd {
	return func() tea.Msg {
		me, err := fetchProtected(token, "/api/me")
		if err != nil {
			return errMsg{fmt.Errorf("failed to fetch /api/me: %v", err)}
		}

		var parsedMe any
		json.Unmarshal([]byte(me), &parsedMe)
		meFormatted, _ := json.MarshalIndent(parsedMe, "", "  ")

		notes, err := fetchProtected(token, "/api/notes")
		if err != nil {
			return errMsg{fmt.Errorf("failed to fetch /api/notes: %v", err)}
		}

		var parsedNotes any
		json.Unmarshal([]byte(notes), &parsedNotes)
		notesFormatted, _ := json.MarshalIndent(parsedNotes, "", "  ")

		return dataFetchedMsg{
			me:    string(meFormatted),
			notes: string(notesFormatted),
		}
	}
}

func fetchProtected(token, path string) (string, error) {
	req, _ := http.NewRequest("GET", BaseURL+path, nil)
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

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
