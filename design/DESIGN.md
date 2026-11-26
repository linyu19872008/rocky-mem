# rocky-mem — 设计文档

| | |
|---|---|
| 状态 | 设计与实现已校准（实现版本 v0.1.2） |
| 版本 | v1.2 |
| 日期 | 2026-11-26 校准并收窄范围；2026-11-24 初稿 |
| 适用 | rocky-mem `0.1.x`（预发布） |
| **范围** | **只做单人模式（Solo）。团队模式是未来规划，不在实现范围** —— [ADR-0007](adr/0007-solo-only-scope.md) |

一个本地优先的**记忆与经验系统**，面向编码 agent。

agent（Cursor、DeepSeek Harness、脚本）通过 MCP 读写；人通过 Web 界面审核、裁定冲突、投喂文件。所有数据留在本机，一个目录、一个 SQLite 文件，`rm` 即清零。

> **范围声明**：0.x 只交付单人模式。本文中凡是提到团队模式、多用户、`membership` 授权、`advisory lock`、多写者并发的地方，**都是设计意图或未来规划，不是现状**。PostgreSQL 支持保留，但定位是「更喜欢 PG 的单人用户」的替代后端，不是共享服务。

> **校准说明**：§6、§11、§13、§14 按实际代码重写，§15.1 逐条列出「设计写过但没做」，§16 按实际交付而非计划组织。

---

## 文档地图

设计层在 `design/`（本目录），契约层在 `docs/`，工程规范在仓库根。划分依据是**读者是谁**，不是重要性。

| 文档 | 位置 | 语言 | 内容 |
|---|---|---|---|
| `DESIGN.md` | `design/` | **中文** | 本文档。目标、原则、架构、机制、路线图 |
| `schema.md` | `design/` | **中文** | 双方言 DDL、索引、检索写入契约、生命周期、迁移 |
| `adr/` | `design/` | **中文** | 架构决策记录（为什么这么选） |
| `README.md` | `design/` | **中文** | 设计文档索引、阅读顺序、维护规则 |
| `mcp-tools.md` | `docs/` | 英文 | agent 侧工具契约（JSON Schema、输出、错误） |
| `error-codes.md` | `docs/` | 英文 | 错误码登记表（公开契约） |
| `i18n.md` | `docs/` | 英文 | 本地化规范与内容多语言 |
| `i18n-glossary.md` | `docs/` | 英文 | 术语对照表 |
| `README.md` / `README.zh-CN.md` | 根 | 英 / 中 | 项目门面 |
| `CONTRIBUTING.md` / `SECURITY.md` / `CHANGELOG.md` | 根 | 英文 | 开源工程规范 |

**语言规则：设计层用中文，对外契约与工程规范用英文。** 理由见 [ADR-0004](adr/0004-i18n-ui-scope-vs-content-language.md) §7。设计文档的读者是维护者，契约文档的读者是外部贡献者与第三方客户端，受众不同，语言就该不同。

**代码标识符在任何文档、任何语言里都是英文**——`mem_search`、`trust='contested'`、`ERR_WRITE_CONFLICT`。翻译标识符是缺陷。

---

## 1. 问题

编码 agent 是无状态的。每个新会话都从零开始，重新学习项目约定，重新踩同一个坑。丢掉的不是代码，是**过程**：试过什么、什么失败了、为什么这个修法有效、在什么条件下失效。

现有机制各抓一个碎片：

| 机制 | 它是什么 | 它缺什么 |
|---|---|---|
| 聊天记录 | 谁说了什么 | 不是「做了什么活」的记录 |
| agent 记忆文件 | 提炼后的结论 | 中间态，尤其是失败 |
| `git log` | 最终代码变更 | 运维、排障、架构讨论、安全处置 |
| 笔记应用 | 非结构化文本 | 检索、出处、任何可信度概念 |

rocky-mem 用一个小的、有主张的存储来补这个缺口：

> 记的不是「发生了什么」，而是**动作、结果、原因**这个三角——
> 并让这个三角能在 30 毫秒内被 agent 检索到。

## 2. 目标与非目标

### 目标

| 编号 | 目标 | 验收方式 |
|---|---|---|
| G1 | agent 一次调用即取回相关知识，且带出处 | `mem_search` 返回上下文包，每条带 `uid` 和至少一条证据引用 |
| G2 | 知识能在 agent 重启和换机后存活 | 单人模式是一个可移植的 SQLite 文件 |
| G3 | 写入有闸门，库不会烂成垃圾场 | 不满足质量闸门的经验进 `proposed`，不进 `active` |
| G4 | 全程可审计、可撤回、可按敏感级别导出 | `purge`、`export --level L1`、读写审计轨迹 |
| G5 | 零外连、零遥测 | 默认不发起任何出网请求 |
| G6 | 是一个合格的开源项目 | 代码与契约英文、CI 矩阵、ADR、SPDX 干净的依赖 |
| G7 | 单人能力做扎实，且不给未来的团队模式挖坑 | 单个二进制；存储层与身份字段留缝但不接线（[ADR-0007](adr/0007-solo-only-scope.md)） |

### 非目标

- 录屏、键盘记录、窗口标题采集。
- 托管 SaaS、多租户隔离。团队模式是**自托管**，不是我们替你运维。
- 绩效监控或「谁产出了什么」报表。一旦系统被这样用，工程师会绕开它，而有偏样本比没有数据更危险。
- 翻译用户写的内容（见 §10）。
- 通用笔记应用。这是有 schema 的知识库。
- 图谱可视化。关系存着，好看的图不是功能。
- 支持 MySQL（理由见 [ADR-0006](adr/0006-two-database-strategy.md)）。

## 3. 设计原则

1. **两种知识，两张表。** 偏好与事实是**状态**（只有一个当前值），经验是**台账**（累积陈述、可被质疑）。一张表无法同时满足两套写语义。→ [ADR-0002](adr/0002-split-memory-and-experience-tables.md)
2. **一个写入漏斗，多个入口。** agent 写入、人手写、文件投喂，走同一条脱敏、去重、闸门管道。
3. **检索返回上下文包，不是结果列表。** 有预算、有分段、带出处。给 agent 甩 20 条记录是失败模式。
4. **没有证据就不算发生。** 经验条目必须指向不可变的原始记录。无证据，不发布。
5. **核心只追加，索引可重建。** 派生层任何时候都能由 `reindex` 重建。
6. **规则优先，LLM 靠后，热路径零 LLM。** 分类、闸门、排序都是确定性规则。
7. **方言差异显式，不做万能抽象。** 两个存储后端共用一个接口，但各自的 SQL 分开维护、分开评审。
8. **失败也是资产。** 没成功的 `howto` 和成功的一样有价值，在上下文包里单列「不要这么做」。
9. **处处留退路。** 单条删除、批量清除、索引重建、无守护进程的 CLI 模式。
10. **UI 文案可翻译；标识符、枚举与内容不翻译。** 入库存英文码，展示层翻译；用户内容保持它被写下时的语言。
11. **降级必须被声明。** 任何能力不可用（FTS5、`pg_trgm`）都要在 `doctor` 和界面上可见，绝不静默。

## 4. 核心概念

### 4.1 记忆（memory）—— 状态

关于**事物现在是什么样**的陈述：约定、约束、术语、环境事实。短、稳定、会话开始时读取。

```yaml
scope: project
scope_ref: "42"            # project_id
mem_key: package_manager
mem_type: pref
title: 包管理器
body_md: "用 pnpm。不要在这个仓库里跑 npm 或 yarn。"
```

记忆支持**就地更新语义（upsert）**：同一 `(scope, scope_ref, mem_key)` 只能有一个 `active` 行。修改会新增一行（同 `lineage`），旧行转 `superseded`。历史保留，当前值唯一。

### 4.2 经验（experience）—— 台账

关于**某个问题是怎么被解决的、什么时候这招不管用**的陈述。永不覆盖。两条互相矛盾的陈述并存，双双标记 `contested`，并且一起返回。

```yaml
exp_type: pitfall
title: 滚动更新期间 ConfigMap 热更新会丢失
symptom: 灰度实例重启后配置回到旧版本
root_cause: 文件挂载的 ConfigMap 更新有约 60s 延迟，滚动更新在窗口内完成
fix_md: 给 Pod 模板加 checksum annotation 强制重建
applies_when: K8s + 文件挂载 ConfigMap + 滚动窗口短于 60s
invalid_when: 配置通过 ConfigMap API 直读而非文件挂载时不适用
outcome: success
```

`invalid_when` 是整张表里最值钱的字段。agent 很少因为不知道解法而失败，通常是**在不适用的条件下用了它**。

### 4.3 原始记录（raw）—— 证据

一切可引用的事实都是不可变的 `raw` 行：一次 git commit、一段会话摘录、一个文件切片、一句手写笔记。`evidence` 只指向 `raw`。这让「出处」只有一个实现点，`purge` 只有一个入口。

### 4.4 上下文包（context pack）

`mem_search` 的返回物。固定英文小标题的 Markdown 文档，在 token 预算内装配：

```markdown
## Preferences
- pnpm；绝不用 npm（project · pinned · m_01J8…）

## Applies
### [pitfall] 滚动更新期间 ConfigMap 热更新会丢失
- Applies when: K8s + 文件挂载 ConfigMap + 滚动窗口短于 60s
- Do: 给 Pod 模板加 checksum annotation 强制重建
- Do not apply when: 配置通过 API 直读而非文件挂载
- Evidence: commit abc1234 (L2 · active · confidence 0.85 · e_01J9…)

## Do not use
- 只跑 `kubectl rollout restart`：配置照样丢（试过，失败 · e_01JA…）

## Background
- 《灰度发布复盘.md》§3.2「ConfigMap 60s 延迟」(c_01J7…)
```

小标题英文且恒定，三个理由：prompt 缓存有效、输出在不同 agent 与语言下可复现、评测可比。正文保持原语言。

## 5. 架构

```
┌────────────────────────────── 接入面 ───────────────────────────────┐
│   MCP (stdio / HTTP)      HTTP /api/v1      CLI      Web (HTML+SSE) │
└────────────────────────────────┬───────────────────────────────────┘
                                 │  同一个 Core API，进程内调用
┌────────────────────────────────▼───────────────────────────────────┐
│  写入漏斗 ingest                                                    │
│   入口：agent 提案 · 人手写 · 文件投喂 · 导入(git/WU)                │
│   闸门：① 去重(hash) ② 脱敏+分级 ③ 分诊 ④ 质量闸门                  │
│   出口：published | pending_review | rejected(带原因)               │
└────────────────────────────────┬───────────────────────────────────┘
┌────────────────────────────────▼───────────────────────────────────┐
│  读取管道 retrieve                                                  │
│   query → 候选集(token 化 + 双后端检索) → 逐类型规则重排            │
│         → 分段装配上下文包（token 预算内）                          │
└────────────────────────────────┬───────────────────────────────────┘
┌────────────────────────────────▼───────────────────────────────────┐
│  存储（Store 接口，两个实现）                                        │
│  不变层   raw（只追加，hash 寻址）                                   │
│  事实层   memory · experience · evidence · tag · audit · project    │
│  派生层   search_doc(+FTS/GIN) · 视图        ← 可 reindex 重建       │
└────────────────────────────────────────────────────────────────────┘
                    │                              │
          sqlite://…（单人）            postgres://…（团队）
```

### 5.1 进程模型

One binary, one daemon, one store.

- `rockymem serve` —— 守护进程，独占存储连接，对外提供 Web 与 `/api/v1`。默认绑 `127.0.0.1:7432`，可用 `--bind` 覆盖。
- `rockymem mcp --stdio` —— **瘦代理**。说 MCP，转发给守护进程。**它不直接打开数据库。**
- `rockymem <cmd>` —— 同为客户端；守护进程未运行时，进程内直连并**独占文件锁**。

发现机制（都在数据库旁边的状态目录里，`$ROCKYMEM_HOME`）：

| 文件 | 作用 |
|---|---|
| `rocky-mem.lock` | flock 独占标记 |
| `rocky-mem.addr` | 守护进程当前地址；客户端优先读它，或读 `ROCKYMEM_URL` |
| `rocky-mem.token` | Bearer token；只在缺失或为空时生成，关闭时不删 |

Token 轮换必须显式：`rockymem serve --rotate-token`（或删掉 `.token` 再启动）。轮换后旧的长连接客户端仍持有旧 token，需要重载 MCP 配置——这是实际踩过的坑，写进 `docs/tier1.md` 了。

原因：Cursor 和 DSH 各自 spawn 一个子进程。若这些子进程直连数据库，我们会得到多个写者打一个单写者数据库。单人模式用 loopback TCP 而非 unix socket，因为 Windows 没有 unix socket。

### 5.2 运行形态：只有单人模式在范围内

| | 单人（Solo）—— **已交付** | 团队（Team）—— **规划中，不实现** |
|---|---|---|
| DSN | `sqlite://<state>/rocky-mem.db` | `postgres://…`（当前仅作单人替代后端） |
| 监听 | loopback + 单枚 token 文件 | 未定义 |
| 用户 | 隐式 `app_user(1,'local')` | 未实现；三张权限表建了但代码不读 |
| 写并发 | 单写者 + 文件锁 | 未验证；advisory lock 与幂等键都没写 |
| 备份 | `VACUUM INTO` 每日快照 | 未实现（需自己 `pg_dump`） |
| 退出 | `rm -rf` 状态目录即清零 | 未定义 |

**同一个二进制、同一套命令**，`--dsn` 只切换存储后端，**不切换语义**。今天选 `postgres://` 得到的仍然是一个单用户库，只是数据放在 PG 里。

`doctor` 会打印当前后端与能力探测结果（FTS5 / `pg_trgm` / daemon 健康度）。

### 5.3 接入面

| 面 | 受众 | 契约稳定性 |
|---|---|---|
| MCP 工具 | agent | 稳定，语义化版本，英文标识符 |
| `HTTP /api/v1` | 脚本、Web、第三方 | 稳定，带版本前缀 |
| CLI | 人、shell | 命令稳定，人类可读输出不保证 |
| Web | 人 | 无兼容承诺 |

## 6. 数据模型

完整 DDL 与双方言差异见 [schema.md](schema.md)。摘要（状态按 v0.1.2 实际迁移）。

状态含义：**已实现** ｜ **预留**（表在、代码不读，属于未来团队模式，见 [ADR-0007](adr/0007-solo-only-scope.md)）｜ **未实现**（在当前单人范围内，是真缺口）。

| 表 | 层 | 作用 | 状态 |
|---|---|---|---|
| `raw` | 不变 | 不可变证据，hash 寻址 | 已实现（`0001`） |
| `memory` | 事实 | 状态：偏好、事实、约束 | 已实现 |
| `experience` | 事实 | 台账：howto/pitfall/decision/postmortem | 已实现 |
| `evidence` | 事实 | 多态指针 → `raw` | 已实现 |
| `tag` | 事实 | 多态标签 | 已实现 |
| `audit` | 事实 | 读写清除轨迹 | 已实现 |
| `meta` | 事实 | `schema_version`、`fts_mode` | 已实现 |
| `project` | 事实 | 仓库身份解析（**替代本地路径**） | 已实现 |
| `feedback` | 事实 | `mem_feedback` 的反馈台账 | 已实现（`0002`） |
| `search_doc`(+FTS/GIN) | 派生 | 统一检索文档 | 已实现 |
| `v_feed` | 派生 | 跨类型浏览视图 | 已实现 |
| `app_user` / `team` / `membership` / `api_token` | 事实 | 身份与权限 | **预留**。只种 `app_user(1,'local')`，三张权限表无任何读写 |
| `setting` | 事实 | 服务端用户设置 | **预留（未接线）**。locale 存 cookie `rocky_mem_locale` |
| `idempotency` | 事实 | 写请求幂等键 | **预留（未接线）**。写幂等靠内容 hash 去重 |
| `source` / `chunk` | 事实 | 投喂的文档与其片段（**不是证据**） | 已实现（`0003`），见 [ADR-0008](adr/0008-corpus-layer.md) |
| `link` | 事实 | 跨实体关联 | **未实现** |
| `term` | 事实 | 双语术语表 | **未建表**；查询扩展用嵌入的 `data/seed/term-map.zh-en.yaml` |
| `wu` | 事实 | 工作单元 | **未实现** |

几条必须明说的不变量：

- **`scope_ref` 对 project 范围存的是 `project_id` 的十进制字符串**，不是本地路径。原因见 §6.1。
- `evidence.owner_id` 是多态引用，数据库层面无法建外键。代价由「唯一写入路径 + `delete` 同事务 + `doctor` 巡检」承担。
- `evidence.raw_id` 是 `ON DELETE RESTRICT`：证据绝不被静默删除。
- 每张内容行都带 `lang`（BCP-47，另有 `und` 未定、`mul` 混合）。它记录内容**是什么语言**，不是翻译键。
- 每张可变内容行都带 `rev`，用于乐观锁。

### 6.1 为什么 `scope_ref` 不能是本地路径

单用户时 `scope_ref = /Users/x/repo/service-a` 工作得很好。多人下两个同事把同一仓库克隆到不同路径，project 范围会**裂成两个互不可见的孤岛**。

因此改为：`cwd` → `git remote` 归一化 → 查/建 `project` 行 → `scope_ref = str(project.id)`。

归一化规则必须字节级一致，否则会造出重复项目：去凭据、`https`/`ssh` 统一为 `host/owner/repo`、去尾部 `.git`、`host` 小写、**`owner/repo` 保持原大小写**（GitHub 上大小写有别，强转小写会把 `Foo/Bar` 和 `foo/bar` 合并）。

`scope_ref` 的取值只有四种，不要自行扩展：

| scope | scope_ref |
|---|---|
| `global` | `''` |
| `team` | `team.id` 的十进制字符串 |
| `project` | `project.id` 的十进制字符串 |
| `session` | 会话 id |

单人模式且仓库没有 remote 时，退化为「绝对路径的哈希」建一个本地 project 行。这条退化规则必须写进文档，否则离线用户会莫名其妙搜不到东西。

## 7. 写入路径

这条漏斗只处理「知识」——记忆与经验。**投喂走另一条路**：文档直接落成语料（`source` + `chunk`），不经过这里的质量闸门，也不产生经验；只有勾选 `distill` 时才会额外走一次 `Record`。理由见 [ADR-0008](adr/0008-corpus-layer.md)。

```
        ┌──────────────────────────────────────────────────────┐
任一入口│ ① 收原始 → raw（sha256 去重，NFC 归一化）             │  共用
        │ ② 脱敏 → 判敏感级别 → 识别语言                       │  共用
        │ ③ 分诊：memory 还是 experience（规则优先）           │  共用
        └──────────────┬───────────────────────┬───────────────┘
                       │                       │
             ┌─────────▼─────────┐   ┌─────────▼────────────────────┐
             │ memory            │   │ experience                   │
             │ 按 (scope,ref,key)│   │ 质量闸门：                    │
             │ upsert → 插新行   │   │   evidence ≥ 1               │
             │ 同 lineage，旧行  │   │   ∧ applies_when 非空         │
             │ 转 superseded     │   │   ∧ confidence ≥ 0.6         │
             │ 默认 active       │   │   ∧ 无 symptom_key 冲突       │
             └───────────────────┘   │ 否则 → proposed              │
                                     └──────────────────────────────┘
```

两个分支的默认可信状态不同，这个不对称就是设计要点：记忆的写入门槛该低（写错的偏好，人两秒就能改），经验的写入门槛该高（写错的解法会污染此后所有会话）。

### 7.1 四道闸门

**① 去重。** `hash = sha256(NFC(text))`。命中既有 `raw.hash` 直接返回，写入幂等。近重复（MinHash ≥ 0.85）不拒绝，但把证据并入既有条目，`confidence` 取 max。

**② 脱敏与分级。** 在**落盘前**，不是导出时。规则包按语言分组，但**所有规则全量应用**，因为语言检测是启发式，漏报比误脱敏贵。规则包版本记录在行上供审计。

**③ 分诊。** 规则决定 memory / experience、`scope`、`lang`。

- 短祈使句、讲约定（"用 X"、"禁止 Y"）→ memory
- 含报错输出、命令、"因为/根因"→ experience

误判可由 `mem_move` 纠正，界面上支持一键改表。

`scope` 的解析分**写入**和**读取**两种场景，方向相反，不能混为一谈：

| | 写入（收窄） | 读取（并集） |
|---|---|---|
| 规则 | 显式参数 > `cwd` 解析出的 project > session | project ∪ global ∪ team（若属于某团队） |
| 失败时 | 落到 `session`，**永不推断为 `global`** | 无 |
| 理由 | 推断出的 global 条目会污染所有人；宁可只在本会话可见 | 需要的是「所有适用于我的」，宽一点无害 |

这是一处容易写错的地方：写入时收窄、读取时放宽，两个方向必须同时正确。

**④ 质量闸门。** 自动发布要求同时满足：`evidence` ≥ 1、`applies_when` 非空、`confidence ≥ 0.6`、无 `symptom_key` 冲突、`sensitivity ≤ 2`。否则进 `proposed`，出现在审核队列。

### 7.2 冲突检测

`symptom_key` 把描述同一故障的条目聚成一簇：

```
symptom_key = sha256(
    排序后的语言中立 token（报错码、路径、工具名）
    或 fallback: normalize(symptom) + "#" + lang
)
```

新条目与既有条目同 `symptom_key` 但 `fix_hash` 不同 → 两条都转 `contested`，共享 `conflict_group`。检索成对返回、相邻渲染，并显式标注「未裁定」。

**检测只能提出问题，回答由人来做。** `contested` 是问题，不是结论。五种裁定，记录追加、可复开：

| 动作 | 语义 | 状态变化 |
|---|---|---|
| `adopt` | 一条对，其余错 | 胜者 `active`，其余 **`falsified`** |
| `split` | 都对，条件不同 | 两条都 `active`，条件写回条目 |
| `reject_both` | 两条都是伪知识 | 全部 `falsified` |
| `defer` | 证据不足 | 保持 `contested`，记 `review_after` |
| `reopen` | 旧裁定被推翻 | 回到 `contested` |

三条硬约束：**理由必填**；**`split` 必须给全四条条件且两边不能相同**（写不出具体条件就 `defer`，糊一句「取决于条件」比不裁更糟）；**裁定不进 MCP**——agent 只能 `mem_feedback` 上报，不能给自己写的东西盖章。

裁定记在 `conflict_resolution`（组级、只追加），关系记在 `link`。详见 [ADR-0009](adr/0009-conflict-adjudication.md) 与 [schema.md](schema.md) §2.9。

### 7.3 并发控制：目前只承诺单人语义

当前**唯一被支持和验证的形态是单人**：一个守护进程、一个写者、`BEGIN IMMEDIATE` + 文件锁。在这套语义下 `rev` CAS 仍然有用——它挡的是「读了旧版本再写回」这类覆盖，而不是多写者竞争。

| 机制 | SQLite（当前唯一承诺） | PostgreSQL（单人替代后端） |
|---|---|---|
| 写者 | 单写者 + flock，天然串行 | 连接池，可并发——但**没有验证过** |
| 更新 | `BEGIN IMMEDIATE` + 队列 | `UPDATE … WHERE uid=? AND rev=?`，零行 → `ERR_WRITE_CONFLICT` |
| 记忆 upsert | 队列内串行，不会撞索引 | 先 supersede 再插入（§5.5），不用 `ON CONFLICT`；并发插入同 key 会撞 `ux_memory_current` |
| 冲突检测 | 无需额外手段 | 设计上要 `pg_advisory_xact_lock`——**未实现** |
| 重试 | 不需要 | 设计上要幂等键表——**表在但未接线** |

**多写者并发不是「还没做」，是「不在当前范围」**（[ADR-0007](adr/0007-solo-only-scope.md)）。若未来开做，顺序是：鉴权 → advisory lock + 幂等键 → 并发压测，**压测不能跳**。

`rev` 的语义分两种，必须区分清楚：

- **带 `rev`** = compare-and-swap。`rev` 过期即 `ERR_WRITE_CONFLICT`，调用方重新读取后再提交。
- **不带 `rev`** = last-write-wins。服务端直接 supersede 当前值，两代都留在版本链里，可追溯。

偏好类内容用 LWW 更顺手（改个包管理器不该要求先读一遍）；经验类只追加，不存在这个问题。

> 实现注记：`ERR_WRITE_CONFLICT` 在单人模式下**同样可达**——两个客户端各自读到同一版本后先后写回即可触发。它不是「团队模式专属」。

### 7.4 检索文档的生命周期

`search_doc` 是派生层，必须与业务行同生共死。**这一条漏了不会报错，只会让检索返回已经失效的记忆**——是最难在测试里发现的一类缺陷。

| 业务事件 | 对 `search_doc` 的动作 |
|---|---|
| 新建 memory / experience | upsert（见 [schema.md](schema.md) §5.1 / §5.2） |
| 编辑 memory，旧行转 `superseded` | **删除旧版本的 `search_doc` 行**（superseded 不参与检索） |
| archived / falsified | 删除 `search_doc` 行 |
| `purge` | 删除 `search_doc` 行与对应 FTS 行 |

即：**只有 `memory.status='active'` 且 `experience.trust ≠ 'archived'` 的行被索引。** 历史版本仍可通过 `lineage` 与 `mem_get` 查到，但不进入检索候选。

`doctor` 的对账公式据此为：

```
count(search_doc where kind='memory')     == count(memory     where status='active')
count(search_doc where kind='experience') == count(experience where trust  <> 'archived')
```

两边不等，说明某条写入路径绕过了 `WriteSearchDoc()` / `DeleteSearchDoc()`。

`symptom` 被编辑时，`symptom_key` 必须同步重算——它是派生值，不是历史事实。

## 8. 读取路径

```
query ──▶ 归一化（NFC、casefold、空白折叠）
      ──▶ 术语扩展（双语术语表，已实现；扩展词权重 0.7）
      ──▶ 候选集（每个类型各自有序）
            memory     top 10
            experience top 50
            chunk      top 2       （默认；显式请求时 5，每源 1–2）
      ──▶ 逐类型规则重排（不跨类型比分数）
      ──▶ 分段装配到预算内
```

**不同检索源的分数永不互相比较。** BM25 的 IDF 项依赖各自语料，800 条文档表和 30000 条文档表的分数不在同一量纲。核心层只要求候选在**单次查询内**有序，最终排序由规则决定。这是拆表的直接推论，也顺带消除了一类隐蔽的排序 bug。

经验类重排权重：

| 信号 | 权重 |
|---|---|
| `trust` | active 1.0 · contested 0.7 · proposed 0.5 · stale 0.4 · falsified 0.1 |
| scope 匹配 | project +0.30 · global +0.10 |
| project 精确匹配 | +0.25 |
| 查询串逐字出现在标题 | +0.40 |
| 30 天内被使用过 | +0.10 |
| `use_count` | 对数加成，≤ +0.20 |

预算：`mem_search` 默认 1200 token，`mem_context` 600，上限 4000。截断时包尾注明省略了几条、怎么取。

## 9. 检索层：一个接口，两个后端

这是本方案最关键的设计。完整契约见 [schema.md](schema.md) §5。

### 9.1 统一的数据形状

检索不直接读业务表，而是读一张**物化检索文档表**：

```
search_doc(id, kind, owner_id, lang, title, body, updated_at)
  唯一索引 (kind, owner_id)
```

业务写入在**同一事务**里调用 `WriteSearchDoc()`。三种好处：

1. 派生层可整体重建，`reindex` 一条路
2. 两个后端共用同一段装配与打分再归一化逻辑
3. 不再用数据库触发器维护索引（触发器无法跨方言复用）

代价：文本存两份（可接受，派生层本来就是冗余的），以及「忘了调用」的风险——由 `doctor` 行数对账覆盖。

### 9.2 两边都用 trigram，这是刻意的

| 能力 | SQLite | PostgreSQL（默认） | PostgreSQL + zhparser（可选） |
|---|---|---|---|
| 索引结构 | FTS5 `tokenize='trigram'` | `pg_trgm` GIN + `ILIKE '%x%'` | 同左，中文再走 `tsvector` |
| 英文精确子串 | ✅ | ✅ | ✅ |
| 标识符/报错码 | ✅ | ✅ | ✅ |
| 英文词干化 | ❌ | ❌ | ✅ |
| 2 字中文 | ❌ 需 `LIKE` 兜底 | ⚠️ 需 `LIKE`（大表退化为顺序扫描） | ✅ |
| 需要扩展 | 无（FTS5 内置） | `pg_trgm`（托管 PG 普遍自带） | `zhparser`（多数托管没有） |

选 `pg_trgm` 而不是 `tsvector`，是从已锁定的原则推出来的，不是图省事：

> [ADR-0004](adr/0004-i18n-ui-scope-vs-content-language.md)：选 trigram 是因为**语料性质**——记忆库里最值钱的是精确字符串（`kubectl rollout restart`、`Xid 69`、`ERR_CONN_REFUSED`、`/pkg/config/reload.go`）。词干还原反而伤害精确性。

`tsvector` 提供的恰好是我们主动拒绝的能力（词干化），却要额外的语言词典。而 `pg_trgm` 的 GIN 索引**支持前置通配符的 `ILIKE`**，语义与 trigram `MATCH` 等价。

### 9.3 打分只在一个后端内可比

- SQLite：`bm25(search_doc_fts, …)` 作粗排
- PostgreSQL：`similarity(text, query)` 作粗排

这两个数**跨后端不可比，也不需要比**——同一时刻只有一个后端在运行。适配器签名是 `SearchCandidates(ctx, query) ([]Candidate, error)`，返回 `{Kind, OwnerID, Score}`，`Score` 仅在本次调用内有意义。**这条必须写进接口注释**，否则半年后会有人拿它做跨源比较。

### 9.4 用对照测试兜住两个后端

同一份 200 条语料 + 50 个查询，在 SQLite 与 PostgreSQL 上各跑一遍，断言 `recall@10 ≥ 0.8`。

不追求完全一致（打分模型不同），但**分歧必须被测量**。这是防止两个后端悄悄漂移的唯一可靠手段。

## 10. 国际化

这是最常被做错的一节，因为两件互不相关的事共用一个名字。

| | **A. 界面本地化** | **B. 内容多语言** |
|---|---|---|
| 性质 | 有限、可穷举的固定词表 | 无限、用户生成 |
| 手段 | 消息目录 + ICU | `lang` 标记 + 跨语言检索 |
| 翻译 | 需要 | **永不执行** |
| 成本 | 天 | 周 |

规则：

- 支持 `en`、`zh-CN`，兜底 **`en`**。
- 解析顺序：`?lang=` → 服务端 `setting.locale` → `Accept-Language`（BCP-47 截断匹配）→ `en`。**在服务端解析**，避免首屏闪烁。
- 入库存英文码（`trust='contested'`），展示层翻译。API 返回 `{code, params}`，绝不返回本地化文案。
- MCP 工具名、参数名、描述、上下文包小标题永久英文。
- 用户内容永不翻译；靠术语表（已实现）与多语向量（未来）桥接。
- 文本在 hash 之前做 NFC 归一化。macOS 给的是 NFD，不做会让去重静默失效、每条存两遍。

细节见 [../docs/i18n.md](../docs/i18n.md) 与 [ADR-0004](adr/0004-i18n-ui-scope-vs-content-language.md)。

## 11. Web 界面

| 页面（路由） | 作用 | 实现状态 |
|---|---|---|
| `/` 检索 | 查询 + 过滤 | 已实现；结果可直接点进详情 |
| `/overview` 概览 | 计数、审核积压、健康度 | 已实现；可下钻到 Library |
| `/library` 浏览 | 不输入查询也能翻 | 已实现（kind/trust 过滤） |
| `/entry` 详情 | 单条全文 + 证据链 + 冲突并排 | 已实现；含 approve/reject/feedback |
| `/review` 审核 | `proposed` 的 triage | 已实现（trust 过滤、冲突分组） |
| `/ingest` 投喂 | 拖文件、粘文本 | 已实现，两阶段，见 §11.1 |
| `/remember` 写记忆 | 手写偏好 | 已实现 |
| `/record` 写经验 | 手写经验 + 证据引用 | 已实现 |
| `/settings` 设置 | doctor 快照、备份、locale/主题 | 已实现 |
| 时间线 | 按天/按项目看工作单元 | **未实现**（依赖未实现的 `wu`） |
| 设置里的 token 管理 | 多 token / 权限 | **未实现**（单人单 token） |

### 11.1 两阶段投喂

```
拖入 → ① hash 判重 → ② 解析 + 分块 → ③ 脱敏预演 → ④ 确认 → ⑤ 提交
```

确认页展示：文件类型、块数、token 总量、**将被替换的敏感片段高亮**、推断的敏感级别。允许对单块做「排除 / 提高敏感级 / 仅 L1」的覆盖（P1 再做细粒度，当前是整篇确认）。

提交后默认**只写语料**（`source` + `chunk` + 检索行），返回 `s_…` 与块数；勾选「同时生成一条待审经验」才走 `distill`，额外产生一条 `proposed` 经验。

这个页面存在的唯一理由是一个具体失败模式：「我以为脱敏了，其实没有」。人提交前看到 diff，是唯一可靠的防线。

### 11.2 无障碍与本地化

目标 WCAG 2.1 AA。全键盘可操作（审核队列就是为键盘设计的）。`<html lang>` 始终按解析结果设置。CSS 只用逻辑属性（`margin-inline-start`），让 RTL 成为配置变更而非重写。

## 12. 安全、隐私与治理

| 规则 | 实现 |
|---|---|
| 单人模式不出本机 | 绑 `127.0.0.1`，无出网请求，无遥测 |
| Web 认证 | 单人用 HttpOnly session cookie（loopback），不用手输 Bearer；MCP/CLI 仍读 token 文件 |
| 团队模式自托管 | **不在当前范围**（[ADR-0007](adr/0007-solo-only-scope.md)）。身份表保留但不接线，不提供任何多用户承诺 |
| 落盘前脱敏 | 规则包在写入漏斗执行，版本记录在行上 |
| 范围隔离 | `scope` 推断只收窄；`project_id` 写入时校验 |
| 敏感分级 | L1 公开方法论 / L2 内部 / L3 机密 / L4 永不落盘 |
| 可撤回 | `purge --uid/--before/--project`，同步执行，写审计 |
| 可审计 | 每次读也记录（查询 + 返回的 uid 列表） |
| 可导出 | `export --level L1` 只出方法论：无代码、无客户名、无内网标识 |
| 不做监控 | UI 与 API 均不提供按人统计产出的视图；审计日志仅管理员可读，且读审计本身被审计 |

L4（明文密钥、脱敏后仍存活的个人标识）永不写入：写入漏斗丢弃该行并记录拒绝原因。

**「不做绩效监控」目前是架构保证，不是政策承诺。** 单人模式的隐私来自「数据只在你机器上、只有一个用户」——没有服务端，也就没有管理员。这比任何政策都硬。

如果未来做团队模式，这条会退化成产品政策（有服务端就有管理员），届时只能靠**不实现那些视图**来兑现。这个代价提前记在这里，免得将来把它当成顺理成章的演进。

## 13. CLI

以下为 v0.1.2 实际命令（`rockymem --help` 为准）：

```
rockymem init                        初始化存储，打印路径
rockymem serve [--dsn …] [--bind 127.0.0.1:7432] [--no-web] [--rotate-token]
rockymem mcp --stdio                 agent 启动的就是这个
rockymem search "<query>"            [--json] 机器可读
rockymem show <uid>
rockymem remember                    写一条记忆（flag 或交互）
rockymem record                      写一条经验
rockymem feedback <uid> --outcome wrong|stale|helpful
rockymem review list|approve|reject
rockymem reindex                     重建检索索引
rockymem purge
rockymem export
rockymem backup [--out <file> | --dir <dir> --retain N]
rockymem restore <snapshot.db>
rockymem eval                        recall@10 检索评测
rockymem adopt-verify [--write-verdict]
rockymem doctor                      完整性、孤儿、索引漂移、环境诊断
rockymem version
```

与设计稿的差异（都是现实，不是遗漏）：

| 设计稿写过 | 实际情况 |
|---|---|
| `rockymem ingest <path>` | **没有**。投喂只走 Web `/ingest` 与 `POST /api/v1/ingest/*` |
| `rockymem term explain "<q>"` | **没有**。术语扩展在检索内部进行，未暴露调试命令 |
| `rockymem i18n check` | **没有**。目录对齐靠前端工具链，不是 CLI |
| — | **新增** `backup` / `restore` / `eval` / `adopt-verify` / `version` |

`--json` 输出是稳定的机器契约；人类可读输出不是。

## 14. 部署与平台约束

| 关注点 | 实际情况 |
|---|---|
| 分发 | `go build` / `go install` 单一二进制；另有 npm 包 `rocky-mem-skill`（技能安装器）与 `skills/rocky-mem/` 的四种宿主适配（claude-code / codex / cursor / dsh）。**brew tap 与容器镜像尚未制作** |
| 传输 | loopback TCP，默认 `127.0.0.1:7432`；Bearer token 文件（模式 600）。不用 unix socket：Windows 没有 |
| Web 认证 | HttpOnly cookie 会话；`--no-web` 可关闭 |
| 状态目录 | XDG / `~/Library/Application Support` / `%APPDATA%`，可用 `ROCKYMEM_HOME` 覆盖 |
| 配置优先级 | flag > env（`ROCKYMEM_URL` / `ROCKYMEM_HOME` / `ROCKYMEM_TEST_PG_DSN`）> 默认 |
| Go 版本 | `go.mod` 声明 `go 1.26.4` |
| SQLite | `modernc.org/sqlite`（纯 Go，免 cgo），需 FTS5 与 `trigram`；不可用则降级为 `LIKE` 并在 `doctor` 报 `ERR_FTS_UNAVAILABLE` |
| PostgreSQL | `pgx/v5` + `pg_trgm` 生成列 + GIN，已可通过 `--dsn postgres://…` 打开并跑共同的一致性套件。`zhparser` 增强**未实现** |
| 前端 | Vite 构建产物 `go:embed` 进二进制，`web/dist` 不入库 |
| 备份 | SQLite：`VACUUM INTO` + `backup`/`restore` + 每日脚本与 launchd 模板。**PostgreSQL 的 backup/restore 未实现** |
| 并发 | 见 §7.3 |
| 检索质量门 | `rockymem eval`：固定语料上 recall@10 ≥ 0.8，已进 `make verify-tier1` |

**性能数字目前是目标，不是测量结果**——仓库里没有 `bench/`。要保留下面这张表，必须先有可复现的基准；在那之前它只能当作设计约束读。

| 路径 | 目标（未测量） |
|---|---|
| 检索 p95 | SQLite 5 万文档 ≤ 30 ms；PostgreSQL 50 万文档 ≤ 50 ms |
| 写入 p95 | ≤ 20 ms / ≤ 30 ms |
| 常驻内存 | < 100 MB / < 300 MB（含连接池） |

## 15. 已知限制

1. **跨语言冲突检测只做到一半。** `symptom_key` 只有存在语言中立 token（报错码、路径、工具名）时才能跨语言归一，否则按语言隔离以避免误合并。补救手段是人工 `link`。
2. **无词干化。** trigram 匹配不到 `config` 与 `configuration` 的形态学关系。这是刻意的取舍，由术语扩展缓解。
3. **2 字中文查询不走索引。** SQLite 侧用 `LIKE` 兜底（毫秒级）；PostgreSQL 侧在 50 万行以上会退化为顺序扫描。检测到 `zhparser` 时自动升级。
4. **多态证据没有参照完整性。** 由唯一写入路径、`purge` 同事务删除、`ON DELETE RESTRICT` 与 `doctor` 共同保证。
5. **trigram 的打分偏噪**（SQLite 用 `bm25()`，PostgreSQL 用 `similarity()`），只作候选生成，不作最终排序；两者之间也不可比。
6. **不支持 MySQL。** 见 [ADR-0006](adr/0006-two-database-strategy.md)。
7. **`postmortem` 与 `pitfall` 共用字段集。** 长篇复盘应进语料（`source`/`chunk`，已实现），不要塞进 `experience` 的 `fix_md`。
8. **`raw` 按内容去重，会丢掉「第二次出现」的上下文。** 同一段报错在两个会话里被引用时，只保留首次的 `actor_id`/`session_id`。当前接受这个损失（证据内容本身不丢）；需要完整出现史时，得另加 `raw_occurrence` 表。
9. **检索文档依赖应用层双写，而非数据库约束。** `search_doc` 与业务行的一致性由 `WriteSearchDoc()`/`DeleteSearchDoc()` 保证，漏调用不会报错，只由 `doctor` 对账发现（见 §7.4）。这是为跨方言可移植付出的代价。

### 15.1 已知偏差（实现 ≠ 设计）

以下都是有意的取舍或尚未完成的缺口，写在这里是因为**读者会以为设计稿描述的就是现状**。

| # | 设计稿写过 | 实现现状 | 性质 |
|---|---|---|---|
| 1 | `term` 表（可增量写入的双语词表） | **未建表**。扩展读嵌入的 `data/seed/term-map.zh-en.yaml`，改词表要改仓库 | 取舍：换来了零运维 |
| 2 | `wu` 工作单元、时间线页 | 未实现（`link` 已随裁定落地，`wu` 仍缺） | 缺口 |
| 3 | 团队模式：多用户、`membership` 授权、advisory lock 下的并发写 | **明确不在当前范围**（[ADR-0007](adr/0007-solo-only-scope.md)）。表保留、代码不接；PostgreSQL 只是单用户替代后端 | 范围决策，**不是缺口** |
| 4 | PostgreSQL 备份交给 PostgreSQL | `backup`/`restore` 仅 SQLite；PG 侧直接报未实现，需自己 `pg_dump` | 缺口 |
| 5 | 检索 p95 ≤ 30 ms 等性能指标 | **从未测量**，仓库无 `bench/` | 声明与证据不匹配 |
| 6 | 审核队列「键盘驱动，5 秒一条」 | Web 无任何键盘快捷键，纯鼠标 | 未实现 |
| 7 | `zhparser` 升级中文路径 | 未实现；2 字中文在 PG 上仍退化为 `LIKE` | 缺口 |
| 8 | `setting` 表存 locale 等服务端设置 | 表存在但**无任何读写**。locale 存 cookie `rocky_mem_locale`，由服务端注入首屏 | 简化，代价是换浏览器不同步 |
| 9 | `idempotency` 表承载写幂等 | 表存在但**无任何读写**。写幂等靠内容 hash → `unchanged` | 死表：要么补上，要么删掉 |

分三类看这张表，别混：

- **第 2、4、6、7 条是缺口** —— 单人模式下也确实没做，属于「把核心做扎实」要补的。
- **第 1 条是取舍** —— 有意为之，代价已写在设计里。
- **第 3、8、9 条是范围与接线问题** —— 不是欠债待还，而是**当前不做**；读 schema 的人最容易在这里误判为「功能存在」。

第 5 条单独说：性能数字**没有任何测量支撑**。要么补 `bench/`，要么把表删掉——留在那里说「可复现」是不诚实的。

**已关闭**：原表第 1、2 条（语料层缺失、`chunk` 值无写入方）已在 v0.1.3 落地，见 [ADR-0008](adr/0008-corpus-layer.md) 与 §16.1。

## 16. 已交付与下一步

路线图不再用 P0–P4 的计划口径，改为按**实际交付**记录。验收命令都在 `Makefile` 里。

### 16.1 已交付（v0.1.3）

| 层 | 内容 | 验收 |
|---|---|---|
| **Tier-0 采纳** | `AGENTS.md`、`.cursor/rules/rocky-mem-mcp.mdc`、`adopt-verify` 审计判定；live 清单已在真实会话勾选通过 | `make verify-tier0`，`docs/adoption-verdict.md` 为 `pass` |
| **Tier-1 运维** | 守护进程（flock + 发现文件 + Bearer `/api/v1`）、`review`、`reindex`/`purge`/`export`、幂等与 `rev` CAS、`eval` recall@10 ≥ 0.8 | `make verify-tier1` |
| **Tier-2 备份与检索** | `backup`/`restore`（`VACUUM INTO`）、≤2 字查询 `LIKE` 兜底、术语表扩展、最小 Web（cookie 会话） | `make verify-tier2` |
| **Tier-3 一致性与 Web** | `store/conformance` 方言中立套件、`/entry` 详情与冲突并排、`/settings`、React SPA（en/zh-CN、明暗） | `make verify-tier3` |
| **反馈闭环** | `feedback` 表与 `mem_feedback` 完整闭环 | 迁移 `0002` + conformance |
| **PostgreSQL（单人切片）** | `postgres://` 打开真实 `pg_trgm` 后端，复用同一 `conformance.Suite`；**单用户语义，不是共享服务** | `ROCKYMEM_TEST_PG_DSN=… go test ./internal/store/conformance` |
| **冲突裁定（P0d）** | `conflict_resolution` / `link` 表、五种裁定（adopt / split / reject_both / defer / reopen）、理由必填、`split` 条件写回并重建索引、检索包标注未裁定冲突、裁定历史可查、修复「归档不删索引」 | 迁移 `0004` + conformance + `go test ./internal/core -run TestResolve` |
| **语料层（P0c）** | `source`/`chunk` 表、投喂默认写语料（`distill` 改为勾选）、`text_hash` 去重、同 `uri` 重投喂自动归档旧版并从索引移除、`## Background` 段带硬上限、`mem_get` 认 `s_`/`c_`、`purge --source`、`reindex`/`doctor` 纳入 chunk | 迁移 `0003` + conformance + `go test ./internal/core` |

六个 MCP 工具、双方言 schema、检索索引生命周期（`DeleteSearchDoc` 在 supersede/archive/purge 上调用）都已落地，与 §7.4、§5.4 的设计一致。语料层的设计与取舍见 [ADR-0008](adr/0008-corpus-layer.md)。

### 16.2 下一步：把单人核心做扎实

排序依据是「对每天用它的人影响多大」，不是「技术上多难」。

| # | 事项 | 为什么它排在这 |
|---|---|---|
| 1 | **`bench/` 或删掉性能表** | 交付里唯一一条「声明无证据」的东西，必须二选一 |
| 2 | **审核的键盘操作** | 设计写「5 秒一条」，实际除了裁定台之外纯鼠标 |
| 3 | **`wu` 与时间线** | 给经验之外加一条按天/按项目的阅读路径 |
| 4 | **向量检索** | `pgvector`（PG）/ 暴力余弦（SQLite）；embedding 走外部服务 |

**团队模式不在这个清单里**，它是独立的未来计划，见 [ADR-0007](adr/0007-solo-only-scope.md)。把它从待办里摘出去，正是为了让上面四项不被稀释。

刻意不做：团队模式与鉴权、MySQL、托管 SaaS、多租户隔离、录屏、绩效报表。

## 17. 风险

| 风险 | 概率 | 影响 | 应对现状 |
|---|---|---|---|
| 库变成垃圾场 | 高 | 高 | 经验默认 `proposed`；`stale` 衰减；概览页显示拒绝率。**已实现** |
| agent 根本不用 | 高 | 致命 | 工具描述内嵌协议；`mem_context` 一次调用即给常驻偏好。**Tier-0 已通过**（`docs/adoption-verdict.md` = pass，含一次真实会话的 live 清单勾选） |
| 假知识污染后续会话 | 高 | 高 | 证据强制；冲突成对返回；`feedback --outcome wrong` 立即转 `contested`。**已实现** |
| 两个后端行为漂移 | 高 | 高 | `store/conformance` 方言中立套件；SQLite 绿，PostgreSQL 已接入同一 `Suite`（`ROCKYMEM_TEST_PG_DSN`）。**单人语义已覆盖** |
| 被误读为支持团队模式 | 中 | 高 | 身份表是「预留未接线」、`scope='team'` 是死值、PostgreSQL 是单人替代后端。三处都已在文档与 ADR-0007 里写明；**这是当前最大的沟通风险，不是技术风险** |
| 投喂时泄露 | 中 | 高 | 两阶段预览 + 脱敏 diff 高亮 + L3 默认禁导出。**已实现** |
| 性能只有声明没有测量 | 中 | 中 | §14 已把数字标为「目标，未测量」；下一步要么建 `bench/`，要么删掉那张表 |
| Go 拖慢交付，项目死在完美设计 | 中 | 致命 | 交付按 Tier 切分，每层都有 `make verify-*` 闸门。**已规避**——四层都已过 |
| 平台差异（FTS5、pg_trgm、路径） | 中 | 中 | 启动探测、文档化降级、三平台 CI |

## 18. 待定

| # | 事项 | 建议 |
|---|---|---|
| 1 | 贡献协议：DCO 还是 CLA | **DCO**。除非计划双许可，那必须现在定 |
| 2 | 中文文档承诺范围 | 只承诺 README + 快速开始；设计文档本来就是中文，不额外增加维护面 |

已决：

- **0.x 只交付单人模式，团队模式推迟** —— [ADR-0007](adr/0007-solo-only-scope.md)。重新审视的条件写在 ADR 里：有真实团队要求共享一个库、或同一人要跨机读写、或单人核心缺口全部关闭。
- **PostgreSQL 实现** —— 已落地（`db/migrations/postgres` + `internal/store/postgres`，复用同一 `conformance.Suite`），但定位是单人替代后端。
- **根目录两份中文原始文档的去向** —— 已移出仓库，不参与版本管理，只作为沿革来源被引用（见文末附录）。

## 附录：沿革

本设计源自两份更早的中文文档（均不在本仓库内）：《工作台账\_设计.md》（个人工作台账 v0.1）与《AI 工作痕迹采集\_技术文档.md》（企业痕迹采集 v1.0）。继承的部分：三层模型（事实/过程/决策）、工作单元抽象、只追加存储、落盘前脱敏、规则优先于 LLM。

改变的部分：本系统的首要矛盾是「**agent 能不能找到它、信它、并且不被它误导**」，而不是「能不能采全」。以及后来加入的两条新约束——多人可用、双数据库——见 [ADR-0005](adr/0005-technology-stack.md) 与 [ADR-0006](adr/0006-two-database-strategy.md)。
