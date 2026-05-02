# System User Flow & Architecture

This document describes the end-to-end flows for the `sir-backend` OAuth 2.0 system, which follows **RFC 8252 (OAuth 2.0 for Native Apps)** using Loopback Interface Redirection.

---

## 1. Initial System Bootstrapping (One-time)
Before the system can be used, it must be initialized with an admin user and a default client.

1.  **Requirement**: Set `SETUP_SECRET` in `.dev.vars` or Cloudflare Secrets.
2.  **Action**: `POST /setup?secret=YOUR_SECRET`
    *   **Input**: JSON containing `admin_email`, `admin_password`, `client_id`, `client_secret`, and `client_name`.
    *   **Outcome**: 
        *   Creates a `User` record with `role: "admin"`.
        *   Creates an `OAuthClient` record.
        *   Returns the IDs of the created resources.

---

## 2. OAuth 2.0 Authorization Code Flow
This flow is used by applications (clients) to authenticate users and obtain access tokens.

### A. Authorization Request
The application redirects the user to the Authorization Server.
*   **Action**: `GET /oauth/authorize`
*   **Params**: `client_id`, `redirect_uri` (must be loopback), `response_type=code`, `state`, `scope`.
*   **Outcome**: The server renders a **Login Form**.

### B. User Login
The user enters their credentials.
*   **Action**: `POST /oauth/authorize`
*   **Process**: 
    1.  Verifies user credentials in the D1 database.
    2.  Generates a single-use `AuthCode` (expires in 10 minutes).
    3.  Redirects back to the application's `redirect_uri` with `?code=XYZ&state=...`.

### C. Token Exchange
The application exchanges the `code` for tokens.
*   **Action**: `POST /oauth/token`
*   **Params**: `grant_type=authorization_code`, `code`, `client_id`, `client_secret`, `redirect_uri`.
*   **Outcome**: Returns a JSON response containing:
    *   `access_token`: A JWT valid for 1 hour.
    *   `refresh_token`: A long-lived random string.
    *   `expires_in`: 3600.

---

## 3. Protected Resource Access
Once the application has an `access_token`, it can call protected APIs.

### A. User Profile (`/api/me`)
*   **Authorization**: `Bearer <access_token>`
*   **Access**: Any valid user.
*   **Outcome**: Returns the `sub` (ID), `email`, `role`, and `scope` from the JWT claims.

### B. Admin User Management (`/api/admin/users`)
*   **Authorization**: `Bearer <access_token>`
*   **Access**: Users with `role: "admin"`.
*   **Outcome**: 
    *   `GET`: Lists all users in the system.
    *   `POST`: Creates a new user manually.

---

## 4. Token Maintenance

### A. Token Refresh
When the `access_token` expires, the application uses the `refresh_token` to get a new one.
*   **Action**: `POST /oauth/token`
*   **Params**: `grant_type=refresh_token`, `refresh_token`, `client_id`, `client_secret`.
*   **Security**: The server uses **Refresh Token Rotation**. The old refresh token is revoked, and a brand new one is issued alongside the new access token.

### B. Revocation
The user or application can explicitly invalidate a session.
*   **Action**: `POST /oauth/revoke`
*   **Params**: `token` (the refresh token).
*   **Outcome**: The token is marked as `revoked` in the D1 database and can no longer be used for refreshing.

---

## Technical Stack Summary
*   **Runtime**: Cloudflare Workers (WASM).
*   **Language**: Go (syumai/workers).
*   **Database**: Cloudflare D1 (SQLite).
*   **ORM**: GORM (with custom D1 dialector).
*   **Identity**: JWT (HS256) + Argon2/Password Hashing.
*   **Redirection**: Loopback Interface (RFC 8252) for 127.0.0.1/localhost security.
