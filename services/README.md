# Services

Docker services ที่รันแยกจาก main backend

## latex-server

Go HTTP server สำหรับ compile LaTeX → PDF ผ่าน LuaLaTeX / pdfLaTeX / XeLaTeX

**Endpoints**
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/compile` | Compile LaTeX source → PDF |
| `POST` | `/fonts` | Upload font file (`.ttf`, `.otf`) |
| `GET` | `/fonts` | List uploaded fonts |
| `DELETE` | `/fonts/{name}` | Delete uploaded font |
| `GET` | `/health` | Health check |

**Font directories ใน container**
| Path | Source | Mode |
|------|--------|------|
| `/usr/local/share/fonts/project` | `./fonts/` (bind mount) | read-only, fc-cache ตอน startup |
| `/usr/local/share/fonts/api` | named volume `fonts_api` | writable, fc-cache หลัง upload/delete |

**Clone เฉพาะ fonts folder**
```bash
git clone --no-checkout --filter=blob:none https://github.com/PuemMTH/sir-backend.git
cd sir-backend
git sparse-checkout init --cone
git sparse-checkout set services/latex-server/fonts
git checkout main
```

**Run local dev**
```bash
cd services/latex-server
docker compose up --build
# server พร้อมที่ http://localhost:3001
```
