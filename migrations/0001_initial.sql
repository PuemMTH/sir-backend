CREATE TABLE IF NOT EXISTS notes (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL,
  title      TEXT NOT NULL,
  content    TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);

CREATE TABLE IF NOT EXISTS latex_files (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL,
  name       TEXT NOT NULL,
  r2_key     TEXT NOT NULL,
  engine     TEXT NOT NULL DEFAULT 'lualatex' CHECK(engine IN ('lualatex', 'pdflatex', 'xelatex')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_latex_files_user_id ON latex_files(user_id);

CREATE TABLE IF NOT EXISTS pdf_cache (
  source_hash TEXT PRIMARY KEY,
  r2_key      TEXT NOT NULL,
  engine      TEXT NOT NULL DEFAULT 'lualatex',
  created_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS user_assets (
  id               TEXT    PRIMARY KEY,
  user_id          TEXT    NOT NULL,
  name             TEXT    NOT NULL,
  r2_key           TEXT    NOT NULL,
  thumbnail_r2_key TEXT,
  mime_type        TEXT    NOT NULL DEFAULT 'application/octet-stream',
  size             INTEGER NOT NULL DEFAULT 0,
  created_at       INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_user_assets_user_id ON user_assets(user_id);
