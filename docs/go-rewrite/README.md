# Vanna Go 重写

本目录包含将 [Vanna](https://github.com/vanna-ai/vanna)（Python Text-to-SQL RAG 框架）用 Go 重写的规划与规格说明。

## 文档索引

| 文档 | 说明 |
|------|------|
| [P0-PLAN.md](./P0-PLAN.md) | P0 阶段总体计划：架构、接口、API、实施顺序 |
| [chart-spec.schema.json](./chart-spec.schema.json) | 图表配置 ChartSpec 的 JSON Schema |

## 背景

原版 Vanna 是 Python RAG 框架，通过向量检索 DDL / 文档 / 历史 SQL，再调用 LLM 生成 SQL。Go 重写的目标：

- **性能与部署**：单二进制、无 Python 运行时、依赖更少
- **P0 功能等价**：复刻核心 Text-to-SQL 能力（训练、生成、执行、图表、辅助功能）
- **暂不优化**：完整多轮对话记忆等增强放在 P1

## 当前状态

- [x] 技术选型与 P0 范围已确认（见 P0-PLAN.md）
- [x] Go 模块骨架与核心 API（`go/`）
- [ ] 集成测试与生产加固
