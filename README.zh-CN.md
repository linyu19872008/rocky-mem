<div align="center">

# rocky-mem

**面向编码 agent 的记忆与经验服务。** 默认本地优先，团队场景自托管。

记忆是状态，经验是台账。一次调用，两者都能检索。

[设计文档](design/DESIGN.md) · [数据模型](design/schema.md) · [MCP 工具](docs/mcp-tools.md) · [决策记录](design/adr/) · [English](README.md)

</div>

---

> **本译文的基准是 `README.md`；英文版为唯一真相，本页可能滞后。**
> 项目已进入 **P0a 实现**：SQLite + MCP/CLI 垂直切片可运行（见 [docs/quickstart-p0a.md](docs/quickstart-p0a.md)）。PostgreSQL 与 Web 尚未实现。
> `design/DESIGN.md` 等设计文档本身就是中文；契约与工程文档为英文，划分见 [design/README.md](design/README.md)。

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

## 特性

- **MCP 服务** — 6 个工具，Cursor、DeepSeek Harness 及任何 MCP 客户端可用；只读工具可以无条件调用。
- **两类知识** — 带稳定版本链的记忆；带可信度状态、冲突检测与证伪机制的经验台账。
- **Web 界面** — 处理不该交给 agent 决定的事：审核队列、冲突裁定、文件投喂（**写入前先看脱敏预览**）。
- **默认双语** — 英文与简体中文，按浏览器语言或你的设置解析；任意语言写的内容都能通过术语表跨语言检索。
- **两个存储后端** — 单人的 SQLite、团队的 PostgreSQL，都不是二等公民；同一套一致性测试在两个后端上跑同样的行为。

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
