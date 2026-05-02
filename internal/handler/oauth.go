package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/syumai/workers/cloudflare"

	"github.com/puemmth/sir-backend/internal/middleware"
	"github.com/puemmth/sir-backend/internal/store"
	"github.com/puemmth/sir-backend/internal/token"
)

// Authorize handles GET /oauth/authorize (login form) and POST /oauth/authorize (submit).
func Authorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		showLoginForm(w, r)
	case http.MethodPost:
		processAuthorize(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func showLoginForm(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	responseType := q.Get("response_type")
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	scope := q.Get("scope")

	if responseType != "code" {
		middleware.WriteError(w, "unsupported_response_type", http.StatusBadRequest)
		return
	}
	if clientID == "" || redirectURI == "" {
		middleware.WriteError(w, "invalid_request: missing client_id or redirect_uri", http.StatusBadRequest)
		return
	}
	// RFC 8252 §8.3: only loopback redirect URIs are permitted
	if !isLoopbackURI(redirectURI) {
		middleware.WriteError(w, "invalid_request: redirect_uri must be a loopback address (RFC 8252 §8.3)", http.StatusBadRequest)
		return
	}

	s, err := store.Open()
	if err != nil {
		middleware.WriteError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	client, err := s.GetClientByID(r.Context(), clientID)
	if err != nil || client == nil {
		middleware.WriteError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	if scope == "" {
		scope = "openid"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, loginFormHTML,
		htmlEscape(client.Name),
		htmlEscape(client.Name),
		htmlEscape(clientID),
		htmlEscape(redirectURI),
		htmlEscape(state),
		htmlEscape(scope),
	)
}

func processAuthorize(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		middleware.WriteError(w, "invalid_request", http.StatusBadRequest)
		return
	}

	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	state := r.FormValue("state")
	scope := r.FormValue("scope")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if !isLoopbackURI(redirectURI) {
		middleware.WriteError(w, "invalid_request: invalid redirect_uri", http.StatusBadRequest)
		return
	}

	s, err := store.Open()
	if err != nil {
		redirectWithError(w, r, redirectURI, state, "server_error")
		return
	}
	defer s.Close()

	client, err := s.GetClientByID(r.Context(), clientID)
	if err != nil || client == nil {
		redirectWithError(w, r, redirectURI, state, "invalid_client")
		return
	}

	user, err := s.GetUserByEmail(r.Context(), email)
	if err != nil || user == nil || !token.VerifyPassword(password, user.PasswordHash, user.Salt) {
		redirectWithError(w, r, redirectURI, state, "access_denied")
		return
	}

	if scope == "" {
		scope = "openid"
	}

	ac, err := s.CreateAuthCode(r.Context(), clientID, user.ID, redirectURI, scope)
	if err != nil {
		redirectWithError(w, r, redirectURI, state, "server_error")
		return
	}

	callbackURL := redirectURI + "?code=" + url.QueryEscape(ac.Code)
	if state != "" {
		callbackURL += "&state=" + url.QueryEscape(state)
	}
	http.Redirect(w, r, callbackURL, http.StatusFound)
}

// Token handles POST /oauth/token
func Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		middleware.WriteOAuthError(w, "invalid_request", http.StatusBadRequest)
		return
	}
	switch r.FormValue("grant_type") {
	case "authorization_code":
		exchangeAuthCode(w, r)
	case "refresh_token":
		exchangeRefreshToken(w, r)
	default:
		middleware.WriteOAuthError(w, "unsupported_grant_type", http.StatusBadRequest)
	}
}

func exchangeAuthCode(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if code == "" || redirectURI == "" || clientID == "" || clientSecret == "" {
		middleware.WriteOAuthError(w, "invalid_request", http.StatusBadRequest)
		return
	}

	s, err := store.Open()
	if err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	client, err := s.GetClientByID(r.Context(), clientID)
	if err != nil || client == nil || client.ClientSecret != clientSecret {
		middleware.WriteOAuthError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	ac, err := s.ConsumeAuthCode(r.Context(), code)
	if err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}
	if ac == nil || ac.Used || time.Now().Unix() > ac.ExpiresAt || ac.ClientID != clientID {
		middleware.WriteOAuthError(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	// RFC 8252 §7.3: match loopback URI ignoring port
	if !loopbackURIMatches(ac.RedirectURI, redirectURI) {
		middleware.WriteOAuthError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	user, err := s.GetUserByID(r.Context(), ac.UserID)
	if err != nil || user == nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	accessToken, err := token.GenerateAccessToken(user.ID, user.Email, user.Role, ac.Scope, cloudflare.Getenv("JWT_SECRET"))
	if err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	rt, err := s.CreateRefreshToken(r.Context(), user.ID, clientID, ac.Scope)
	if err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	writeTokenResponse(w, accessToken, rt.Token, ac.Scope)
}

func exchangeRefreshToken(w http.ResponseWriter, r *http.Request) {
	rawToken := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if rawToken == "" || clientID == "" || clientSecret == "" {
		middleware.WriteOAuthError(w, "invalid_request", http.StatusBadRequest)
		return
	}

	s, err := store.Open()
	if err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	client, err := s.GetClientByID(r.Context(), clientID)
	if err != nil || client == nil || client.ClientSecret != clientSecret {
		middleware.WriteOAuthError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	rt, err := s.GetRefreshToken(r.Context(), rawToken)
	if err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}
	if rt == nil || rt.Revoked || rt.ClientID != clientID || time.Now().Unix() > rt.ExpiresAt {
		middleware.WriteOAuthError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	user, err := s.GetUserByID(r.Context(), rt.UserID)
	if err != nil || user == nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	// Rotate: revoke old before issuing new
	if err := s.RevokeRefreshToken(r.Context(), rawToken); err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	accessToken, err := token.GenerateAccessToken(user.ID, user.Email, user.Role, rt.Scope, cloudflare.Getenv("JWT_SECRET"))
	if err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	newRT, err := s.CreateRefreshToken(r.Context(), user.ID, clientID, rt.Scope)
	if err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	writeTokenResponse(w, accessToken, newRT.Token, rt.Scope)
}

// Revoke handles POST /oauth/revoke (RFC 7009)
func Revoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil || r.FormValue("token") == "" {
		middleware.WriteOAuthError(w, "invalid_request", http.StatusBadRequest)
		return
	}

	s, err := store.Open()
	if err != nil {
		middleware.WriteOAuthError(w, "server_error", http.StatusInternalServerError)
		return
	}
	defer s.Close()

	// RFC 7009: always return 200, even when token was not found
	_ = s.RevokeRefreshToken(r.Context(), r.FormValue("token"))
	w.WriteHeader(http.StatusOK)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// isLoopbackURI returns true when the URI targets a loopback interface (RFC 8252 §8.3).
func isLoopbackURI(rawURI string) bool {
	u, err := url.Parse(rawURI)
	if err != nil || u.Scheme != "http" {
		return false
	}
	h := u.Hostname()
	return h == "127.0.0.1" || h == "localhost" || h == "::1"
}

// loopbackURIMatches compares two loopback URIs ignoring the port number (RFC 8252 §7.3).
func loopbackURIMatches(registered, request string) bool {
	reg, err1 := url.Parse(registered)
	req, err2 := url.Parse(request)
	if err1 != nil || err2 != nil {
		return false
	}
	return reg.Scheme == req.Scheme &&
		strings.EqualFold(reg.Hostname(), req.Hostname()) &&
		reg.Path == req.Path
}

func redirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, state, errCode string) {
	u := redirectURI + "?error=" + url.QueryEscape(errCode)
	if state != "" {
		u += "&state=" + url.QueryEscape(state)
	}
	http.Redirect(w, r, u, http.StatusFound)
}

func writeTokenResponse(w http.ResponseWriter, accessToken, refreshToken, scope string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(token.AccessTokenTTL.Seconds()),
		"refresh_token": refreshToken,
		"scope":         scope,
	})
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// loginFormHTML — format args: appName, appName, clientID, redirectURI, state, scope
const loginFormHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Sign in — %s</title>
  <style>
    *{box-sizing:border-box;margin:0;padding:0}
    body{font-family:system-ui,sans-serif;background:#f4f4f5;display:flex;align-items:center;justify-content:center;min-height:100vh}
    .card{background:#fff;border-radius:10px;box-shadow:0 2px 16px rgba(0,0,0,.1);padding:2rem;width:360px}
    h1{font-size:1.15rem;margin-bottom:1.5rem;color:#111}
    label{display:block;font-size:.82rem;color:#555;margin:.9rem 0 .25rem}
    input{width:100%%;padding:.6rem .75rem;border:1px solid #d1d5db;border-radius:6px;font-size:.95rem}
    input:focus{outline:2px solid #2563eb;border-color:transparent}
    button{margin-top:1.5rem;width:100%%;padding:.7rem;background:#2563eb;color:#fff;border:none;border-radius:6px;font-size:1rem;cursor:pointer;font-weight:500}
    button:hover{background:#1d4ed8}
  </style>
</head>
<body>
  <div class="card">
    <h1>Sign in to %s</h1>
    <form method="POST" action="/oauth/authorize">
      <input type="hidden" name="client_id"    value="%s">
      <input type="hidden" name="redirect_uri" value="%s">
      <input type="hidden" name="state"        value="%s">
      <input type="hidden" name="scope"        value="%s">
      <label>Email</label>
      <input type="email"    name="email"    required autofocus  autocomplete="email">
      <label>Password</label>
      <input type="password" name="password" required            autocomplete="current-password">
      <button type="submit">Sign In</button>
    </form>
  </div>
</body>
</html>`
