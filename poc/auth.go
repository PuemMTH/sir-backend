package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/skratchdot/open-golang/open"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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
