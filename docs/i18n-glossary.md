# i18n glossary

Canonical translations for domain vocabulary. Use these, do not invent synonyms:
inconsistent terminology is the most visible quality defect in a translated UI,
and it also breaks search inside the docs.

## 1. Domain terms

| English | zh-CN | Notes |
|---|---|---|
| entry | 条目 | Generic for a row in `memory` or `experience`. Prefer the specific term when the distinction matters. |
| memory | 记忆 | The state table. Never "内存" (that is RAM). |
| experience | 经验 | The ledger table. |
| evidence | 证据 | A pointer to a `raw` row. |
| raw / raw record | 原始记录 | The immutable evidence layer. |
| scope | 范围 | `global` / `team` / `project` / `session`. |
| scope ref | 范围标识 | The repo path or team id that a scope points at. |
| trust | 可信度 | The experience lifecycle state. Distinct from `confidence`. |
| confidence | 置信度 | The author's self-assessed certainty, 0–1. |
| proposed | 待审 | Not yet verified. |
| active | 生效 | Current and trusted. |
| contested | 有争议 | Conflicting entries exist. |
| stale | 已过时 | Not used or verified recently. |
| falsified | 已证伪 | Demonstrated wrong. |
| archived | 已归档 | Retained but out of the active set. |
| superseded | 已被取代 | A newer version of the same memory exists. |
| lineage | 版本链 | The stable identity of a memory across edits. |
| review queue | 审核队列 | |
| context pack | 上下文包 | What `mem_search` returns. Never "上下文包列表". |
| provenance | 出处 | |
| sensitivity | 敏感级别 | L1–L4. |
| redaction | 脱敏 | |
| ingest | 采集 / 导入 | "采集" for automated capture, "导入" for a file drop. |
| symptom | 现象 | The observable failure. |
| root cause | 根因 | |
| fix | 解法 | |
| applies when | 适用条件 | |
| invalid when | 失效条件 | Not "不适用条件"; shorter and matches the field name. |
| work unit (WU) | 工作单元 | P2. |
| corpus | 语料 | Dropped files, as opposed to knowledge. |
| chunk | 切片 | A passage of a source document. |
| source | 来源文件 | |
| feedback | 反馈 | |
| timeline | 时间线 | |

## 2. UI actions

| English | zh-CN |
|---|---|
| Approve | 通过 |
| Reject | 拒绝 |
| Edit | 编辑 |
| Move | 移动 |
| Merge evidence | 合并证据 |
| Mark stale | 标记过时 |
| Purge | 清除 |
| Export | 导出 |
| Reindex | 重建索引 |
| Drop files here | 拖放文件到此处 |
| Nothing waiting for review | 没有待审核的条目 |

## 3. Rules

1. **Do not translate:** tool names (`mem_search`), parameter names, error codes,
   file paths, command lines, enum codes, tag values, product names.
2. **Do not translate technical identifiers** even when a Chinese equivalent
   exists — `commit`, `Pod`, `ConfigMap`, `rollout` stay as-is in Chinese text.
3. **Keep the English term in parentheses** on first use in long-form docs when
   the Chinese term is ambiguous: 上下文包（context pack）.
4. **Tone:** concise, imperative for actions (通过, not 点击通过按钮), neutral for
   states. No exclamation marks, no "请" in button labels.
5. **Punctuation:** Chinese UI uses full-width punctuation; do not append a
   full-width period to short labels or button text.
6. **Numbers:** keep Arabic numerals in both languages.

## 4. Ambiguities resolved here, deliberately

| Term | Rejected alternative | Why |
|---|---|---|
| `memory` → 记忆 | 内存 | 内存 means RAM. A memory system called 内存库 is confusing to any developer. |
| `experience` → 经验 | 知识 | "Knowledge" is the umbrella; experience is specifically the problem→fix ledger. |
| `trust` → 可信度 | 信任度 | 信任度 reads as interpersonal trust; 可信度 is a property of the record. |
| `context pack` → 上下文包 | 上下文片段 | It is a single assembled artifact with a budget, not a collection of fragments. |
| `stale` → 已过时 | 陈旧 | 陈旧 reads as an insult about quality; 已过时 is a state. |
