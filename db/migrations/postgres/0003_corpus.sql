-- rocky-mem 0003: corpus layer (source / chunk) — PostgreSQL dialect.
--
-- Mirrors db/migrations/sqlite/0003_corpus.sql. Types follow 0001's rules:
-- identity PKs, BIGINT for counters/time, no WITHOUT ROWID equivalent.
--
-- Corpus is background material, not evidence: no `raw` row is created for it.
-- No backfill (pre-0003 ingest truncated text into proposed experiences).

CREATE TABLE source (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  uid           TEXT    NOT NULL UNIQUE,
  kind          TEXT    NOT NULL CHECK (kind IN ('file','paste')),
  uri           TEXT    NOT NULL,
  title         TEXT    NOT NULL,
  format        TEXT    NOT NULL,
  text_hash     TEXT    NOT NULL,
  char_count    BIGINT  NOT NULL DEFAULT 0,
  chunk_count   BIGINT  NOT NULL DEFAULT 0,
  lang          TEXT    NOT NULL DEFAULT 'und',
  sensitivity   INTEGER NOT NULL DEFAULT 2 CHECK (sensitivity BETWEEN 1 AND 3),
  status        TEXT    NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','archived')),
  superseded_by BIGINT REFERENCES source(id),
  project_id    BIGINT REFERENCES project(id),
  actor_id      BIGINT REFERENCES app_user(id),
  created_at    BIGINT  NOT NULL,
  t_last_used   BIGINT,
  use_count     BIGINT  NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX ux_source_text_hash ON source(text_hash);
CREATE INDEX idx_source_project ON source(project_id, status, created_at);

CREATE TABLE chunk (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  uid         TEXT    NOT NULL UNIQUE,
  source_id   BIGINT  NOT NULL REFERENCES source(id) ON DELETE CASCADE,
  ord         BIGINT  NOT NULL,
  heading     TEXT,
  text        TEXT    NOT NULL,
  char_count  BIGINT  NOT NULL,
  sensitivity INTEGER NOT NULL DEFAULT 2 CHECK (sensitivity BETWEEN 1 AND 3)
);

CREATE UNIQUE INDEX ux_chunk_ord ON chunk(source_id, ord);
CREATE INDEX idx_chunk_source ON chunk(source_id);

UPDATE meta SET v = '3' WHERE k = 'schema_version';
