<div align="center">

<img src="assets/brand/icon-512.png" width="132" alt="Rocky，rocky-mem 吉祥物">

# rocky-mem

**面向编码 agent 的记忆与经验服务。** 本地优先，只做单人模式。

记忆是状态，经验是台账。一次调用，两者都能检索。

[![ci](https://github.com/linyu19872008/rocky-mem/actions/workflows/ci.yml/badge.svg)](https://github.com/linyu19872008/rocky-mem/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/linyu19872008/rocky-mem?sort=semver)](https://github.com/linyu19872008/rocky-mem/releases)
[![license](https://img.shields.io/github/license/linyu19872008/rocky-mem)](LICENSE)
[![go](https://img.shields.io/github/go-mod/go-version/linyu19872008/rocky-mem)](go.mod)
[![skill](https://img.shields.io/npm/v/rocky-mem-skill?label=skill)](https://www.npmjs.com/package/rocky-mem-skill)


[设计文档](design/DESIGN.md) · [数据模型](design/schema.md) · [MCP 工具](docs/mcp-tools.md) · [决策记录](design/adr/) · [English](README.md)

</div>

---

> **本译文的基准是 `README.md`；英文版为唯一真相，本页可能滞后。**
> 当前进度：**Solo 可用 — v0.2.2** — Cookie 鉴权 React UI（知识库 / 记住 / 记录经验 / 投喂 / 审核）、
> 文件投喂（默认存为语料）、冲突裁定、备份与术语扩展；验收 `make verify-tier3`
>（[docs/web.md](docs/web.md)、[docs/ingest.md](docs/ingest.md)）。
> **PostgreSQL Open + conformance 已落地；团队鉴权仍属后续。** `design/DESIGN.md` 等为中文；契约与工程文档为英文，
> 划分见 [design/README.md](design/README.md)。

## 为什么做这个

每一次 agent 会话都从零开始。上一次的排查路径——试过哪些命令、为什么最后修好了、什么条件下又会失效——全部消失。留下来的只有一个 commit，记录**改了什么**，不记录**为什么**。

现有工具各自只抓到一块：

- 聊天记录记的是「谁说了什么」，不是「做了什么活」
- agent 的记忆文件只留结论，过程全丢
- `git log` 只覆盖代码

rocky-mem 保留**动作、结果、原因**这个三角，并让它能在 30 毫秒内被 agent 检索到。

## 当前只做单人模式

0.x 支持**一台机器、一个人**。这不是测试版的临时限制，而是产品范围（`design/adr/0007-solo-only-scope.md`），所有已交付的东西都是按这个前提做的：一个守护进程、一个 SQLite 文件、一枚 token。

| | 单人（`sqlite://`）—— **支持** | 团队 —— **规划中，未实现** |
|---|---|---|
| 数据 | 本机一个文件 | — |
| 身份 | 一个隐式本地用户 | 用户 / 团队 / token 都还没有 |
| 备份 | `VACUUM INTO` 快照 | — |
| 启动 | `rockymem serve` | — |

`--dsn postgres://…` 现在能用，但它给你的是**用 PostgreSQL 存的单用户库**，不是协作：没有鉴权、没有成员关系、没有并发写验证、没有备份。见 [docs/postgres.md](docs/postgres.md)。

## 用起来是什么样

agent 在开始工作前调用一次：

```json
{ "query": "滚动更新后配置丢失", "cwd": "/Users/x/repo/service-a" }
```

拿到有预算、分好段、带出处的上下文包：

```markdown
## Applies
### [pitfall] 滚动更新期间 ConfigMap 热更新会丢失
- Applies when: K8s + 文件挂载的 ConfigMap + 滚动窗口短于 60s
- Do: 给 Pod 模板加 checksum annotation 强制重建
- Do not apply when: 配置通过 ConfigMap API 直读，而非文件挂载
- Evidence: commit abc1234 (L2 · active · confidence 0.85 · e_01J9K…)

## Do not use
- 只跑 `kubectl rollout restart`：配置照样丢（试过，不行 · e_01JA…）
```

**「Do not use」这一段才是重点。** agent 很少因为不知道解法而失败，更多是因为在不适用的条件下用了它。

## 设计原则

1. **两种知识，两张表。** 偏好与事实是**状态**（只有一个当前值），经验是**台账**（累积陈述、可被质疑）。一张表无法同时满足两套写语义。→ [ADR-0002](design/adr/0002-split-memory-and-experience-tables.md)
2. **一个写入漏斗，多个入口。** agent 写入、人手写、文件投喂走同一条脱敏、去重、闸门管道。
3. **检索返回上下文包，不是结果列表。** 有预算、有分段、带出处。
4. **没有证据就不算发生。** 经验条目必须指向不可变的原始记录。
5. **核心只追加，索引可重建。** `reindex` 能重建全部派生数据。
6. **规则优先，LLM 靠后。** 热路径里没有任何 LLM。
7. **方言差异显式。** 两个存储后端共用一个接口，各自的 SQL 分开维护、分开评审——不做假装它们一样的魔法查询。
8. **失败也是资产。** 没成功的 `howto` 和成功的一样有价值。
9. **降级必须声明。** FTS5 或 `pg_trgm` 不可用时，`doctor` 和界面都要说，绝不静默。
10. **可随时删除。** 单人模式就是一个目录，`rm -rf` 掉什么都不剩。

## 特性（已交付 vs 规划）

**已交付（单人 SQLite）：**

- **MCP 服务** — 6 个工具，**stdio 与 Streamable HTTP 两种传输**（`POST /mcp`，无状态，Bearer 鉴权）。
  多窗口时先 `make serve`，MCP/CLI 用同一 DSN 走守护进程。
- **两类知识** — 带稳定版本链的记忆；带可信度、冲突与证伪的经验台账。
- **运维 CLI** — `review` / `reindex` / `purge` / `export` / `backup`（`make backup`） / `restore` / `eval`。
- **检索** — FTS5 + 短查询 LIKE + zh→en 术语扩展。
- **Web SPA** — 检索、**知识库**、**记住** / **记录经验**、审核、条目、**投喂**、概览、设置（一键备份）；en·zh-CN；明暗主题。
- **投喂** — 两阶段预览/确认（plain + docx/xlsx/PDF 文本层；脱敏差异）。默认存成**语料**（`source` + `chunk`），勾选「同时生成待审经验」才走 `distill`。
- **冲突裁定** — 两条对同一症状给了不同解法时，由人裁定：采纳其一 / 条件互补 / 都不成立 / 暂缓 / 复开。裁定只追加、理由必填；未裁定的冲突在上下文包里显式标注。
- **语料检索** — 片段进 `## Background` 段，有硬上限（默认 2 条 / 每源 1 条），绝不挤掉经验；`mem_get` 读片段（`c_…`）或列文档目录（`s_…`）；`purge --source` 删整篇。
- **幂等写入** — remember/record 内容去重；memory `rev` CAS（MCP 已暴露 `rev`）。

**已交付（PostgreSQL 存储切片）：**

- **Open + migrate + Store** — `--dsn postgres://…` / `postgresql://…`，`pg_trgm`；conformance 用 `ROCKYMEM_TEST_PG_DSN`。**定位是单用户替代后端，不是共享服务。**

**下一步（在范围内）：**

- **`wu` 时间线 / 向量** — 见 `design/DESIGN.md` §16.2。
- **`bench/`** — 性能数字目前仍是目标，不是测量值。

**0.x 不做：**

- **团队模式** — 共享鉴权、membership、多写者并发、PG 备份。属于未来规划，不在当前实现范围，见 `design/adr/0007-solo-only-scope.md`。
- **MySQL**、托管 SaaS、多租户隔离。

## 安装

```bash
go install github.com/linyu19872008/rocky-mem/cmd/rockymem@latest
```

或者直接从 [Releases](https://github.com/linyu19872008/rocky-mem/releases) 下载二进制：
darwin / linux / windows × amd64 / arm64，附 `checksums.txt`。二进制是静态的
（CGO 关闭）且内嵌 Web UI，不需要额外依赖。

```bash
rockymem init  --dsn sqlite://$HOME/.local/share/rocky-mem/rocky-mem.db
rockymem serve --dsn sqlite://$HOME/.local/share/rocky-mem/rocky-mem.db
# stdio MCP:  rockymem mcp
# HTTP  MCP:  http://127.0.0.1:7432/mcp   （token 在数据库旁边）
```

只有 `go install` 需要 Go 1.26+；下载的发布二进制不需要。

## 本仓库快速开始

```bash
make build
./scripts/seed-demo.sh
make serve                 # 独占 .local/rocky-mem.db
# .local/rocky-mem.token 跨重启复用；需要作废时：
#   ./bin/rockymem --dsn sqlite://$PWD/.local/rocky-mem.db serve --rotate-token
make verify-tier1
```

## 明确不做

不做录屏与键鼠记录 · 不做托管 SaaS · 不做互不信任组织之间的多租户隔离 · 不做生产力报表 · 不翻译你写的内容 · 不做通用笔记应用 · 不做图谱可视化 · **不支持 MySQL**（理由见 [ADR-0006](design/adr/0006-two-database-strategy.md)）。

「不搞生产力报表」这条最重要：被用于监控的系统一定会被绕过，而有偏的样本比没有数据更危险。

## 给 agent 的技能包

本仓库同时发布 **`rocky-mem-skill`**（npm）——agent 侧的「行为手册」：何时查台账、
怎么读上下文包、什么写法会被拒绝。技能包与服务器**同一个 tag 发布**，CI 断言
`skills/rocky-mem/SKILL.md` 的 `version` 与 `npm/skill/package.json` 一致。

| 宿主 | 安装 |
|---|---|
| DeepSeek Harness | 装插件 [`rocky-mem-dsh`](https://github.com/linyu19872008/rocky-mem-dsh)：自动注册技能并在会话启动时注入上下文 |
| Cursor | `npx rocky-mem-skill install --host cursor` |
| Claude Code | `npx rocky-mem-skill install --host claude-code` |
| Codex / 任意 AGENTS.md 宿主 | `npx rocky-mem-skill install --host codex` |

规范源是 `skills/rocky-mem/`；`npm/skill/` 在发布时由 `prepack` 从它生成，不手工编辑。

## 文档

| | |
|---|---|
| [design/DESIGN.md](design/DESIGN.md) | 设计：目标、原则、架构、机制、路线（中文） |
| [design/schema.md](design/schema.md) | 双方言 DDL、索引、检索写入契约、迁移（中文） |
| [docs/mcp-tools.md](docs/mcp-tools.md) | agent 侧契约（英文） |
| [docs/error-codes.md](docs/error-codes.md) | 稳定错误码表（英文） |
| [docs/i18n.md](docs/i18n.md) | 本地化与内容多语言（英文） |
| [design/adr/](design/adr/) | 不可逆决策的记录（中文） |
| [design/README.md](design/README.md) | 设计文档索引、阅读顺序、维护规则 |
| [docs/README.md](docs/README.md) | 契约文档索引；哪类文档该放哪里 |
| [skills/rocky-mem/](skills/rocky-mem/) | agent 技能规范源（双语行为手册 + 各宿主适配） |

## 参与贡献

欢迎贡献。请先读 [CONTRIBUTING.md](CONTRIBUTING.md)，尤其注意一条：**schema 或查询的改动必须在同一个 PR 里同时落到两个方言目录**。

代码、注释、commit 信息与契约一律英文；设计文档中文。issue 可用中文或英文。

## 许可

[Apache-2.0](LICENSE)，版权归各贡献者。
