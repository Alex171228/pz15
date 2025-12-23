-- 001_init.sql
CREATE TABLE IF NOT EXISTS notes (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notes_audit (
  id BIGSERIAL PRIMARY KEY,
  note_id BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notes_created_id ON notes (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_notes_title_gin
  ON notes USING GIN (to_tsvector('simple', title));
