# End-to-End System Flow

This document provides a visual and technical breakdown of the `sir-backend` system flows, separated by **Client Perspective** and **Server Internal Logic**.

---

## 1. System Bootstrapping (Initialization)

Before any users can log in, the system must be seeded with an admin and a client.

```mermaid
sequenceDiagram
    participant Admin as Developer/Admin
    participant Server as sir-backend (Go)
    participant DB as Cloudflare D1 (SQLite)

    Admin->>Server: POST /setup?secret=SETUP_SECRET
    Note over Server: Validates SETUP_SECRET against Environment Variable
    Server->>DB: INSERT INTO users (role: 'admin')
    Server->>DB: INSERT INTO oauth_clients
    DB-->>Server: Success
    Server-->>Admin: 201 Created (Admin ID, Client ID)
```

---

## 2. OAuth 2.0 Authorization Code Flow (RFC 8252)

This flow describes how a Native/Web application obtains access to a user's account.

### A. Authorization & Login
```mermaid
sequenceDiagram
    participant User as User (Browser)
    participant Client as Client App (e.g., localhost:8080)
    participant Server as sir-backend (Go)
    participant DB as Cloudflare D1 (SQLite)

    Client->>User: Redirect to Server /oauth/authorize
    User->>Server: GET /oauth/authorize?client_id=...&redirect_uri=...
    Server-->>User: Renders HTML Login Form
    User->>Server: POST /oauth/authorize (email, password)
    
    Server->>DB: SELECT user WHERE email = ?
    DB-->>Server: User Data (Hashed Password)
    Note over Server: Verifies Password (Argon2/HMAC)
    
    Server->>DB: INSERT INTO auth_codes (code, user_id, expires_at)
    Server-->>User: 302 Redirect to Client redirect_uri?code=XYZ
    User->>Client: Handles Redirect with Code
```

### B. Token Exchange & Refresh
```mermaid
sequenceDiagram
    participant Client as Client App
    participant Server as sir-backend (Go)
    participant DB as Cloudflare D1 (SQLite)

    Note over Client: Exchange Code for Tokens
    Client->>Server: POST /oauth/token (grant_type=authorization_code, code, secret)
    Server->>DB: SELECT & UPDATE auth_codes (set used=1)
    Server->>DB: INSERT INTO refresh_tokens
    Note over Server: Signs JWT Access Token (HS256)
    Server-->>Client: 200 OK (access_token, refresh_token)

    Note over Client: When Access Token expires...
    Client->>Server: POST /oauth/token (grant_type=refresh_token, refresh_token)
    Server->>DB: SELECT refresh_token (check revoked/expired)
    Server->>DB: UPDATE refresh_tokens (revoke old)
    Server->>DB: INSERT INTO refresh_tokens (issue new - Rotation)
    Server-->>Client: 200 OK (new access_token, new refresh_token)
```

---

## 3. Resource Access Flow (Notes CRUD)

How an authenticated application manages data.

```mermaid
sequenceDiagram
    participant Client as Client App
    participant Auth as Auth Middleware
    participant Handler as Notes Handler
    participant DB as Cloudflare D1 (SQLite)

    Client->>Auth: POST /api/notes (Header: Bearer <JWT>)
    Note over Auth: Validates JWT Signature & Expiry
    Note over Auth: Extracts UserID from Claims
    Auth->>Handler: Forward Request with Context(UserID)
    
    Handler->>DB: INSERT INTO notes (user_id, title, content)
    DB-->>Handler: Success (Note ID)
    Handler-->>Client: 201 Created (Note JSON)

    Client->>Auth: GET /api/notes
    Auth->>Handler: Forward Context(UserID)
    Handler->>DB: SELECT * FROM notes WHERE user_id = ?
    DB-->>Handler: List of User's Notes
    Handler-->>Client: 200 OK (Notes Array)
```

---

## Flow Summary

### Client-Side Flow
1.  **Directs** user to the login page.
2.  **Captures** the authorization code from the callback URL.
3.  **Exchanges** the code for tokens via a secure back-channel POST.
4.  **Stores** the refresh token securely.
5.  **Attaches** the access token to the `Authorization: Bearer` header for all API calls.

### Server-Side Flow
1.  **Authenticates** the user against D1.
2.  **Issues** short-lived JWTs and long-lived Refresh Tokens.
3.  **Enforces** Refresh Token Rotation (one-time use for refresh tokens).
4.  **Scopes** data access: All database queries for resources (like Notes) are hard-coded to filter by the `UserID` found in the JWT.
