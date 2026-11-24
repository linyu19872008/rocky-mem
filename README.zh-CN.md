<div align="center">

# rocky-mem

**面向编码 agent 的记忆与经验服务。** 默认本地优先，团队场景自托管。

记忆是状态，经验是台账。一次调用，两者都能检索。

[设计文档](design/DESIGN.md) · [数据模型](design/schema.md) · [MCP 工具](docs/mcp-tools.md) · [决策记录](design/adr/) · [English](README.md)

</div>

---

> **本译文的基准是 `README.md`；英文版为唯一真相，本页可能滞后。**
> 当前进度：**Tier-3 + Web SPA 骨架** — Cookie 鉴权的 React UI（中英、明暗主题）、
> `store/conformance`、备份与术语扩展；验收 `make verify-tier3`
>（[docs/tier3.md](docs/tier3.md)、[docs/web.md](docs/web.md)）。已打 tag `v0.1.0-tier3`。
> PostgreSQL / 投喂仍属后续。`design/DESIGN.md` 等为中文；契约与工程文档为英文，
> 划分见 [design/README.md](design/README.md)。

## 为什么做这个

每一次 agent 会话都从零开始。上一次的排查路径——试过哪些命令、为什么最后修好了、什么条件下又会失效——全部消失。留下来的只有一个 commit，记录**改了什么**，不记录**为什么**。

现有工具各自只抓到一块：

- 聊天记录记的是「谁说了什么」，不是「做了什么活」
- agent 的记忆文件只留结论，过程全丢
- `git log` 只覆盖代码

rocky-mem 保留**动作、结果、原因**这个三角，并让它能在 30 毫秒内被 agent 检索到。

## 两种形态，同一个二进制

| | 单人 | 团队 |
|---|---|---|
| 存储 | SQLite（单文件） | PostgreSQL |
| 网络 | 仅 loopback | 可配置 bind |
| 身份 | 一个隐式本地用户 | 用户、团队、API token |
| 备份 | `VACUUM INTO` 快照 | 交给数据库服务 |
| 启动 | `rockymem serve --dsn sqlite://…` | `rockymem serve --dsn postgres://…` |

同一个二进制、同一套命令、同一个 MCP 接口，差别只有 DSN。

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

- **MCP 服务** — 6 个工具。多窗口时先 `make serve`，MCP/CLI 用同一 DSN 走守护进程。
- **两类知识** — 带稳定版本链的记忆；带可信度、冲突与证伪的经验台账。
- **运维 CLI** — `review` / `reindex` / `purge` / `export` / `backup`（`make backup`） / `restore` / `eval`。
- **检索** — FTS5 + 短查询 LIKE + zh→en 术语扩展。
- **Web SPA** — React 检索 / 审核 / 条目 / 设置；en·zh-CN；明暗主题（`make web`）。
- **幂等写入** — remember/record 内容去重；memory `rev` CAS。

**规划中：**

- **更完整 Web** — 冲突裁定、文件投喂脱敏预览、双语界面。
- **PostgreSQL 团队模式** — 同一套 MCP，双方言 conformance。

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

## 参与贡献

欢迎贡献。请先读 [CONTRIBUTING.md](CONTRIBUTING.md)，尤其注意一条：**schema 或查询的改动必须在同一个 PR 里同时落到两个方言目录**。

代码、注释、commit 信息与契约一律英文；设计文档中文。issue 可用中文或英文。

## 许可

[Apache-2.0](LICENSE)，版权归各贡献者。
