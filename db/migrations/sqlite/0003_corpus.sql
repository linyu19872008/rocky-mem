-- rocky-mem 0003: corpus layer (source / chunk)
--
-- Corpus is background material, not evidence: no `raw` row is created for it.
-- `chunk.text` keeps its own copy of the (already redacted) slice, so no
-- byte/character offset arithmetic is needed and the two dialects behave the same.
--
-- No backfill: documents ingested before 0003 were turned into proposed
-- experiences with their text truncated at 4000 chars, so nothing lossless
-- can be rebuilt from them.

CREATE TABLE source (
  id            INTEGER PRIMARY KEY,
  uid           TEXT    NOT NULL UNIQUE,
  kind          TEXT    NOT NULL CHECK (kind IN ('file','paste')),
  uri           TEXT    NOT NULL,
  title         TEXT    NOT NULL,
  format        TEXT    NOT NULL,
  text_hash     TEXT    NOT NULL,
  char_count    INTEGER NOT NULL DEFAULT 0,
  chunk_count   INTEGER NOT NULL DEFAULT 0,
  lang          TEXT    NOT NULL DEFAULT 'und',
  sensitivity   INTEGER NOT NULL DEFAULT 2 CHECK (sensitivity BETWEEN 1 AND 3),
  status        TEXT    NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','archived')),
  superseded_by INTEGER REFERENCES source(id),
  project_id    INTEGER REFERENCES project(id),
  actor_id      INTEGER REFERENCES app_user(id),
  created_at    INTEGER NOT NULL,
  t_last_used   INTEGER,
  use_count     INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX ux_source_text_hash ON source(text_hash);
CREATE INDEX idx_source_project ON source(project_id, status, created_at);

CREATE TABLE chunk (
  id          INTEGER PRIMARY KEY,
  uid         TEXT    NOT NULL UNIQUE,
  source_id   INTEGER NOT NULL REFERENCES source(id) ON DELETE CASCADE,
  ord         INTEGER NOT NULL,
  heading     TEXT,
  text        TEXT    NOT NULL,
  char_count  INTEGER NOT NULL,
  sensitivity INTEGER NOT NULL DEFAULT 2 CHECK (sensitivity BETWEEN 1 AND 3)
);

CREATE UNIQUE INDEX ux_chunk_ord ON chunk(source_id, ord);
CREATE INDEX idx_chunk_source ON chunk(source_id);

UPDATE meta SET v = '3' WHERE k = 'schema_version';
