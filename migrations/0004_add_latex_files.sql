CREATE TABLE IF NOT EXISTS latex_files (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL,
  name       TEXT NOT NULL,
  r2_key     TEXT NOT NULL,
  engine     TEXT NOT NULL DEFAULT 'lualatex' CHECK(engine IN ('lualatex', 'pdflatex', 'xelatex')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_latex_files_user_id ON latex_files(user_id);
