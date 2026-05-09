# User API Reference

## Auth Tokens

| Type | TTL |
|------|-----|
| Access Token (JWT) | 1 hour |
| Refresh Token | 30 days |
| Authorization Code | 10 minutes |

All protected routes require:
```
Authorization: Bearer <access_token>
```

---

## Public Routes

### POST /register

Register a new user account. Returns the new user's ID.

**Request Body**

```json
{
  "email": "user@example.com",
  "password": "secret"
}
```

**Response** `201 Created`

```json
{
  "id": "abc123",
  "email": "user@example.com",
  "role": "user"
}
```

**Errors**

| Code | Reason |
|------|--------|
| 400 | Missing email or password |
| 409 | Email already exists |

---

### POST /setup

Initialize the first admin user and OAuth client. Requires `SETUP_SECRET` env var.

**Query Params**

| Param | Required | Description |
|-------|----------|-------------|
| `secret` | Yes | Must match `SETUP_SECRET` env var |

**Request Body**

```json
{
  "admin_email": "admin@example.com",
  "admin_password": "secret",
  "client_id": "my-client",
  "client_secret": "client-secret",
  "client_name": "Default Client"
}
```

**Response** `201 Created`

```json
{
  "admin_id": "abc123",
  "admin_email": "admin@example.com",
  "client_id": "my-client",
  "client_name": "Default Client"
}
```

**Errors**

| Code | Reason |
|------|--------|
| 401 | Missing or wrong `secret` |
| 409 | Users already exist |

---

### GET /oauth/authorize

Display the login form.

**Query Params**

| Param | Required | Description |
|-------|----------|-------------|
| `client_id` | Yes | OAuth client ID |
| `redirect_uri` | Yes | Loopback URI (RFC 8252) |
| `state` | No | CSRF state value |
| `scope` | No | Defaults to `openid` |

**Response** — HTML login form

---

### POST /oauth/authorize

Submit login credentials and receive an authorization code.

**Request Body** (form or JSON)

| Field | Required |
|-------|----------|
| `client_id` | Yes |
| `redirect_uri` | Yes |
| `email` | Yes |
| `password` | Yes |
| `state` | No |
| `scope` | No |

**Response** — Redirect to `redirect_uri?code=<code>&state=<state>`

**Errors** — Redirect to `redirect_uri?error=<error_code>`

---

### POST /oauth/token

Exchange an authorization code or refresh token for new tokens.

**Authorization Code Grant**

```json
{
  "grant_type": "authorization_code",
  "code": "<auth_code>",
  "redirect_uri": "http://localhost:3000/callback",
  "client_id": "my-client",
  "client_secret": "client-secret"
}
```

**Refresh Token Grant**

```json
{
  "grant_type": "refresh_token",
  "refresh_token": "<refresh_token>",
  "client_id": "my-client",
  "client_secret": "client-secret"
}
```

**Response** `200 OK`

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "<refresh_token>",
  "scope": "openid"
}
```

**Errors**

| Code | Reason |
|------|--------|
| 400 | Invalid grant type or missing fields |
| 401 | Bad client credentials or invalid/expired code |

---

### POST /oauth/revoke

Revoke a refresh token (RFC 7009). Always returns `200` even if token not found.

**Request Body**

```json
{
  "token": "<refresh_token>"
}
```

**Response** `200 OK` (no body)

---

## Protected Routes

> Require `Authorization: Bearer <access_token>`

### GET /api/me

Return the authenticated user's profile.

**Response** `200 OK`

```json
{
  "id": "abc123",
  "email": "user@example.com",
  "role": "user",
  "scope": "openid"
}
```

**Errors**

| Code | Reason |
|------|--------|
| 401 | Missing, invalid, or expired token |

---

## Protected Routes (All Roles)

> Require `Authorization: Bearer <access_token>` — any authenticated user

### GET /api/users/:id

Get a user by ID. A regular user may only fetch their own profile. An admin may fetch any user.

**URL Params**

| Param | Description |
|-------|-------------|
| `id` | User ID |

**Response** `200 OK`

```json
{
  "id": "abc123",
  "email": "user@example.com",
  "role": "user",
  "created_at": 1714000000
}
```

**Errors**

| Code | Reason |
|------|--------|
| 400 | Missing user ID |
| 401 | Invalid token |
| 403 | Requesting another user's profile without admin role |
| 404 | User not found |

---

### GET /api/admin/users

List all users ordered by creation date (newest first).

**Response** `200 OK`

```json
[
  {
    "id": "abc123",
    "email": "user@example.com",
    "role": "user",
    "created_at": 1714000000
  }
]
```

**Errors**

| Code | Reason |
|------|--------|
| 401 | Invalid token |

---

### POST /api/admin/users

Create a new user.

**Request Body**

```json
{
  "email": "newuser@example.com",
  "password": "secret",
  "role": "user"
}
```

| Field | Required | Values |
|-------|----------|--------|
| `email` | Yes | Must be unique |
| `password` | Yes | |
| `role` | No | `user` (default) or `admin` |

**Response** `201 Created`

```json
{
  "id": "xyz789",
  "email": "newuser@example.com",
  "role": "user"
}
```

**Errors**

| Code | Reason |
|------|--------|
| 400 | Missing email or password |
| 401 | Invalid token |
| 409 | Email already exists |

---

### POST /api/compile

Compile LaTeX source to PDF with automatic caching. The backend computes `MD5(engine + ":" + source)` and checks the `pdf_cache` table before calling the compile server.

- **Cache hit** (`X-Cache: HIT`) — PDF returned directly from R2, no compilation cost.
- **Cache miss** (`X-Cache: MISS`) — Proxied to the upstream compile server. On success the PDF is stored in R2 and the hash recorded in D1 for future requests.

**Request Body**

```json
{
  "source": "\\documentclass{article}...",
  "engine": "lualatex"
}
```

| Field | Required | Values |
|-------|----------|--------|
| `source` | Yes | Full LaTeX source |
| `engine` | No | `lualatex` (default), `pdflatex`, `xelatex` |

**Response** `200 OK` — `application/pdf` binary

| Header | Values |
|--------|--------|
| `X-Cache` | `HIT` or `MISS` |

**Error Response** `422 Unprocessable Entity`

```json
{
  "error": "compilation failed",
  "log": "...LaTeX log output..."
}
```

**Errors**

| Code | Reason |
|------|--------|
| 400 | Missing or empty `source` |
| 401 | Invalid token |
| 422 | LaTeX compilation error (log included) |
| 502 | Compile server unreachable |

---

### GET /api/latex-files

List all LaTeX files owned by the authenticated user, ordered by last updated (newest first).

**Response** `200 OK`

```json
[
  {
    "id": "abc123",
    "user_id": "xyz789",
    "name": "thesis.tex",
    "r2_key": "latex/xyz789/abc123.tex",
    "engine": "lualatex",
    "created_at": 1714000000,
    "updated_at": 1714003600
  }
]
```

**Errors**

| Code | Reason |
|------|--------|
| 401 | Invalid token |

---

### POST /api/latex-files

Create a new LaTeX file. Content is stored in R2; metadata is stored in D1.

**Request Body**

```json
{
  "name": "thesis.tex",
  "content": "\\documentclass{article}...",
  "engine": "lualatex"
}
```

| Field | Required | Values |
|-------|----------|--------|
| `name` | Yes | File display name |
| `content` | No | LaTeX source (default: empty) |
| `engine` | No | `lualatex` (default), `pdflatex`, `xelatex` |

**Response** `201 Created` — metadata only (no `content` field)

```json
{
  "id": "abc123",
  "user_id": "xyz789",
  "name": "thesis.tex",
  "r2_key": "latex/xyz789/abc123.tex",
  "engine": "lualatex",
  "created_at": 1714000000,
  "updated_at": 1714000000
}
```

**Errors**

| Code | Reason |
|------|--------|
| 400 | Missing `name` |
| 401 | Invalid token |

---

### GET /api/latex-files/:id

Fetch file metadata from D1 and content from R2.

**Response** `200 OK`

```json
{
  "id": "abc123",
  "user_id": "xyz789",
  "name": "thesis.tex",
  "engine": "lualatex",
  "content": "\\documentclass{article}...",
  "created_at": 1714000000,
  "updated_at": 1714003600
}
```

**Errors**

| Code | Reason |
|------|--------|
| 401 | Invalid token |
| 404 | File not found or not owned by user |

---

### PUT /api/latex-files/:id

Update name, engine, and/or content. All fields are optional. If `content` is provided, the R2 object is overwritten in-place.

**Request Body** (all fields optional)

```json
{
  "name": "renamed.tex",
  "content": "\\documentclass{article}...",
  "engine": "xelatex"
}
```

**Response** `200 OK` — updated metadata (no `content` field)

**Errors**

| Code | Reason |
|------|--------|
| 400 | Invalid JSON body |
| 401 | Invalid token |
| 404 | File not found or not owned by user |

---

### DELETE /api/latex-files/:id

Delete the file from R2 (best-effort) and remove the D1 row.

**Response** `204 No Content`

**Errors**

| Code | Reason |
|------|--------|
| 401 | Invalid token |
| 404 | File not found or not owned by user |

---

### GET /api/assets

List all uploaded asset files owned by the authenticated user, ordered by upload date (newest first).

**Response** `200 OK`

```json
[
  {
    "id": "abc123",
    "user_id": "xyz789",
    "name": "photo.jpg",
    "r2_key": "assets/xyz789/abc123",
    "mime_type": "image/jpeg",
    "size": 204800,
    "created_at": 1714000000
  }
]
```

---

### POST /api/assets

Upload a new asset file. Content is stored in R2; metadata (original filename, MIME type, size) is stored in D1. Maximum file size is 10 MB.

**Request Body** — `multipart/form-data`

| Field | Required | Description |
|-------|----------|-------------|
| `file` | Yes | File to upload (any type) |

**Response** `201 Created`

```json
{
  "id": "abc123",
  "user_id": "xyz789",
  "name": "photo.jpg",
  "r2_key": "assets/xyz789/abc123",
  "mime_type": "image/jpeg",
  "size": 204800,
  "created_at": 1714000000
}
```

**Errors**

| Code | Reason |
|------|--------|
| 400 | Missing `file` field or malformed multipart |
| 401 | Invalid token |
| 413 | File exceeds 10 MB |

---

### DELETE /api/assets/:id

Delete the asset from R2 (best-effort) and remove the D1 row.

**Response** `204 No Content`

**Errors**

| Code | Reason |
|------|--------|
| 401 | Invalid token |
| 404 | Asset not found or not owned by user |

---

## Route Summary

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/register` | None | Register a new user account |
| POST | `/setup` | SETUP_SECRET | Create first admin + OAuth client |
| GET | `/oauth/authorize` | None | Show login form |
| POST | `/oauth/authorize` | None | Login and get auth code |
| POST | `/oauth/token` | None | Exchange code or refresh token |
| POST | `/oauth/revoke` | None | Revoke refresh token |
| GET | `/api/me` | JWT | Get own profile |
| GET | `/api/users/:id` | JWT | Get user by ID (own or any if admin) |
| GET | `/api/admin/users` | JWT | List all users |
| POST | `/api/admin/users` | JWT | Create new user |
| POST | `/api/compile` | JWT | Compile LaTeX to PDF (MD5-cached) |
| GET | `/api/latex-files` | JWT | List own LaTeX files |
| POST | `/api/latex-files` | JWT | Create a new LaTeX file |
| GET | `/api/latex-files/:id` | JWT | Get file metadata + content |
| PUT | `/api/latex-files/:id` | JWT | Update name, engine, or content |
| DELETE | `/api/latex-files/:id` | JWT | Delete a LaTeX file |
| GET | `/api/assets` | JWT | List own uploaded assets |
| POST | `/api/assets` | JWT | Upload a new asset file (max 10 MB) |
| DELETE | `/api/assets/:id` | JWT | Delete an uploaded asset |

---

## JWT Claims

```json
{
  "sub": "<user_id>",
  "email": "user@example.com",
  "role": "user",
  "scope": "openid",
  "iat": 1714000000,
  "exp": 1714003600
}
```
