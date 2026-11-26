-- rocky-mem 0004: conflict adjudication + cross-entity links (PostgreSQL dialect).
--
-- A conflict is a question, not an answer: two experiences share a symptom_key
-- but propose different fixes. `contested` records that question; this table
-- records the human answer, append-only, so a group can be re-opened later.
--
-- `link` carries the durable relation between entities (complements for
-- condition-split pairs, merged_from, derived_from). It is deliberately generic
-- and NOT a replacement for evidence.

CREATE TABLE conflict_resolution (
  id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  group_id     TEXT   NOT NULL,
  action       TEXT   NOT NULL
                 CHECK (action IN ('adopt','split','reject_both','defer','reopen')),
  kept_id      BIGINT REFERENCES experience(id),
  note         TEXT   NOT NULL,
  actor_id     BIGINT REFERENCES app_user(id),
  review_after BIGINT,
  created_at   BIGINT NOT NULL
);

CREATE INDEX idx_resolution_group ON conflict_resolution(group_id, created_at);

CREATE TABLE link (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  src_type   TEXT   NOT NULL CHECK (src_type IN ('memory','experience','source','chunk')),
  src_id     BIGINT NOT NULL,
  dst_type   TEXT   NOT NULL CHECK (dst_type IN ('memory','experience','source','chunk')),
  dst_id     BIGINT NOT NULL,
  relation   TEXT   NOT NULL
               CHECK (relation IN ('complements','contradicts','merged_from','derived_from','relates')),
  note       TEXT,
  created_at BIGINT NOT NULL
);

CREATE UNIQUE INDEX ux_link ON link(src_type, src_id, dst_type, dst_id, relation);
CREATE INDEX idx_link_dst ON link(dst_type, dst_id);

UPDATE meta SET v = '4' WHERE k = 'schema_version';
