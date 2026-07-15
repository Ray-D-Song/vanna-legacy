# Vanna Go

Go 语言实现的 Vanna Text-to-SQL 服务。规划见 [docs/go-rewrite/P0-PLAN.md](../docs/go-rewrite/P0-PLAN.md)。

## 快速开始

```bash
cd go
cp config.example.yaml config.yaml
# 编辑 config.yaml，设置 OPENAI_API_KEY 与 database.dsn

go run ./cmd/vanna serve --config config.yaml
```

浏览器打开 `http://localhost:8080`。

## 依赖

- OpenAI 兼容 Chat / Embeddings API
- chromem-go 嵌入式向量库（无需独立 PG）
- MySQL 或 DuckDB 作为业务查询库

## API

基础路径 `/api/v1`：

- `POST /ask` — 一站式问答
- `POST /generate_sql` / `POST /run_sql` — 分步流程
- `POST /train` — 训练 DDL / 文档 / 问答 SQL
- `GET /training_data` / `DELETE /training_data/{id}`
- `GET /healthz`

Session 通过响应中的 `session_id` 或 Header `X-Session-ID` 传递。

## 构建

```bash
cd go
go build -o vanna ./cmd/vanna
```
