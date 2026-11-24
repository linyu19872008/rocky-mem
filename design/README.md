# 设计文档

这个目录只放**设计层**文档：为什么这样做、数据长什么样、有哪些取舍。对外契约与工程规范不在这里，见 [`../docs/`](../docs/) 与仓库根目录。

## 目录约定

| 目录 | 放什么 | 语言 | 读者 |
|---|---|---|---|
| `design/`（本目录） | 设计文档、数据模型、架构决策记录 | **中文** | 维护者 |
| `docs/` | 对外契约（MCP 工具、错误码）与规范（i18n） | 英文 | 外部贡献者、第三方客户端 |
| 仓库根 | README、CONTRIBUTING、SECURITY、CHANGELOG、LICENSE | 英文（README 另有中文版） | 开源协作的公共语言 |

划分依据不是重要性，是**读者是谁**。设计里的权衡用母语表达更精确；对外契约必须与代码标识符一致，而标识符是英文。详见 [ADR-0004](adr/0004-i18n-ui-scope-vs-content-language.md) §7。

**代码标识符在任何文档、任何语言里都是英文**：`mem_search`、`trust='contested'`、`ERR_WRITE_CONFLICT`。翻译标识符是缺陷，不是本地化。

## 文件清单

| 文件 | 内容 | 状态 |
|---|---|---|
| [DESIGN.md](DESIGN.md) | 目标、原则、架构、机制、路线图、风险、待定项 | 定稿 |
| [schema.md](schema.md) | 双方言 DDL、索引、检索写入契约、迁移、运维、禁止操作 | 定稿 |
| [adr/0001](adr/0001-record-architecture-decisions.md) | 为什么记录架构决策 | 已采纳 |
| [adr/0002](adr/0002-split-memory-and-experience-tables.md) | 记忆与经验拆成两张表 | 已采纳 |
| [adr/0003](adr/0003-sqlite-first-with-portable-schema.md) | SQLite 优先、无并发写者（历史） | **已被 0005 取代** |
| [adr/0004](adr/0004-i18n-ui-scope-vs-content-language.md) | 界面本地化与内容语言分离 | 已采纳 |
| [adr/0005](adr/0005-technology-stack.md) | 采用 Go，单二进制支撑双形态 | 已采纳 |
| [adr/0006](adr/0006-two-database-strategy.md) | 只支持 SQLite 与 PostgreSQL | 已采纳 |

## 建议阅读顺序

**第一次接触项目：**

1. [`../README.md`](../README.md) — 三分钟知道它是什么
2. [DESIGN.md](DESIGN.md) §1–§5 — 问题、原则、核心概念、架构
3. [schema.md](schema.md) §1、§2.4、§2.5 — 两张核心表为什么长这样
4. [ADR-0002](adr/0002-split-memory-and-experience-tables.md) — 拆表的完整论证

**要动手实现：**

1. [schema.md](schema.md) 全文，尤其 §5（检索文档的写入契约）、§5.4（生命周期）与 §5.5（supersede 的顺序）
2. [`../docs/mcp-tools.md`](../docs/mcp-tools.md) — 对外契约
3. [`../docs/error-codes.md`](../docs/error-codes.md) — 错误码登记表
4. [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — 工具链与提交规范

**要评估技术选型：**

1. [ADR-0005](adr/0005-technology-stack.md) — 为什么是 Go
2. [ADR-0006](adr/0006-two-database-strategy.md) — 为什么只支持两库、为什么用 `pg_trgm`

## 维护规则

1. **ADR 一旦采纳不再修改。** 要改就写新 ADR 引用旧的，旧文件只加一行 `状态：已被 ADR-00xx 取代`。
2. **`DESIGN.md` 与 `schema.md` 描述当前状态**，随实现变化就地更新。它们与 ADR 的分工是：ADR 记「当时为什么这么选」，设计文档记「现在是什么」。
3. **新增设计文档前先问：它是不是某条 ADR 的展开？** 如果是，写在 ADR 里，不要新增文件。
4. **图表用 ASCII**，不用图片。图片会过期且无法 diff。
5. **设计文档里的代码块必须是可执行的。** `schema.md` 的 SQLite DDL 由 CI 抽取执行；改 DDL 必须同时跑通。
6. **跨目录链接用相对路径**，移动文件时一并更新。

## 尚未编写

- 长篇复盘式设计（例如「检索质量如何评测」）——等 P1 有真实数据再说。
- 向量检索设计（P3）。

## 历史材料

仓库根目录还留着两份更早的中文文档：`工作台账_设计.md` 与 `AI工作痕迹采集_技术文档.md`。它们是本设计的输入材料，不是当前设计的一部分，按 `DESIGN.md` §18 待定项决定去留。
