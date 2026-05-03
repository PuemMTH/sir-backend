# Commands Reference

## sir-backend (Cloudflare Worker — Go/WASM)

### Build

```sh
# Standard Go build (≈18 MB — no TinyGo required)
make build-std

# TinyGo build (smaller binary — requires TinyGo installed)
make build

# Verify the project compiles (no output produced)
GOOS=js GOARCH=wasm go build ./...
```

### Deploy

```sh
# Build (std) + deploy to production
make build-std && npx wrangler deploy

# Deploy only (if already built)
npx wrangler deploy
```

### Local Dev

```sh
make dev
# or
make build-std && npx wrangler dev
```

### Logs (live tail)

```sh
npx wrangler tail --format pretty
```

### D1 Database — Migrations

```sh
# Apply all pending migrations to production
npx wrangler d1 migrations apply sir-db --remote

# Apply to local dev DB
npx wrangler d1 migrations apply sir-db

# Create a new migration file
npx wrangler d1 migrations create sir-db <migration-name>

# List applied migrations
npx wrangler d1 migrations list sir-db --remote
```

### D1 Database — Ad-hoc Queries

```sh
# Run a query against production
npx wrangler d1 execute sir-db --remote --command "SELECT * FROM users;"

# Run a query against local dev DB
npx wrangler d1 execute sir-db --command "SELECT * FROM users;"
```

### Secrets / Env Vars

```sh
# Set a secret (e.g. JWT_SECRET)
npx wrangler secret put JWT_SECRET

# List all secrets
npx wrangler secret list
```

### Clean

```sh
make clean   # removes build/
```

---

## sir-latex (Cloudflare Pages — React/Vite)

### Install

```sh
pnpm install
```

### Local Dev

```sh
pnpm dev
```

### Build

```sh
pnpm build
```

### Deploy to Production

```sh
# Build + deploy to production (branch=main)
pnpm run deploy
```

### Preview (local)

```sh
pnpm preview
```

### Lint

```sh
pnpm lint
```

---

## latex-server (Docker — Go compile service)

> Located at `services/latex-server/`. Runs on the remote server behind a reverse proxy.

### Start

```sh
cd services/latex-server
docker compose up -d
```

### Stop

```sh
docker compose down
```

### Rebuild & Restart

```sh
docker compose up -d --build
```

### View Logs

```sh
docker compose logs -f latex-server
```

### Health Check

```sh
curl https://ipulab.com/service/latex-server/health
```

### Test Compile

```sh
curl -X POST https://ipulab.com/service/latex-server/compile \
  -H "Content-Type: application/json" \
  -d '{"source":"\\documentclass{article}\\begin{document}Hello\\end{document}","engine":"lualatex"}' \
  --output test.pdf
```
