# Data Flow Diagram (DFD)

This document illustrates how data flows between entities, processes, and data stores within the `sir-backend` system.

---

## 1. High-Level Data Flow

The following diagram shows how information (Credentials, Codes, Tokens, and Resources) moves through the system.

```mermaid
graph TD
    %% Entities
    User((User/Browser))
    ClientApp[Client Application]
    
    subgraph sir_backend [sir-backend / Cloudflare Worker]
        AuthHandler{OAuth Handler}
        Middleware[Auth Middleware]
        NoteHandler[Notes Handler]
        TokenLib[Token & Crypto Lib]
    end

    subgraph Storage [Cloudflare D1 Database]
        DB_Users[(Users Table)]
        DB_OAuth[(OAuth Tables<br/>Codes/Clients/Tokens)]
        DB_Notes[(Notes Table)]
    end

    %% Flows
    User -- "1. Credentials (POST)" --> AuthHandler
    AuthHandler -- "2. Verify/Hash" --> TokenLib
    TokenLib -- "3. Query/Compare" --> DB_Users
    
    AuthHandler -- "4. Generate Code" --> DB_OAuth
    AuthHandler -- "5. Redirect with Code" --> User
    User -- "6. Forward Code" --> ClientApp
    
    ClientApp -- "7. Exchange Code (POST)" --> AuthHandler
    AuthHandler -- "8. Issue JWT & Refresh Token" --> DB_OAuth
    AuthHandler -- "9. Tokens (JSON)" --> ClientApp

    ClientApp -- "10. API Request (JWT)" --> Middleware
    Middleware -- "11. Validate Signature" --> TokenLib
    Middleware -- "12. Authorized Request (UserID)" --> NoteHandler
    NoteHandler -- "13. CRUD Scoped by UserID" --> DB_Notes
```

---

## 2. Data Elements & Security

| Data Element | Sensitivity | Transformation | Storage Policy |
| :--- | :--- | :--- | :--- |
| **User Password** | High | Hashed via Argon2/HMAC | Never stored in plain text. |
| **Auth Code** | Medium | Random 32-char string | Stored in D1, marked `used` after 1 min. |
| **Access Token (JWT)** | High | Signed with HS256 (JWT_SECRET) | **Not stored** on server (Stateless). |
| **Refresh Token** | High | Random 48-char string | Stored in D1, revoked after use (Rotation). |
| **User Notes** | Medium | None (Scoped to UserID) | Stored in D1, accessible only by owner. |

---

## 3. Security Boundaries

1.  **Trust Boundary: User Browser <-> Server**
    *   Data is protected via **TLS (HTTPS)**.
    *   Credentials are sent only via POST body to prevent logging in URLs.
2.  **Trust Boundary: Client App <-> Server**
    *   The `client_secret` acts as the primary authentication for the Client.
    *   Loopback enforcement prevents "Inter-app communication" attacks on the authorization code.
3.  **Persistence Boundary: Server <-> D1**
    *   The database is only accessible by the Worker via the `DB` binding.
    *   ORM hooks ensure IDs and Timestamps are generated in a trusted environment before hitting the disk.

---

## 4. Lifecycle of a Note (Data Flow Example)

1.  **Input**: Client sends `{ "title": "Secret", "content": "..." }` + `Authorization: Bearer <JWT>`.
2.  **Extraction**: Middleware decodes JWT, verifying it hasn't been tampered with using the server's secret. It extracts `sub: "user_123"`.
3.  **Process**: Handler receives the request and explicitly constructs a DB query: `INSERT INTO notes VALUES (user_id="user_123", ...)`.
4.  **Storage**: D1 writes the record to the encrypted persistent store on the Cloudflare Edge.
5.  **Output**: JSON representation of the saved note is returned to the Client.
