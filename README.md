# sir-backend

A robust, maintainable OAuth 2.0 backend built with **Go**, **GORM**, and **Cloudflare D1**. This project implements a secure Authorization Server specifically designed for native and web applications using the Loopback Interface Redirection (RFC 8252).

## 🚀 Key Features

- **OAuth 2.0 Authorization Code Flow**: Fully compliant with [RFC 8252](https://tools.ietf.org/html/rfc8252).
- **Security First**: 
  - Refresh Token Rotation (Automatic revocation of old tokens).
  - JWT Access Tokens (HS256).
  - Password Hashing (Argon2 with custom salt/HMAC).
- **GORM & D1 Integration**: Uses a custom, CGO-free dialector to run GORM on Cloudflare's WASM-based SQLite (D1).
- **Role-Based Access Control (RBAC)**: Support for `admin` and `user` roles.
- **Example Resource**: A complete "Notes" CRUD API to demonstrate authenticated data scoping.

## 🛠 Tech Stack

- **Runtime**: [Cloudflare Workers](https://workers.cloudflare.com/) (WASM)
- **Language**: [Go](https://go.dev/) (via [syumai/workers](https://github.com/syumai/workers))
- **Database**: [Cloudflare D1](https://developers.cloudflare.com/d1/) (SQLite)
- **ORM**: [GORM](https://gorm.io/)
- **Configuration**: [Wrangler](https://developers.cloudflare.com/workers/wrangler/)

## 📂 Project Structure

```text
.
├── docs/                 # Detailed documentation (Mermaid diagrams, flows)
├── internal/             # Core application logic (Go packages)
│   ├── handler/          # HTTP Request handlers
│   ├── middleware/       # Auth & Role middleware
│   ├── model/            # GORM database models
│   ├── store/            # Data Access Object (DAO) & D1 Dialector
│   └── token/            # JWT & Crypto utilities
├── migrations/           # SQL migration files for D1
├── main.go               # Application entry point
├── Makefile              # Build and development commands
└── wrangler.toml         # Cloudflare Workers configuration
```

## 🏁 Getting Started

### 1. Prerequisites
- Go 1.21+
- Node.js & npm (for Wrangler)
- `npx wrangler d1 migrations apply sir-db --local`

### 2. Environment Setup
Create a `.dev.vars` file for local secrets:
```bash
SETUP_SECRET=your-secret-key
JWT_SECRET=your-jwt-signing-secret
```

### 3. Local Development
```bash
make dev
```
The server will start on `http://localhost:8787`.

### 4. Initialization
Seed your first admin user and OAuth client:
```bash
curl -X POST "http://localhost:8787/setup?secret=your-secret-key" \
     -H "Content-Type: application/json" \
     -d '{
       "admin_email": "admin@example.com",
       "admin_password": "securepassword",
       "client_id": "test-client",
       "client_secret": "test-secret",
       "client_name": "My Native App"
     }'
```

## 🧪 Testing

Run the end-to-end test suite to verify the OAuth flow and Note CRUD operations:
```bash
./test_flow.sh
```

## 📖 Documentation

For deeper architectural details, check the `docs/` directory:
- [End-to-End Flow (Diagrams)](docs/END_TO_END_FLOW.md)
- [Data Flow & Security](docs/DATA_FLOW.md)
- [User Journey](docs/USER_FLOW.md)
- [Deployment Guide](docs/DEPLOYMENT.md)

## 📄 License

MIT
