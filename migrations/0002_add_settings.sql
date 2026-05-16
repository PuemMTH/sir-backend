CREATE TABLE IF NOT EXISTS settings (
  key        TEXT    PRIMARY KEY,
  value      TEXT    NOT NULL,
  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

INSERT OR IGNORE INTO settings (key, value, updated_at)
VALUES ('compile_url', 'https://ipulab.com/service/latex-server/compile', unixepoch());
