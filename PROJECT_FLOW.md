# sir-backend — Project Flow

## Purpose
This document summarizes the request, auth, data, and compile flows for the sir-backend Cloudflare Worker (Go/WASM).

## High-level architecture
- Runtime: Cloudflare Workers via github.com/syumai/workers (Go → WASM).
- HTTP entry: main.go registers routes on http.ServeMux and uses workers.Serve(middleware.CORSMiddleware(mux)).
- Storage: D1 (SQL) accessed via GORM in internal/store. R2 used for large file content (LaTeX sources, PDFs).
- Auth: OAuth2 Authorization Code + JWT access tokens and refresh tokens. JWT signing secret provided via env var JWT_SECRET.

## Route categories
- Public: /, /health, /api/docs, /api/docs/openapi.json, /oauth/*, /register, /setup
- Protected (AuthMiddleware): /api/me, /api/notes*, /api/compile, /api/latex-files*, /api/users/:id, /api/admin/*
- Admin-only: routes wrapped with RequireRole("admin") where applicable (admin bypasses role checks).

## Request handling flow (general)
1. Incoming request hits main.go mux routing.
2. If route wrapped with middleware.Chain(..., middleware.AuthMiddleware, ...), the AuthMiddleware runs first.
3. AuthMiddleware checks Authorization: Bearer <token>, validates JWT with token.ValidateAccessToken(JWT_SECRET), injects claims into context.
4. Handler reads claims via middleware.ClaimsFromCtx(r.Context()) and performs logic (db calls, R2 operations, compile proxying, etc.).
5. Store.Open() is used per request to obtain a Store (GORM over D1 connector). Remember to defer s.Close().

## OAuth2 (authorization code) flow
1. GET /oauth/authorize — show login form.
2. POST /oauth/authorize — authenticate user credentials; on success create an AuthCode via store.CreateAuthCode and redirect to redirect_uri?code=<code>&state=<state>.
3. Client exchanges code at POST /oauth/token with grant_type=authorization_code. Token handler calls store.ConsumeAuthCode, creates RefreshToken via store.CreateRefreshToken, and issues an access JWT (expires_in ~3600s) plus refresh token.
4. Refresh flow: client posts grant_type=refresh_token to /oauth/token, server validates stored RefreshToken and issues new access_token (and optionally new refresh token).
5. POST /oauth/revoke revokes the refresh token (store.RevokeRefreshToken).

Notes: AuthCode and RefreshToken models generate tokens and expiries in BeforeCreate hooks (internal/model). Auth codes are single-use.

## JWT validation and roles
- token.ValidateAccessToken(raw, JWT_SECRET) validates signature and expiry, returning token.Claims (sub, email, role, scope, iat, exp).
- AuthMiddleware injects claims into context; handlers use middleware.ClaimsFromCtx.
- RequireRole(role) enforces role or admin bypass.

## Data storage patterns
- D1 (SQL via d1.OpenConnector("DB")) is the canonical metadata store: users, oauth_clients, auth_codes, refresh_tokens, notes, latex_file metadata, pdf_cache, system_logs.
- R2 is used for storing LaTeX file contents and compiled PDFs; handlers construct r2 keys (e.g., latex/<user_id>/<file_id>.tex) and read/write via Cloudflare APIs.

## LaTeX compile + caching flow
1. POST /api/compile with {source, engine}.
2. Backend computes MD5(engine + ":" + source) → sourceHash.
3. Check store.GetPDFCache(sourceHash):
   - If found: fetch PDF from R2 using the stored r2_key and return with header X-Cache: HIT.
   - If not found: proxy request to upstream compile server, on success store PDF in R2, call store.CreatePDFCache(sourceHash, r2_key, engine), return PDF with X-Cache: MISS.
4. Cache insert ignores duplicate-unique constraint errors (safe race handling).

## LaTeX file lifecycle
- POST /api/latex-files: create metadata row in D1 (store.CreateLatexFile) and upload content to R2 under r2_key.
- GET /api/latex-files/:id: read metadata from D1 and content from R2.
- PUT /api/latex-files/:id: update metadata in D1 (store.UpdateLatexFile) and overwrite R2 object if content provided.
- DELETE /api/latex-files/:id: best-effort delete in R2 and delete D1 row (store.DeleteLatexFile).

## Notes and user resources
- Notes are small private documents stored fully in D1 (store.CreateNote, ListNotes, GetNote, DeleteNote).
- Users: CreateUser, GetUserByEmail/ID, ListUsers, UpdateUser, DeleteUser through store.* functions.

## Admin setup
- /setup endpoint runs one-time initialization creating first admin user and default OAuth client. Secured by a SETUP_SECRET query param.
- Admin endpoints for user management and system logs exist under /api/admin/*.

## Developer reminders
- Build target: GOOS=js GOARCH=wasm go build ./...
- Always open/close Store per request: s, err := store.Open(); defer s.Close().
- Add new routes in four places: handler file, main.go routing, docs.go openapi JSON, and user.md route summary.

## Sequence (OAuth + token exchange) — condensed
1. User -> /oauth/authorize (login) -> server creates AuthCode in D1.
2. Client -> /oauth/token (authorization_code) -> server consumes code, creates RefreshToken, returns access_token (JWT) + refresh_token.
3. Client -> protected endpoints with Authorization: Bearer <access_token>.
4. When access_token expires, client uses refresh_token at /oauth/token to get new access_token.

---
Generated from main.go, handlers, middleware, store, model, and user.md.

