-- rocky-mem P0 SQLite schema (from design/schema.md)

PRAGMA foreign_keys = ON;

CREATE TABLE meta (
  k TEXT PRIMARY KEY,
  v TEXT NOT NULL
) WITHOUT ROWID;

CREATE TABLE app_user (
  id           INTEGER PRIMARY KEY,
  email        TEXT,
  display_name TEXT    NOT NULL,
  status       TEXT    NOT NULL DEFAULT 'active'
                 CHECK (status IN ('active','disabled')),
  created_at   INTEGER NOT NULL
);

CREATE UNIQUE INDEX ux_app_user_email
  ON app_user(email) WHERE email IS NOT NULL AND email <> '';

CREATE TABLE team (
  id         INTEGER PRIMARY KEY,
  name       TEXT    NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE TABLE membership (
  team_id     INTEGER NOT NULL REFERENCES team(id)     ON DELETE CASCADE,
  user_id     INTEGER NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
  member_role TEXT    NOT NULL
                CHECK (member_role IN ('owner','admin','member','viewer')),
  PRIMARY KEY (team_id, user_id)
);

CREATE TABLE api_token (
  id           INTEGER PRIMARY KEY,
  user_id      INTEGER NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
  name         TEXT    NOT NULL DEFAULT '',
  token_hash   TEXT    NOT NULL,
  scopes       TEXT    NOT NULL DEFAULT 'read,write',
  expires_at   INTEGER,
  last_used_at INTEGER,
  created_at   INTEGER NOT NULL
);
CREATE UNIQUE INDEX ux_api_token_hash ON api_token(token_hash);

CREATE TABLE project (
  id              INTEGER PRIMARY KEY,
  name            TEXT    NOT NULL,
  remote_url_norm TEXT    NOT NULL DEFAULT '',
  root_hint       TEXT,
  created_by      INTEGER REFERENCES app_user(id),
  created_at      INTEGER NOT NULL
);
CREATE UNIQUE INDEX ux_project_remote
  ON project(remote_url_norm) WHERE remote_url_norm <> '';

CREATE TABLE raw (
  id               INTEGER PRIMARY KEY,
  uid              TEXT    NOT NULL UNIQUE,
  hash             TEXT    NOT NULL UNIQUE,
  channel          TEXT    NOT NULL
                     CHECK (channel IN ('manual','agent','file','git','transcript','web')),
  source_ref       TEXT,
  actor_id         INTEGER REFERENCES app_user(id),
  agent            TEXT,
  session_id       TEXT,
  cwd              TEXT,
  project_id       INTEGER REFERENCES project(id),
  mime             TEXT,
  lang             TEXT    NOT NULL DEFAULT 'und',
  body             BLOB    NOT NULL,
  redaction        TEXT,
  rulepack_version TEXT,
  sensitivity      INTEGER NOT NULL DEFAULT 2
                     CHECK (sensitivity BETWEEN 1 AND 3),
  created_at       INTEGER NOT NULL
);

CREATE INDEX idx_raw_time    ON raw(created_at);
CREATE INDEX idx_raw_project ON raw(project_id, created_at);

CREATE TABLE memory (
  id            INTEGER PRIMARY KEY,
  uid           TEXT    NOT NULL UNIQUE,
  lineage       TEXT    NOT NULL,
  scope         TEXT    NOT NULL
                  CHECK (scope IN ('global','team','project','session')),
  scope_ref     TEXT    NOT NULL DEFAULT '',
  mem_key       TEXT,
  mem_type      TEXT    NOT NULL
                  CHECK (mem_type IN ('pref','fact','constraint','glossary')),
  title         TEXT    NOT NULL,
  body_md       TEXT    NOT NULL,
  lang          TEXT    NOT NULL DEFAULT 'und',
  pinned        INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0,1)),
  priority      INTEGER NOT NULL DEFAULT 0,
  status        TEXT    NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','superseded','archived')),
  superseded_by INTEGER REFERENCES memory(id),
  sensitivity   INTEGER NOT NULL DEFAULT 2
                  CHECK (sensitivity BETWEEN 1 AND 3),
  actor_id      INTEGER REFERENCES app_user(id),
  rev           INTEGER NOT NULL DEFAULT 1,
  created_at    INTEGER NOT NULL,
  t_last_used   INTEGER,
  use_count     INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX ux_memory_current
  ON memory(scope, scope_ref, mem_key)
  WHERE mem_key IS NOT NULL AND status = 'active';

CREATE INDEX idx_memory_scope
  ON memory(scope, scope_ref, status, pinned DESC, priority DESC);

CREATE INDEX idx_memory_lineage ON memory(lineage, created_at);

CREATE TABLE experience (
  id             INTEGER PRIMARY KEY,
  uid            TEXT    NOT NULL UNIQUE,
  exp_type       TEXT    NOT NULL
                   CHECK (exp_type IN ('howto','pitfall','decision','postmortem')),
  scope          TEXT    NOT NULL DEFAULT 'project'
                   CHECK (scope IN ('global','team','project','session')),
  scope_ref      TEXT    NOT NULL DEFAULT '',
  title          TEXT    NOT NULL,
  symptom        TEXT,
  symptom_key    TEXT,
  root_cause     TEXT,
  fix_md         TEXT,
  fix_hash       TEXT,
  applies_when   TEXT,
  invalid_when   TEXT,
  outcome        TEXT    NOT NULL DEFAULT 'unknown'
                   CHECK (outcome IN ('success','failed','unknown')),
  confidence     REAL    NOT NULL DEFAULT 0.5
                   CHECK (confidence BETWEEN 0 AND 1),
  trust          TEXT    NOT NULL DEFAULT 'proposed'
                   CHECK (trust IN ('proposed','active','contested','stale',
                                    'falsified','archived')),
  conflict_group TEXT,
  cost           TEXT,
  lang           TEXT    NOT NULL DEFAULT 'und',
  sensitivity    INTEGER NOT NULL DEFAULT 2
                   CHECK (sensitivity BETWEEN 1 AND 3),
  actor_id       INTEGER REFERENCES app_user(id),
  verified_by    INTEGER REFERENCES app_user(id),
  rev            INTEGER NOT NULL DEFAULT 1,
  t_created      INTEGER NOT NULL,
  t_verified     INTEGER,
  t_last_used    INTEGER,
  use_count      INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_exp_scope    ON experience(scope, scope_ref, trust);
CREATE INDEX idx_exp_trust    ON experience(trust, t_last_used);
CREATE INDEX idx_exp_symptom  ON experience(symptom_key) WHERE symptom_key IS NOT NULL;
CREATE INDEX idx_exp_conflict ON experience(conflict_group) WHERE conflict_group IS NOT NULL;

CREATE TABLE evidence (
  id         INTEGER PRIMARY KEY,
  owner_type TEXT    NOT NULL CHECK (owner_type IN ('memory','experience')),
  owner_id   INTEGER NOT NULL,
  raw_id     INTEGER NOT NULL REFERENCES raw(id) ON DELETE RESTRICT,
  quote      TEXT,
  role       TEXT    NOT NULL DEFAULT 'support'
               CHECK (role IN ('support','refute','context')),
  created_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX ux_evidence      ON evidence(owner_type, owner_id, raw_id, role);
CREATE INDEX        idx_evidence_owner ON evidence(owner_type, owner_id);
CREATE INDEX        idx_evidence_raw   ON evidence(raw_id);

CREATE TABLE tag (
  owner_type TEXT    NOT NULL CHECK (owner_type IN ('memory','experience')),
  owner_id   INTEGER NOT NULL,
  tag        TEXT    NOT NULL,
  PRIMARY KEY (owner_type, owner_id, tag)
) WITHOUT ROWID;
CREATE INDEX idx_tag_lookup ON tag(tag, owner_type);

CREATE TABLE audit (
  id          INTEGER PRIMARY KEY,
  ts          INTEGER NOT NULL,
  actor_id    INTEGER,
  agent       TEXT,
  session_id  TEXT,
  action      TEXT    NOT NULL,
  target_type TEXT,
  target_id   INTEGER,
  target_uid  TEXT,
  detail      TEXT
);
CREATE INDEX idx_audit_ts     ON audit(ts);
CREATE INDEX idx_audit_actor  ON audit(actor_id, ts);
CREATE INDEX idx_audit_target ON audit(target_type, target_id);

CREATE TABLE setting (
  key        TEXT PRIMARY KEY,
  value      TEXT NOT NULL,
  updated_at INTEGER NOT NULL
) WITHOUT ROWID;

CREATE TABLE idempotency (
  actor_id      INTEGER NOT NULL,
  idem_key      TEXT    NOT NULL,
  request_hash  TEXT    NOT NULL,
  response_json TEXT    NOT NULL,
  created_at    INTEGER NOT NULL,
  PRIMARY KEY (actor_id, idem_key)
) WITHOUT ROWID;

CREATE TABLE search_doc (
  id         INTEGER PRIMARY KEY,
  kind       TEXT    NOT NULL CHECK (kind IN ('memory','experience','chunk')),
  owner_id   INTEGER NOT NULL,
  lang       TEXT    NOT NULL DEFAULT 'und',
  title      TEXT    NOT NULL,
  body       TEXT    NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX ux_search_doc ON search_doc(kind, owner_id);

CREATE VIRTUAL TABLE search_doc_fts USING fts5(
  title, body,
  tokenize='trigram'
);

CREATE VIEW v_feed AS
  SELECT 'memory'     AS etype, id, uid, scope, scope_ref, title,
         status       AS state, created_at AS ts
    FROM memory
  UNION ALL
  SELECT 'experience' AS etype, id, uid, scope, scope_ref, title,
         trust        AS state, t_created  AS ts
    FROM experience;

INSERT INTO meta(k, v) VALUES ('schema_version', '1');
INSERT INTO meta(k, v) VALUES ('fts_mode', 'trigram');
