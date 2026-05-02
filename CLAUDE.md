# sir-backend

Cloudflare Worker written in Go (WASM via `github.com/syumai/workers`). Build target is always `GOOS=js GOARCH=wasm`.

## Project layout

```
main.go                          route registration
internal/handler/                one file per resource group
  user.go                        Register, Me, GetUser, AdminUsers
  note.go                        Notes, NoteDetail
  oauth.go                       Authorize, Token, Revoke
  setup.go                       Setup (one-time init)
  docs.go                        DocsUI, DocsJSON + embedded OpenAPI spec
internal/middleware/auth.go      AuthMiddleware, RequireRole, Chain
internal/store/store.go          all DB operations (GORM + D1)
internal/model/model.go          structs: User, Note, OAuthClient, AuthCode, RefreshToken
internal/token/                  JWT, password hashing, random strings
poc/                             standalone TUI OAuth demo (separate go.mod)
user.md                          human-readable API reference
```

## When you add, remove, or change an API route

Update **all four** of these in the same change:

### 1. `internal/handler/<resource>.go`
- Add the handler function.
- Use `middleware.WriteError(w, "message", httpCode)` for error responses.
- Use `middleware.ClaimsFromCtx(r.Context())` to read JWT claims.

### 2. `main.go`
- Register the route on `mux`.
- Wrap with `middleware.Chain(...)` for any route that needs auth:
  ```go
  // any authenticated user
  mux.Handle("/api/foo", middleware.Chain(
      http.HandlerFunc(handler.Foo),
      middleware.AuthMiddleware,
  ))

  // admin only
  mux.Handle("/api/bar", middleware.Chain(
      http.HandlerFunc(handler.Bar),
      middleware.AuthMiddleware,
      middleware.RequireRole("admin"),
  ))
  ```
- Public routes use plain `mux.HandleFunc`.

### 3. `internal/handler/docs.go` — OpenAPI spec
- Add or update the path entry inside the `openapiJSON` const string.
- Follow the existing pattern: `tags`, `summary`, `security`, `requestBody`, `responses`.
- Add new reusable schemas to `components.schemas` when needed.
- Security options: `"security": [{"bearerAuth": []}]` for JWT-protected routes, omit for public.

### 4. `user.md` — human-readable reference
- Add or update the route section under the correct heading:
  - **Public Routes** — no auth
  - **Protected Routes (All Roles)** — JWT required
  - **Admin Routes** — JWT + admin role
- Update the **Route Summary** table at the bottom.

## Auth rules

| Middleware | Who can call |
|------------|--------------|
| none | anyone |
| `AuthMiddleware` | any valid JWT |
| `AuthMiddleware` + `RequireRole("admin")` | admin role only (admin bypasses all role checks) |

## Build

```bash
GOOS=js GOARCH=wasm go build ./...
```

Never run plain `go build` — it will fail because `syscall/js` is WASM-only.

## Database

D1 via GORM. All queries go through `internal/store/store.go`. Open and close a store per request:

```go
s, err := store.Open()
if err != nil { ... }
defer s.Close()
```
