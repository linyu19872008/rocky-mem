# 数据模型

| | |
|---|---|
| 状态 | 与迁移 `0001`–`0004` 逐对象一致（CI 校验） |
| 适用 | SQLite（单人 / 小团队）与 PostgreSQL（团队） |
| 相关 | [ADR-0002](adr/0002-split-memory-and-experience-tables.md)（拆表）、[ADR-0006](adr/0006-two-database-strategy.md)（双库策略）、[ADR-0008](adr/0008-corpus-layer.md)（语料层）、[ADR-0009](adr/0009-conflict-adjudication.md)（冲突裁定） |

---

## 1. 约定

| 主题 | 规则 |
|---|---|
| 类型 | 只用整数、文本、浮点、二进制；不用方言特有类型。时间一律 Unix 纪元秒 |
| 时间 | `INTEGER` / `BIGINT`，UTC。格式化在客户端用 `Intl.*` 完成 |
| 布尔 | SQLite 用 `INTEGER` 0/1 + `CHECK`；PostgreSQL 用 `BOOLEAN` |
| JSON | 存 `TEXT`，且**只存永不用于过滤或 join 的附属信息** |
| 标识 | 内部 `id` 整数主键；对外 `uid` 文本唯一 |
| `uid` 格式 | `<前缀>_<ULID>`：`r_` 原始、`m_` 记忆、`e_` 经验、`s_` 来源、`c_` 切片。字典序即创建序 |
| 枚举 | 存英文码，用 `CHECK` 约束。**展示文案永不入库** |
| 文本归一化 | 一切文本在 hash 与入库前做 `NFC` 归一化 |
| 保留字 | 用户表叫 `app_user`（`user` 是 PostgreSQL 保留字）；成员角色列叫 `member_role`；`mem_key` 不叫 `key` |
| `scope_ref` | 只取四种值：`global` → `''`；`team` → `team.id`（**预留，当前写入后永远检索不到**）；`project` → `project.id`（十进制字符串）；`session` → 会话 id。**永不存文件路径** |

### 1.1 为什么内部 `id` 和对外 `uid` 都要有

检索索引需要一个稳定的整数 rowid 与之关联；而 ULID 是对外、可移植、可按时间排序的句柄，用于引用、导出和 API。两列加起来每行约 20 字节，换来的是「换数据库时不破坏任何外部引用」。

---

## 2. SQLite

迁移文件（当前 `schema_version = 4`）：

| 文件 | 内容 |
|---|---|
| `db/migrations/sqlite/0001_init.sql` | 基础表、索引与视图 |
| `db/migrations/sqlite/0002_feedback.sql` | `feedback` 表（`mem_feedback` 用的反馈台账） |
| `db/migrations/sqlite/0003_corpus.sql` | `source` / `chunk` 语料层（见 §2.7） |
| `db/migrations/sqlite/0004_conflict.sql` | `conflict_resolution` / `link`（见 §2.9） |

连接建立时执行：

```sql
PRAGMA journal_mode  = WAL;
PRAGMA synchronous   = NORMAL;
PRAGMA foreign_keys  = ON;
PRAGMA busy_timeout  = 5000;
PRAGMA temp_store    = MEMORY;
PRAGMA trusted_schema = OFF;
```

启动探测：

```sql
SELECT sqlite_version();                                  -- 需 >= 3.35（trigram 自 3.34，RETURNING 自 3.35）
SELECT sqlite_compileoption_used('ENABLE_FTS5');          -- 需为 1
```

任一不满足时进入降级模式：`LIKE` 检索、界面横幅提示、`doctor` 报 `ERR_FTS_UNAVAILABLE`。**不静默降级。**

### 2.1 元信息与身份

```sql
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

-- 单人模式的本地用户没有邮箱；团队模式下同一邮箱不应重复。
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
```

单人模式 `init` 时种一行 `app_user(1,'local')`，其余表留空。

> **这四张是「预留未接线」表。** 0.x 只做单人模式（`DESIGN.md` / [ADR-0007](adr/0007-solo-only-scope.md)），代码**不读也不写** `team` / `membership` / `api_token`。保留它们是为了避免日后做破坏性迁移，不代表功能存在。同样地，`setting` 与 `idempotency` 两张表也已建但未接线（locale 存 cookie，幂等靠内容 hash）。看到表就以为能力在，是这套 schema 最容易踩的误读。

### 2.2 项目（替代本地路径）

```sql
CREATE TABLE project (
  id              INTEGER PRIMARY KEY,
  name            TEXT    NOT NULL,
  remote_url_norm TEXT    NOT NULL DEFAULT '',   -- 无 remote 时为空串
  root_hint       TEXT,                          -- 最近一次见到的本地路径，仅供单人模式显示
  created_by      INTEGER REFERENCES app_user(id),
  created_at      INTEGER NOT NULL
);
CREATE UNIQUE INDEX ux_project_remote
  ON project(remote_url_norm) WHERE remote_url_norm <> '';
```

`remote_url_norm` 归一化规则（必须完全一致，否则会造出重复项目）：

1. 去掉 `user:pass@` 形式的凭据
2. `https://host/a/b.git`、`git@host:a/b.git`、`ssh://git@host/a/b` 统一为 `host/a/b`
3. 去掉尾部 `.git` 与尾部 `/`
4. `host` 小写；`owner/repo` **保持原大小写**（GitHub 上大小写有别，强转会撞名）
5. 无 remote 的本地仓库：`remote_url_norm = ''`，`root_hint` 存绝对路径，靠 `root_hint` 反查

### 2.3 证据层

```sql
CREATE TABLE raw (
  id               INTEGER PRIMARY KEY,
  uid              TEXT    NOT NULL UNIQUE,
  hash             TEXT    NOT NULL UNIQUE,   -- sha256(NFC(body))
  channel          TEXT    NOT NULL
                     CHECK (channel IN ('manual','agent','file','git','transcript','web')),
  source_ref       TEXT,                      -- 会话 id / commit sha / 路径 / url
  actor_id         INTEGER REFERENCES app_user(id),
  agent            TEXT,                      -- 'cursor' | 'dsh' | 'human' | ...
  session_id       TEXT,
  cwd              TEXT,
  project_id       INTEGER REFERENCES project(id),
  mime             TEXT,
  lang             TEXT    NOT NULL DEFAULT 'und',
  body             BLOB    NOT NULL,
  redaction        TEXT,                      -- JSON：命中规则与替换计数
  rulepack_version TEXT,
  sensitivity      INTEGER NOT NULL DEFAULT 2
                     CHECK (sensitivity BETWEEN 1 AND 3),
  created_at       INTEGER NOT NULL
);

CREATE INDEX idx_raw_time    ON raw(created_at);
CREATE INDEX idx_raw_project ON raw(project_id, created_at);
```

要点：

- `hash UNIQUE` 是去重机制，不是优化。它让每次写入天然幂等。
- `sensitivity` 永不取 4。L4 内容在入库前被拒。
- `body` 用 `BLOB`，压缩负载与非 UTF-8 摘录都不需要改 schema；文本按 UTF-8 字节存。

### 2.4 记忆（状态表）

```sql
CREATE TABLE memory (
  id            INTEGER PRIMARY KEY,
  uid           TEXT    NOT NULL UNIQUE,
  lineage       TEXT    NOT NULL,              -- 跨版本稳定
  scope         TEXT    NOT NULL
                  CHECK (scope IN ('global','team','project','session')),
  scope_ref     TEXT    NOT NULL DEFAULT '',   -- project 范围存 project_id 的十进制字符串
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
```

要点：

- 那条部分唯一索引就是这张表存在的理由。它把「一个 key 只有一个当前值」变成数据库保证，而不是约定。
- `lineage` 跨编辑不变。外部引用（别的条目提到它、agent 引用它）指向 lineage，`get` 解析到当前 `active` 行。没有它，每次编辑都会让历史引用失效。
- 编辑 = 插新行（新 id、同 lineage）+ 旧行转 `superseded`。只有 `status`、`superseded_by`、`t_last_used`、`use_count`、`rev` 会被原地更新。
- **记忆的出处只有一个机制：`evidence` 表（`owner_type='memory'`）。** 这里刻意没有再放一个 `origin_raw_id`——两套出处机制必然有一套会过期。记忆的 `evidence` 可以为 0 条（人手写的一句话常常没有来源），但有就必须指向真实 `raw`。

**编辑（supersede）的顺序有陷阱**，见 §5.5；写反了会遇到唯一索引或外键错误。

### 2.5 经验（台账）

```sql
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
  fix_hash       TEXT,                         -- sha256(normalize(fix_md))

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
```

`exp_type` 与字段的映射（避免为每种类型加列）：

| exp_type | symptom | root_cause | fix_md | applies_when | invalid_when |
|---|---|---|---|---|---|
| `howto` | 要达成什么 | 为什么这么做 | 步骤 | 前提条件 | 何时失效 |
| `pitfall` | 报错/异常 | 真因 | 规避做法 | 触发条件 | 何时无害 |
| `decision` | 当时的约束 | 备选与取舍 | 最终决定 | 有效期 | 推翻条件 |
| `postmortem` | 故障现象 | 因果链 | 改进项 | 适用架构 | 失效场景 |

刻意没有 `body_md`：结构化字段才是上下文包能分段、`invalid_when` 能单独被检索的原因。长篇叙事属于 `source`/`chunk`。

### 2.6 证据、标签、审计、设置、幂等、反馈

```sql
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
  action      TEXT    NOT NULL,   -- read|write|update|verify|reject|purge|export
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

-- 迁移 0002：mem_feedback 的反馈台账
CREATE TABLE feedback (
  id         INTEGER PRIMARY KEY,
  owner_type TEXT    NOT NULL CHECK (owner_type IN ('memory','experience')),
  owner_id   INTEGER NOT NULL,
  owner_uid  TEXT    NOT NULL,          -- 冗余：uid 会在 supersede 后变化，留原值便于人读日志
  outcome    TEXT    NOT NULL CHECK (outcome IN ('helpful','wrong','stale')),
  note       TEXT,
  actor_id   INTEGER REFERENCES app_user(id),
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_feedback_owner
  ON feedback(owner_type, owner_id, outcome, created_at);
```

`owner_id` 是多态引用，数据库层面无法建外键。代价由三条承担：唯一写入路径、`purge` 在同一事务里按 `(owner_type, owner_id)` 删除、`doctor` 巡检孤儿。反方向由 `raw_id` 的 `RESTRICT` 保证。

`audit` 记录**读操作**，不只是写。三个用途：合规追溯、`use_count` 的数据源、回答「你凭什么给我这条」。

`feedback` 记的是**一次判定**而不是当前状态：`outcome=wrong` 会在同一步把经验转 `contested`，但历史反馈全部保留。这样「这条被三个人说过不对」是可查的，而「它现在是什么状态」只看 `experience.trust`。

### 2.7 语料层：`source` / `chunk`（迁移 `0003`）

语料是**背景材料，不是证据**：不写 `raw`，也不建 `evidence`。理由见 [ADR-0008](adr/0008-corpus-layer.md)。

```sql
CREATE TABLE source (
  id            INTEGER PRIMARY KEY,
  uid           TEXT    NOT NULL UNIQUE,          -- s_<ULID>
  kind          TEXT    NOT NULL CHECK (kind IN ('file','paste')),
  uri           TEXT    NOT NULL,                 -- 文件名或 "paste"
  title         TEXT    NOT NULL,
  format        TEXT    NOT NULL,                 -- plain|docx|xlsx|pdf
  text_hash     TEXT    NOT NULL,                 -- sha256(NFC(脱敏后全文))，去重键
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
  uid         TEXT    NOT NULL UNIQUE,            -- c_<ULID>
  source_id   INTEGER NOT NULL REFERENCES source(id) ON DELETE CASCADE,
  ord         INTEGER NOT NULL,
  heading     TEXT,                               -- 仅 markdown 填
  text        TEXT    NOT NULL,                   -- 已脱敏的切片，自带副本
  char_count  INTEGER NOT NULL,
  sensitivity INTEGER NOT NULL DEFAULT 2 CHECK (sensitivity BETWEEN 1 AND 3)
);

CREATE UNIQUE INDEX ux_chunk_ord ON chunk(source_id, ord);
CREATE INDEX idx_chunk_source ON chunk(source_id);
```

三个要点：

- **`text_hash` 是去重键**，取「脱敏后全文」而非文件字节。同一文档另存一次不会产生第二份；命中既有 hash 时返回 `unchanged`，不写新行。
- **`chunk.text` 存自己的副本**，不存指向 `raw` 的偏移。SQLite 的 `substr` 对 BLOB 按字节、对 TEXT 按字符，PostgreSQL 按字符——中文下差三倍，算错不报错，只是乱码。
- **`status='archived'` 是语料的版本机制**。同一 `project_id` + `uri` 重新投喂且 `text_hash` 变化时，旧 `source` 转 `archived`，**并同时删除它全部 chunk 的 `search_doc` 行**（见 §5.4）。

### 2.8 派生层：检索文档

```sql
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
```

刻意**不用 external-content 模式**：外部内容表在更新时必须先按旧值写入 `'delete'` 命令行，容易写错且要求先读旧值。独立 FTS 表可以直接 `DELETE WHERE rowid=?`。代价是文本存两份——这本来就是派生层的正常代价（PostgreSQL 侧的生成列同样会存一份）。

`kind='chunk'` 自迁移 `0003` 起有写入方：投喂确认时为每个 chunk 写一行，`owner_id` 指向 `chunk.id`，`title` 带源标题（便于「找那份文档」命中全部片段）。`kinds: ["source"]` 仍返回 `501`——文档是容器，只有片段进入索引。

```sql
CREATE VIEW v_feed AS
  SELECT 'memory'     AS etype, id, uid, scope, scope_ref, title,
         status       AS state, created_at AS ts
    FROM memory
  UNION ALL
  SELECT 'experience' AS etype, id, uid, scope, scope_ref, title,
         trust        AS state, t_created  AS ts
    FROM experience;
```

### 2.9 冲突裁定与关联（迁移 `0004`）

冲突检测（`symptom_key` 相同、`fix_hash` 不同）只能提出问题。`contested` 是问题，`conflict_resolution` 是人的回答。理由见 [ADR-0009](adr/0009-conflict-adjudication.md)。

```sql
CREATE TABLE conflict_resolution (
  id           INTEGER PRIMARY KEY,
  group_id     TEXT    NOT NULL,              -- 指向 experience.conflict_group
  action       TEXT    NOT NULL
                 CHECK (action IN ('adopt','split','reject_both','defer','reopen')),
  kept_id      INTEGER REFERENCES experience(id),   -- adopt 的胜者
  note         TEXT    NOT NULL,              -- 必填：没有理由的裁定就是猜
  actor_id     INTEGER REFERENCES app_user(id),
  review_after INTEGER,                       -- defer 用
  created_at   INTEGER NOT NULL
);
CREATE INDEX idx_resolution_group ON conflict_resolution(group_id, created_at);

CREATE TABLE link (
  id         INTEGER PRIMARY KEY,
  src_type   TEXT    NOT NULL CHECK (src_type IN ('memory','experience','source','chunk')),
  src_id     INTEGER NOT NULL,
  dst_type   TEXT    NOT NULL CHECK (dst_type IN ('memory','experience','source','chunk')),
  dst_id     INTEGER NOT NULL,
  relation   TEXT    NOT NULL
               CHECK (relation IN ('complements','contradicts','merged_from','derived_from','relates')),
  note       TEXT,
  created_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX ux_link ON link(src_type, src_id, dst_type, dst_id, relation);
CREATE INDEX idx_link_dst ON link(dst_type, dst_id);
```

四个要点：

- **`conflict_resolution` 只追加。** 同一组可以裁多次（复开后再裁）；当前结论 = `ORDER BY created_at DESC LIMIT 1`。不要在条目上存「已裁定」字段——裁定是组级事实，挂到条目上会重复且不一致。
- **`adopt` 的败者写 `falsified`，不是 `archived`。** `archived` 表示「不再流通（重复/过时）」，`falsified` 表示「被证明是错的」。两者都必须删索引。
- **`split` 要写回条目。** 除了插一行裁定，还要更新两条的 `applies_when` / `invalid_when` 并重写 `search_doc`，否则条件只活在裁定记录里，检索用不上。
- **`link` 是关系，不是证据。** `split` 写双向 `complements`；`merged_from` / `derived_from` 预留给后续。唯一索引保证同一对同类关系只有一行。


---

## 3. PostgreSQL

PostgreSQL 版由 SQLite 版按下列规则派生。**两套迁移文件都是手写的**，CI 用一致性套件保证语义一致——不做自动生成，以免一种方言的写法被机械复制到另一种而不经评审。

### 3.1 类型替换

| SQLite | PostgreSQL |
|---|---|
| `INTEGER PRIMARY KEY` | `BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY` |
| `INTEGER`（时间、计数） | `BIGINT` |
| `BLOB` | `BYTEA` |
| `INTEGER … CHECK (x IN (0,1))` | `BOOLEAN` |
| `REAL` | `DOUBLE PRECISION` |
| `… WITHOUT ROWID` | 删除该子句 |

其余照搬：`TEXT`、`CHECK` 约束、部分唯一索引、`ON DELETE RESTRICT/CASCADE`、`UNION ALL` 视图，两边语法一致。

`0002_feedback` 同样两边各一份：PostgreSQL 版只需把 `INTEGER PRIMARY KEY` 换成 `BIGINT GENERATED ALWAYS AS IDENTITY`、其余 `INTEGER` 换 `BIGINT`。两个方言的迁移编号必须一一对应，否则一致性套件会在 CI 里报错。

### 3.2 检索层替换

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE search_doc (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  kind       TEXT   NOT NULL CHECK (kind IN ('memory','experience','chunk')),
  owner_id   BIGINT NOT NULL,
  lang       TEXT   NOT NULL DEFAULT 'und',
  title      TEXT   NOT NULL,
  body       TEXT   NOT NULL,
  search_text TEXT GENERATED ALWAYS AS (title || ' ' || body) STORED,
  updated_at BIGINT NOT NULL
);
CREATE UNIQUE INDEX ux_search_doc ON search_doc(kind, owner_id);
CREATE INDEX idx_search_doc_trgm ON search_doc USING gin (search_text gin_trgm_ops);
```

没有独立的 FTS 表：生成列 + GIN 索引由数据库自己维护，写入只需要 upsert `search_doc` 一行。

### 3.3 可选的中文增强

```sql
-- 仅在实例已安装 zhparser 时执行；由 doctor 探测，不由迁移脚本强建
CREATE TEXT SEARCH CONFIGURATION zh (PARSER = zhparser);
ALTER TEXT SEARCH CONFIGURATION zh ADD MAPPING FOR n,v,a,i,e,l WITH simple;
```

检测到后，2 字以上中文查询额外走 `tsvector` 路径与 `ILIKE` 结果合并；否则只用 `ILIKE`。

---

## 4. 差异对照总表

| 能力 | SQLite | PostgreSQL | 影响 |
|---|---|---|---|
| 检索索引 | FTS5 `trigram` | `pg_trgm` GIN | 语义一致（都是子串匹配），打分不同 |
| 检索写入 | 三步（见 §5） | 一步（生成列） | 见 §5 |
| 2 字中文 | `LIKE` 兜底，毫秒级 | `LIKE`，5 万行以内可接受，更大需 `zhparser` | 已记录为限制 |
| 大小写 | `trigram` 与 `LIKE` 对 ASCII 均不敏感 | `ILIKE` 显式不敏感 | 查询构造器按方言生成 |
| 备份 | `VACUUM INTO` | `pg_dump` / PITR | 运维文档分开写 |
| 并发 | 单写者 | 连接池 + 乐观锁 + advisory lock | 见 `DESIGN.md` §7.3 |
| 保留字 | 宽松 | 严格（故 `app_user`、`member_role`） | 命名规则统一 |

---

## 5. 检索文档的写入契约

这是全库唯一「多个后端写法不同、且写错不会立刻报错」的地方。必须逐字实现。

### 5.1 SQLite：三步

```sql
-- 与业务写在同一个事务内。参数用命名形式（:kind / :owner_id / :lang / :title
-- / :body / :now），避免位置参数在实现里错位。

-- 1) 查既有 id（可能为空）
SELECT id FROM search_doc WHERE kind = :kind AND owner_id = :owner_id;

-- 2a) 有：先删索引项，再更新注册表
DELETE FROM search_doc_fts WHERE rowid = :id;

UPDATE search_doc
   SET lang = :lang, title = :title, body = :body, updated_at = :now
 WHERE id = :id;

-- 2b) 无：插入注册表并取回 id（SQLite >= 3.35 支持 RETURNING）
INSERT INTO search_doc(kind, owner_id, lang, title, body, updated_at)
VALUES (:kind, :owner_id, :lang, :title, :body, :now)
RETURNING id;

-- 3) 写入索引项，rowid 必须等于 search_doc.id
INSERT INTO search_doc_fts(rowid, title, body) VALUES (:id, :title, :body);
```

**为什么不用触发器**：触发器无法跨方言复用（PostgreSQL 侧用生成列 + GIN，机制完全不同），且会把「派生层可整体重建」这件事藏进数据库。改用应用层单点写入 + `reindex` + `doctor` 对账。

**为什么不用 external-content**：那样更新时必须先按**旧值**写 `'delete'` 命令行，对顺序和列序都敏感。独立表只要 rowid。

### 5.2 PostgreSQL：一步

```sql
INSERT INTO search_doc(kind, owner_id, lang, title, body, updated_at)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (kind, owner_id) DO UPDATE
  SET lang = excluded.lang,
      title = excluded.title,
      body = excluded.body,
      updated_at = excluded.updated_at;
```

生成列与 GIN 索引由数据库维护，无第二步。

### 5.3 一致性保障

| 手段 | 作用 |
|---|---|
| 唯一写入函数 `WriteSearchDoc()` / `DeleteSearchDoc()` | 所有写路径只能走它们 |
| `doctor` 行数对账 | 见 §5.4 的对账公式 |
| `reindex` | `DELETE` 全部 `search_doc` 后按业务表重建（含 FTS） |
| `store/conformance` | 同一套用例在两个后端上跑，断言结果一致 |

### 5.4 生命周期：什么时候必须删索引

**这一节是 §5 里最容易漏掉的一段。** 只写不删不会报错，只会让检索返回已经失效的条目——测试里几乎发现不了，上线后才被用户看见。

被索引的集合只有三类行：

```sql
-- 记忆：只有当前版本进索引，superseded 不进
SELECT id FROM memory WHERE status = 'active';
-- 经验：归档的不进索引
SELECT id FROM experience WHERE trust <> 'archived';
-- 语料：只有 active 文档的片段进索引
SELECT c.id FROM chunk c JOIN source s ON s.id = c.source_id WHERE s.status = 'active';
```

因此下列事件必须调用 `DeleteSearchDoc(kind, owner_id)`：

| 事件 | 动作 |
|---|---|
| 记忆被编辑，旧行转 `superseded` | 删除旧行的 `search_doc` 与 FTS 行 |
| 记忆或经验被 `archived` / `falsified` | 删除（裁定为 `falsified` 的也必须删，否则 `doctor` 报漂移） |
| `purge` | 在同一事务里删除业务行、`evidence`、`tag`、`search_doc` |
| `source` 转 `archived`（同 `uri` 重投喂） | **删除该文档全部 chunk 的 `search_doc`**——否则旧版本会继续参与检索 |
| `source` 被删 | chunk 级联删除（`source_id` 上是 `ON DELETE CASCADE`），其 `search_doc` 同事务删除 |

对账公式（`doctor` 每次运行都校验）：

```
count(search_doc WHERE kind='memory')     == count(memory     WHERE status='active')
count(search_doc WHERE kind='experience') == count(experience WHERE trust  <> 'archived')
count(search_doc WHERE kind='chunk')      == count(chunk JOIN source WHERE source.status='active')
```

两边不等，说明某条路径绕过了写入/删除函数，或索引损坏。

`symptom` 被改写时 `symptom_key` 必须重算——它是派生值，不是历史事实；重算后可能触发新的冲突簇（见 `DESIGN.md` §7.2）。

### 5.5 记忆编辑（supersede）的顺序

**编辑（supersede）必须按这个顺序，不能想当然：**

```sql
-- 同一个事务内
-- 1) 先把旧行挪出 active：否则 ux_memory_current 会挡住接下来的插入
UPDATE memory
   SET status = 'superseded', rev = rev + 1
 WHERE id = :old_id AND rev = :expected_rev;   -- 影响 0 行 → ERR_WRITE_CONFLICT

-- 2) 插入新版本，得到 new_id
INSERT INTO memory(uid, lineage, scope, scope_ref, mem_key, mem_type,
                   title, body_md, lang, actor_id, rev, created_at)
VALUES (:uid, :lineage, ...)
RETURNING id;

-- 3) 回填指向
UPDATE memory SET superseded_by = :new_id WHERE id = :old_id;

-- 4) 旧版本的检索文档删除，新版本写入
```

顺序反过来的两种写法都会失败：

- 先插新行 → 撞 `ux_memory_current`（旧行还是 active）
- 先给旧行填 `superseded_by` → 撞外键（新行还不存在）

`superseded_by` 生成新 id 之前只能为 NULL，这是它必须可空的原因。

---

## 6. 这张 schema 必须支撑的查询

| 查询 | 路径 |
|---|---|
| 会话启动取常驻记忆 | `idx_memory_scope`，`status='active'`，按 `pinned DESC, priority DESC` |
| 按 key 取当前值 | `ux_memory_current`，唯一索引，单行 |
| 版本历史 | `idx_memory_lineage` |
| 经验全文检索 | `search_doc_fts MATCH ?`（SQLite）/ `search_text ILIKE`（PG），再回表 |
| 待审队列 | `idx_exp_trust`，`trust='proposed'` |
| 冲突集合 | `idx_exp_conflict` |
| 某项目的全部条目 | `idx_exp_scope` + `idx_raw_project` |
| 出处下钻 | `idx_evidence_owner` → `raw` by `id` |
| 幂等重放 | `idempotency` 主键 |

跨类型文本排序**不是**查询模式：每个类型各自有序，在装配层合并。因此 `search_doc` 虽是一张表，检索时仍按 `kind` 分组取候选。

---

## 7. 迁移

- `meta.schema_version` 为整数，每次迁移 +1。只进不退。**当前为 `4`**（`0001_init` → `0002_feedback` → `0003_corpus` → `0004_conflict`）。
- 迁移在单个事务内按序应用，**应用前自动 `VACUUM INTO` 快照**（SQLite）或提示外部备份（PG）。快照路径打印在输出里。
- 迁移文件：`db/migrations/sqlite/NNNN_*.sql`、`db/migrations/postgres/NNNN_*.sql`，编号必须一一对应。已落地的四组：`0001_init`、`0002_feedback`、`0003_corpus`、`0004_conflict`。
- 不支持降级。需要回退就恢复快照。
- 跨大版本（0.x → 1.x）提供 `export --format jsonl` + 导入工具作为逃生通道。

`meta` 的已知键：`schema_version`、`fts_mode`（`trigram` 或 `pg_trgm`，由 `doctor` 展示，用来解释当前检索路径）。

### 7.1 可移植性规则（保持数据库可换）

1. 只用整数/文本/浮点/二进制四类。
2. 时间存整数秒，不用 `timestamp` 类型。
3. `uid` 由应用生成，不依赖数据库自增语义。
4. 检索藏在 `Store.SearchCandidates()` 后面，调用方看不到 `MATCH` 或 `ILIKE`。
5. 触发器不是设计的一部分（本版本已无触发器）；派生层必须能由 `reindex` 重建。
6. 导出用 JSONL，不做裸文件拷贝。

### 7.2 容量估算（单用户一年）

| 表 | 行数 | 约 |
|---|---|---|
| `raw` | 20 000 | 300 MB（主项，可配保留期） |
| `memory` | 1 000 | 2 MB |
| `experience` | 5 000 | 20 MB |
| `evidence` | 12 000 | 2 MB |
| `search_doc` | 6 000 | 30 MB（含索引副本） |
| `audit` | 500 000 | 80 MB |

团队模式按人数线性放大；50 人年约 5～10 GB，仍在单实例 PostgreSQL 的舒适区。

---

## 8. 运维

| 作业 | 命令 | 频率 |
|---|---|---|
| 备份 | SQLite：`VACUUM INTO`；PG：`pg_dump` 或托管快照 | 每日 |
| 优化 | `PRAGMA optimize`（SQLite）/ `VACUUM (ANALYZE)`（PG） | 每周 |
| 重建索引 | `rockymem reindex` | 换 tokenizer、批量导入后 |
| 完整性 | `integrity_check` / `amcheck` + 孤儿巡检 | `doctor`，每周 |
| 状态衰减 | 180 天未用且未验证 → `trust='stale'` | 每日 |
| 审计归档 | 超过保留期的 `audit` 行导出后删除 | 每月 |

`doctor` 还要比对 `search_doc` 与业务表行数；不一致说明有人绕过了 `WriteSearchDoc()`，或索引损坏。

---

## 9. 禁止操作

| 操作 | 原因 | 替代 |
|---|---|---|
| `UPDATE raw` | `raw` 是不可变层 | 插新行 |
| 删除仍被 `evidence` 引用的 `raw` | 破坏出处 | 先删条目，或整棵子树 `purge` |
| 把展示文案写进枚举列 | 等于把本地化写进数据 | 存码，展示层翻译 |
| 为单一 `exp_type` 新增可空列 | 重演拆表前的列膨胀 | 新表，或用 `source`/`chunk` |
| 存 L4 内容 | 永不落盘 | 入库前拒绝并记录 |
| 在 SQL 里按 `title` 排序 | 无 ICU collation，中文按码位排 | 按时间或相关性；排序在客户端做 |
| 绕过 `WriteSearchDoc()` 直写 `search_doc` | 索引与业务表漂移，且不报错 | 只走唯一写入函数 |
