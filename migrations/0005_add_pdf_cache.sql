CREATE TABLE IF NOT EXISTS pdf_cache (
  source_hash TEXT PRIMARY KEY,
  r2_key      TEXT NOT NULL,
  engine      TEXT NOT NULL DEFAULT 'lualatex',
  created_at  INTEGER NOT NULL
);
