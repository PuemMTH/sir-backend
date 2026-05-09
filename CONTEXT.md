# SIR Platform — Project Context

## Repos

| Path | Description |
|------|-------------|
| `C:\Users\admin\Desktop\sir-backend` | Cloudflare Worker (Go/WASM via syumai/workers) |
| `C:\Users\admin\Desktop\sir-latex` | Vite + React + TypeScript frontend (Cloudflare Pages) |

---

## Backend (sir-backend)

### Deploy workflow (Windows — `make` not available)

```bash
GOOS=js GOARCH=wasm go build -o build/worker.wasm .
cp build/worker.wasm build/app.wasm
# Temporarily set wrangler.toml [build] command = "echo skip"
wrangler deploy
# Restore command = "make build-std"
```

### Bindings

| Binding | Resource |
|---------|----------|
| `DB` | D1: `sir-db` |
| `LATEX_BUCKET` | R2: `sir-latex-files` |
| `COMPILE_URL` | `https://ipulab.com/service/latex-server/compile` |

### Key routes

| Route | Auth | Description |
|-------|------|-------------|
| `POST /api/compile` | JWT | Compile LaTeX → PDF |
| `GET/POST /api/latex-files` | JWT | List / create files |
| `PUT/DELETE /api/latex-files/:id` | JWT | Update / delete file |
| `GET/POST /api/assets` | JWT | List / upload assets |
| `DELETE /api/assets/:id` | JWT | Delete asset |
| `POST /oauth/token` | — | authorization_code flow only (no password grant) |

### Compile caching — 3 levels

| Level | Header | Speed | TTL |
|-------|--------|-------|-----|
| Cloudflare Cache API | `X-Cache: CF-HIT` | ~5ms | 86400s |
| D1 hash + R2 PDF | `X-Cache: HIT` | ~100ms | permanent |
| Upstream compile | `X-Cache: MISS` | ~10-30s | — |

Cache key = `MD5(engine + ":" + source + assetIDs)`

### Assets

- Uploaded to R2 at `assets/{userID}/{assetID}`
- Metadata in D1 `user_assets` table (`migrations/0006_add_user_assets.sql`)
- On compile: all user assets fetched from R2, base64-encoded, sent to latex-server as `files[]`

### Database migrations

**Always write a migration file** when adding/modifying tables: `migrations/NNNN_<name>.sql`

Apply: `wrangler d1 migrations apply sir-db --remote`

---

## latex-server (services/latex-server)

Docker service on `ipulab.com`. Go HTTP server inside `texlive/texlive:latest`.

- Accepts `files []fileEntry` (base64) in compile request body
- Writes files to temp dir alongside `document.tex`
- `entrypoint.sh` must have **LF line endings** (not CRLF)

### Docker workflow (mandatory after any edit)

```bash
docker-compose build
docker-compose up -d
curl https://ipulab.com/service/latex-server/health  # must return {"ok":true}
```

### Test compile with files

```bash
curl -X POST http://localhost:3001/compile \
  -H "Content-Type: application/json" \
  -d @payload.json -o out.pdf -w "%{http_code}"
```

---

## Frontend (sir-latex)

### Features

- **Dashboard**: file list with inline rename
- **Editor**:
  - CodeMirror LaTeX editor with debounced auto-compile
  - PDF preview via `react-pdf`
  - Auto-save on compile success
  - No loading overlay during recompile (old PDF stays visible)
  - Cache badge: `⚡ CF` (cyan) or `◎ cached` (blue) with tooltip
  - Asset panel (Files button): upload / list / delete / copy `\includegraphics{}` reference
  - Export PDF button
  - Export PNG button (pdfjs canvas, 2x resolution, per-page)
  - Ctrl+S / Cmd+S to save (uses `e.code === "KeyS"` for Thai keyboard compatibility)

### Deploy

```bash
pnpm run deploy  # build + wrangler pages deploy --branch=main
```

---

## SSH hosts

| Alias | Host | User |
|-------|------|------|
| `ipu` | 10.222.44.224 | ipu |
| `deploy` | 10.222.44.224 | deploy |
| `home` | h.puem.me | puem |
| `pv` | 157.85.98.168 | root |
