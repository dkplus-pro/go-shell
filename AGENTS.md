# AGENTS.md — go-shell 开发规范

本仓库是 pnpm + Turborepo monorepo，包含 TypeScript 应用（`apps/demo`、`apps/web`）、共享配置（`packages/*`）与 Go 后端（`apps/server`、`packages/api-go`）。
总体规划与分阶段任务见 `docs/plans/go-backend-v1.md`，执行任何任务前先读它。本文件是全仓库必须遵守的开发铁律。

## 环境要求

- Go：依赖 GOTOOLCHAIN=auto 自动切换工具链（connect/buf 需要 go ≥1.26，首次自动下载）；本地 go ≥1.21 即可引导。代码生成工具经 `apps/server/go.mod` 的 `tool` 指令固定版本，用 `go tool <name>` 调用；不要用全局安装的 CLI
- Node + pnpm（沿用仓库现有版本约束 `.nvmrc`）
- Docker（本地起 PostgreSQL：`docker compose up -d pg`；可选观测栈：`docker compose --profile obs up -d`）

## 铁律（违反即打回）

1. **proto-first**：接口变更一律从改 `proto/` 开始。流程：改 proto → `buf lint` → `buf generate` → **生成物随同提交** → 再改 handler。禁止先写 Go/TS 再补 proto。
2. **禁止手改生成物**：`packages/api-go/gen`、`packages/api-go/openapi`、`apps/web/src/api`、`apps/server/db/gen`（sqlc 产物）下的文件只能由工具生成。改不动 = 改源头重新生成。
3. **生成幂等**：`make gen` 重跑必须零 diff。CI 会校验，不一致即失败。
4. **schema 变更走迁移**：用 golang-migrate up/down 文件，禁止业务代码里自动迁移、禁止手写运行时 SQL 拼接。查询全部写在 `db/queries/*.sql` 交给 sqlc。
5. **错误规范**：服务层返回 `*connect.Error` + connect 状态码；业务错误码定义在 `internal/errcode` 并配 detail `common.v1.ErrorInfo`。禁止裸 `panic` 穿透 handler（recovery 拦截器只兜底，不作为错误处理手段）。
6. **鉴权分层**：认证（JWT）与授权（RBAC）是两个拦截器。新增公开接口必须显式加白名单并在代码评审中说明理由。
7. **新资源表必须评估租户字段**：v1 单租户，但表设计时显式决定是否预留 `tenant_id`（写迁移注释说明结论），事后 retrofit 代价极高。
8. **测试门槛**：service 层单测 + repo 层 PG 集成测试。新业务代码没有测试不允许合并。

## 命令入口

| 场景           | 命令                                                                                      |
| -------------- | ----------------------------------------------------------------------------------------- |
| Go 服务开发    | `cd apps/server && make dev`                                                              |
| proto 重新生成 | `cd apps/server && make gen`（覆盖 buf + sqlc；web 的 orval 在 `apps/web` 下 `pnpm gen`） |
| 数据库迁移     | `make migrate-up` / `make migrate-down`（apps/server）                                    |
| 全仓校验       | 根目录 `pnpm verify`（含 Go vet/build/test；buf lint 自 Phase 1 加入）                    |
| 单测/集成测试  | `go test ./...`（集成测试自动跳过未启动 PG 的环境，勿删跳过逻辑）                         |

## 代码与工程约定

- **目录**：业务代码只放 `apps/server/internal`；模块内三件套 `service.go` / `connect_handler.go` / 数据访问（经 db 包）。公共能力（拦截器、存储、认证）单独成包，业务模块间禁止互相 import。
- **RPC 框架**：ConnectRPC。handler 用 `connect.NewHandler` 挂 `net/http`，一个端口同时服务 gRPC / gRPC-Web / HTTP-JSON。不引入 grpc-gateway，HTTP 路径保持 RPC 风格（`POST /包名.服务/方法`）。
- **web 调用链**：`apps/web` 用 orval 从 `packages/api-go/openapi` 生成 axios 客户端，axios 基座在 `src/lib/http.ts`（token 注入、401 刷新重放、统一错误）。
- **命名**：proto 包名 `api.<域>.v1`；Go 模块路径 `github.com/dkplus-pro/go-shell/...`；迁移文件名 `NNNN_description.up.sql/down.sql` 顺序递增。
- **日志**：只许 `slog` 结构化字段（JSON），禁止 fmt.Println 调试残留。
- **性能默认**：每个 handler 显式 context 超时；慢查询/大响应需要流式处理（文件下载必须流式）。
- **提交**：遵循仓库 commitlint（conventional commits）；生成物改动独立提交，前缀 `chore(gen):`。

## 分阶段执行规则

- 按 `docs/plans/go-backend-v1.md` 的 Phase 0–7 顺序执行，每阶段一个分支/PR，验收标准全绿才算完成，禁止跨阶段提前实现。
- 对计划的任何偏离（换库、改结构、跳过验收项）必须先更新计划文档的决策表并注明理由，再写代码。
