-- P0b: feedback ledger for mem_feedback

CREATE TABLE IF NOT EXISTS feedback (
  id         INTEGER PRIMARY KEY,
  owner_type TEXT    NOT NULL CHECK (owner_type IN ('memory','experience')),
  owner_id   INTEGER NOT NULL,
  owner_uid  TEXT    NOT NULL,
  outcome    TEXT    NOT NULL CHECK (outcome IN ('helpful','wrong','stale')),
  note       TEXT,
  actor_id   INTEGER REFERENCES app_user(id),
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_feedback_owner
  ON feedback(owner_type, owner_id, outcome, created_at);

UPDATE meta SET v = '2' WHERE k = 'schema_version';
