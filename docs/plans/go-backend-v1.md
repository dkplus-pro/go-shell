# go-shell 通用 Go 后端 v1 落地计划

> 状态：**已定稿**（2026-10-25 访谈锁定全部架构决策）
> 执行方式：分阶段，每阶段一个分支/PR，由后续跟进模型按序领取执行
> 配套规范：根目录 `AGENTS.md`（跟进模型必读）

## 1. 背景与目标

在本 pnpm + Turborepo TS monorepo 中新增一个**无行业属性、可随时接入任意业务**的通用 Go 后端服务（`apps/server`），要求高性能、高并发、高可用；以 proto 为单一契约源，同时服务内部 gRPC 调用方与 web HTTP/JSON 调用方；附带一个最基础的接口验证 web 端（`apps/web`）。

**明确的非目标（v1 不做，但留好接口）：**

- 多租户隔离（用户/权限模型不堵死，见 §9 演进）
- S3/MinIO 对象存储（Storage 接口可插拔，v1 只做本地实现）
- OpenTelemetry 分布式追踪（指标与日志已齐，OTel 后续加）
- k8s 部署形态（提供 Dockerfile 即可）

## 2. 已锁定决策（访谈结论，不再重议）

| #   | 决策点        | 结论                                                                                                  | 理由摘要                                                                              |
| --- | ------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| D1  | 业务范围      | 认证+用户、RBAC、字典/配置 KV、文件上传 **全做**                                                      | 通用业务底座的完整验证面                                                              |
| D2  | RPC 框架      | **ConnectRPC（connect-go）**                                                                          | 一份 proto 同时出 gRPC / gRPC-Web / HTTP/JSON 三协议，同端口同 handler，无 gateway 层 |
| D3  | HTTP 路径形态 | 接受 **RPC 风格路径**（`POST /包名.服务/方法`，无副作用方法可 GET）                                   | 已确认不为 REST 资源路径引入 grpc-gateway                                             |
| D4  | 持久层        | **PostgreSQL + sqlc**                                                                                 | 生产级；SQL 生成类型安全代码，无 ORM 运行时开销                                       |
| D5  | proto 工具链  | **buf**；**生成物（\*.pb.go / OpenAPI / sqlc / TS）全部提交入库**                                     | 跟进模型与 CI 不装工具链即可编译；proto 改动必须同步重新生成并一并提交                |
| D6  | 交付节奏      | **分 8 个阶段**（见 §6），每阶段独立可编译可测                                                        | 降低单次交付体量，失败可回滚可接手                                                    |
| D7  | Go 布局       | 根 `go.work` + 双模块：`apps/server`（主服务）+ `packages/api-go`（proto 生成物）                     | 未来第二个 Go 服务直接复用接口模块                                                    |
| D8  | web 技术栈    | **Modern.js**（与 demo 一致）                                                                         | 复用 repo 现有构建链与共享配置                                                        |
| D9  | web 调用链    | **OpenAPI → orval 生成 TS 类型 + axios 请求函数**，axios 基座统一封装（拦截器带 token、统一错误处理） | 用户指定；OpenAPI 同时服务第三方对接                                                  |
| D10 | 可观测基线    | **slog(JSON) + /debug/pprof + Prometheus /metrics**，connect 拦截器自动埋点                           | 排查线上问题的最小完整集                                                              |
| D11 | 文件存储      | Storage **可插拔接口 + 本地磁盘实现**                                                                 | compose 不起 MinIO，一期最轻                                                          |
| D12 | 多租户        | 不做，预留演进位（见 §9）                                                                             | 事后 retrofit 代价已通过设计缓解                                                      |

## 3. 总体架构

```
                      ┌──────────────────────────────────────────┐
                      │              apps/server (Go)            │
   内部服务 ── gRPC ─▶ │  net/http                                │
   (二进制,高性能)      │   ├─ connect handlers (auth/user/rbac/   │
                      │   │   dict/file)                          │
   web (Modern.js) ──▶ │   ├─ 三协议自动协商: gRPC|gRPC-Web|JSON   │
   axios+orval  HTTP   │   ├─ /healthz /readyz                    │
                      │   ├─ /metrics (Prometheus)                │
   第三方 ── OpenAPI ─▶ │   └─ /debug/pprof/*                      │
                      │  拦截器链: recovery→reqID→日志→指标→       │
                      │           超时→限流→鉴权(JWT)→RBAC        │
                      └───────────────┬──────────────────────────┘
                                      │ sqlc (编译期生成)
                                      ▼
                              PostgreSQL 17
                                      │
                      Storage 接口 ── 本地磁盘实现 (未来 S3)
```

**契约流（单一事实源 = proto）：**

```
proto/*.proto ──buf generate──▶ packages/api-go/gen/...   (Go: connect+pb, 提交)
             └───────────────▶ packages/api-go/openapi/   (OpenAPI v3, 提交)
                                          │
             apps/web (orval) ────────────┘  生成 TS 类型 + axios 函数 (提交)
```

## 4. 目录结构（目标态）

```
go-shell/
├── AGENTS.md                        # 开发规范（本计划配套，已创建）
├── go.work                          # go 1.24; apps/server + packages/api-go
├── docker-compose.yml               # postgres:17 (+ profile: prometheus/grafana)
├── proto/                           # ★ 所有 proto 契约（单一事实源）
│   ├── buf.yaml                     # module 定义 + lint 配置(STANDARD)
│   ├── buf.gen.yaml                 # 插件: connect-go / protobuf-go / connect-openapi
│   └── api/
│       ├── common/v1/common.proto   # 分页、错误详情等公共类型
│       ├── auth/v1/auth.proto       # 注册/登录/刷新/登出
│       ├── user/v1/user.proto       # 用户 CRUD
│       ├── rbac/v1/rbac.proto       # 角色/权限点/绑定/校验
│       ├── dict/v1/dict.proto       # 字典/配置 KV
│       └── file/v1/file.proto       # 上传/下载/删除/列表
├── packages/
│   └── api-go/                      # 独立 Go 模块（生成物，禁止手改）
│       ├── go.mod                   # github.com/dkplus-pro/go-shell/packages/api-go
│       ├── gen/…                    # buf 生成（提交）
│       └── openapi/…                # OpenAPI v3 生成（提交）
└── apps/
    ├── server/                      # Go 主服务模块
    │   ├── go.mod                   # github.com/dkplus-pro/go-shell/apps/server
    │   ├── Makefile                 # dev / test / lint / gen / migrate 等入口
    │   ├── package.json             # 薄壳：scripts 桥接 make，纳入 turbo 任务图
    │   ├── Dockerfile               # 多阶段构建，distroless 运行
    │   ├── cmd/server/main.go
    │   ├── internal/
    │   │   ├── config/              # env 配置 (caarlos0/env)
    │   │   ├── server/              # http 组装、拦截器链、优雅退出
    │   │   ├── interceptor/         # recovery/reqID/logging/metrics/timeout/
    │   │   │                        # ratelimit/auth/rbac (connect interceptors)
    │   │   ├── authn/               # JWT 签发校验(golang-jwt/v5)、argon2id 哈希
    │   │   ├── module/              # 业务模块
    │   │   │   ├── auth/  user/  rbac/  dict/  file/
    │   │   │   └─ (每模块: service.go + repo(用db包) + connect_handler.go)
    │   │   └── storage/             # Storage 接口 + local 实现
    │   └── db/
    │       ├── migrations/          # golang-migrate up/down
    │       ├── queries/*.sql        # sqlc 输入
    │       └── sqlc.yaml
    └── web/                         # Modern.js 接口验证端
        ├── orval.config.ts          # 输入: packages/api-go/openapi
        ├── src/api/                 # orval 生成（提交）
        ├── src/lib/http.ts          # axios 基座：注入 token、统一错误、401 刷新
        └── src/routes/…             # 每业务模块一个验证页
```

## 5. 关键技术约束（写代码前必读）

1. **proto-first**：任何接口变更从改 `proto/` 开始 → `buf lint` → `buf generate` → 生成物随同提交 → 再改 handler 实现。禁止手改 `packages/api-go/gen`、`openapi`、`src/api` 下任何生成文件。
2. **服务组装**：connect-go 的 `NewHandler` 挂 `net/http`，一个端口同时服务 gRPC / gRPC-Web / Connect-JSON，**不引入** grpc-go server 与 grpc-gateway。
3. **错误规范**：统一用 `*connect.Error` + connect 状态码；业务错误码放 proto detail（`common.v1.ErrorInfo`），错误码表集中在 `internal/errcode` 包并同步文档。
4. **数据层**：一切 schema 变更走迁移文件（禁止自动迁移）；一切查询写在 `db/queries/*.sql` 由 sqlc 生成，禁止运行时拼 SQL。
5. **鉴权**：JWT 双 token（短期 access + 可撤销 refresh，DB 存 refresh 哈希支持轮换与登出）；密码 argon2id；authn（认证）与 authz（RBAC 鉴权）是两个拦截器。
   5a. **工具链实测结论（2026-10-25 定稿）**：buf.build 远程插件域不可达，生成链全部走本地 go tool 插件——`go get -tool` 固定 `buf`、`protoc-gen-go`、`protoc-gen-connect-go`、`protoc-gen-connect-openapi`（sudorandom/protoc-gen-connect-openapi，输出 OpenAPI 3.1）四个工具；`buf.gen.yaml` 用 `local: ["go","tool","<plugin>"]` 列表形式调用（已端到端验证）。connect v1.21+ 与 buf v1.75 要求 go ≥1.26，依赖 GOTOOLCHAIN=auto 自动下载新工具链（本机已验证可用）。
6. **性能默认值**：所有 handler 显式 `context` 超时；连接池参数可配；pprof 常开（生产可按配置关闭）；限流拦截器默认全局限速 + 可配 per-IP。
7. **测试**：service 层单测（mock repo）+ repo 层跑真实 PG 的集成测试（`docker compose up -d pg` 前置）；每个阶段的验收标准里测试是硬门槛。

## 6. 分阶段任务

> 规则：每阶段一个分支 `feat/phase-N-xxx`；生成物改动与触发它的源改动同一提交；验收标准全绿才算完成；禁止跨阶段提前实现。

### Phase 0 — 仓库手术 + Go 骨架

**目标**：pages 清理完毕、Go 服务空转起来、工具链全通。
任务：

- 删除 `scripts/deploy-github-pages.sh`；根 `package.json` 删 `build:pages`；清理 `README.md` 与 `apps/demo/package.json` 中 pages 相关引用
- 建 `go.work`（go 1.24）+ `apps/server/go.mod`；`tools.go`/go tool 固定 buf、migrate、sqlc 版本
- `docker-compose.yml`（postgres:17，volume 持久化；prometheus/grafana 放 compose profile）
- `cmd/server/main.go`：config 加载 → slog(JSON) 初始化 → HTTP 组装（仅 `/healthz` `/readyz` `/metrics` `/debug/pprof/*`）→ 优雅退出（SIGTERM 排空连接）
- 拦截器链骨架按 §5 顺序挂好（本阶段 recovery/reqID/logging/metrics/timeout 生效，auth/rbac 留空位）
- `Makefile`（dev/test/lint/gen/migrate-up/down）+ 薄壳 `package.json` 桥接 turbo；`scripts/verify.sh` 与 `scripts/ci.sh` 增加 Go 段（go vet/build/test）
- 校对根 `AGENTS.md`（已创建），按实际情况补命令入口
  验收：`make dev` 起服后 `curl /healthz` 200、`/metrics` 有指标、SIGTERM 优雅退出日志可见；`pnpm verify` 全绿（含 Go）；仓库内 grep 不到 github-pages 引用。

### Phase 1 — proto 契约 + 全量生成链

**目标**：六个 proto 定稿（字段级评审），生成链一次跑通。
任务：

- `proto/buf.yaml`（lint STANDARD、breaking 比对 main）、`buf.gen.yaml`
- `common/v1`：PageRequest/PageResponse/ErrorInfo
- auth/user/rbac/dict/file 五服务 proto（含方法注释 = 未来 OpenAPI 文档主体；无副作用查询标 `idempotency_level: NO_SIDE_EFFECTS`）
- `buf generate` → `packages/api-go`（go 模块 + connect 代码 + OpenAPI v3 yaml）全部提交
- server 侧为每个服务挂**未实现**的 handler（返回 `connect.CodeUnimplemented`），路由生效可被 curl/gRPC 调用到
- CI 增加 `buf lint` + `buf breaking --against '.git#branch=main'`
  验收：`buf lint` 0 警告；生成物入库且 `make gen` 幂等（重跑无 diff）；curl 任一方法得到 Unimplemented 的 JSON 错误体；OpenAPI yaml 可被 swagger 编辑器打开且含全部方法。

### Phase 2 — 认证 + 用户

**目标**：注册/登录/刷新/登出全通，用户 CRUD 完整。
任务：

- 迁移：`users` 表（id/username/email/password_hash/status/预留 tenant_id 位 NULL/timestamps）、`refresh_tokens` 表（user_id/token_hash/expires_at/revoked_at）
- sqlc 查询集；`internal/authn`（JWT v5 签验、argon2id）；`internal/module/auth`（register/login/refresh/logout，refresh 轮换+撤销）；`internal/module/user`（CRUD + 分页）
- authn 拦截器实装：白名单（login/register/health…）外的调用必须带 `Authorization: Bearer`
- 单测：auth service 全路径（成功/密码错/用户名重复/过期 token/刷新轮换后旧 token 失效）；repo 集成测试连真实 PG
  验收：curl 完成 注册→登录→带 token 查自己→刷新→旧 refresh 再用被拒→登出 全链路；`go test ./...` 绿；错误场景返回规范错误码。

### Phase 3 — RBAC

任务：roles/permissions/role_permissions/user_roles 迁移；rbac 模块（角色 CRUD、权限点 CRUD、绑定/解绑、`Check` 内部方法）；authz 拦截器按 `服务.方法→所需权限点` 映射表鉴权；预置超管角色与种子迁移；单测覆盖越权/放行边界。
验收：无权限调用返回 `PermissionDenied`；授权后放行；超管全通；权限变更即时生效（无缓存或缓存失效正确）。

### Phase 4 — 字典/配置 KV

任务：dict_entries 迁移（scope/group/key/value(jsonb)/备注/timestamps，唯一约束 scope+group+key）；dict 模块 CRUD + 按 group 批量拉取；读接口标记 NO_SIDE_EFFECTS 可 GET；单测含唯一约束冲突与并发 upsert。
验收：CRUD + 分组批量读取经 curl 验证；重复 key 返回规范冲突错误码。

### Phase 5 — 文件上传

任务：`internal/storage` 定义 `Storage` 接口（Put/Get/Delete/Stat/List）+ `LocalStorage`（根目录可配、按日期分桶）；files 表迁移（元数据、sha256、size、uploader）；file 模块（上传 multipart、下载流式、删除、列表）；下载走 `NO_SIDE_EFFECTS`；上传大小/类型白名单可配；单测（临时目录）+ 集成测试。
验收：curl 上传→列表→下载 sha256 一致→删除后 404；超限文件被拒并返回明确错误码。

### Phase 6 — web 验证端

**目标**：Modern.js 验证面板，全部接口可视化打通。
任务：

- `apps/web` Modern.js 应用（复用共享 eslint/tsconfig，纳入 turbo）
- `orval.config.ts` 指向 `packages/api-go/openapi` → 生成 TS 类型 + axios 函数到 `src/api`（提交）
- `src/lib/http.ts` axios 基座：请求注入 Bearer、401 自动刷新重放一次、统一错误 toast/文案、baseURL 走 env
- 页面：登录页（token 展示/刷新）、用户管理、角色权限绑定、字典 KV 编辑、文件上传下载；首页放健康检查与接口连通性面板
  验收：`pnpm dev` 起 web 后纯界面完成 Phase 2–5 的全部验收场景；`pnpm lint`/`typecheck` 绿；生成物与手写代码分离清晰。

### Phase 7 — 收尾

任务：根 README 重写（架构图、快速开始、协议说明三节）；`docs/api` 放 OpenAPI 使用说明；scripts/verify.sh 终检（pnpm 全家 + go 全家 + buf lint）；可选：简单压测脚本（vegeta/hey）与基线数字记录。
验收：新人按 README 从 clone 到 curl 通 ≤15 分钟；CI 全绿。

## 7. Pages / workflow 清理清单（Phase 0 执行）

- [ ] 删除 `scripts/deploy-github-pages.sh`
- [ ] 根 `package.json` 删除 `build:pages` script
- [ ] `README.md` 移除 GitHub Pages 部署相关段落
- [ ] `apps/demo/package.json` 移除 pages 相关字段/脚本（如 homepage、deploy 脚本）
- [ ] 全仓 `grep -ri "github.io\|pages" --exclude-dir=node_modules` 复核无残留（注意区分分页 pagination 误报）

## 8. 风险与对策

| 风险                            | 对策                                                              |
| ------------------------------- | ----------------------------------------------------------------- |
| 跟进模型手改生成物              | AGENTS.md 铁律 + `make gen` 幂等校验进 CI（重跑 diff 非空即失败） |
| proto 设计返工                  | Phase 1 强制字段级评审，OpenAPI 先行人肉过一遍再写实现            |
| Connect-JSON 与 gRPC 行为不一致 | 集成测试双协议各跑一遍同一用例集                                  |
| 生成物提交导致 PR 噪音大        | 生成物独立 commit（`chore(gen): …`），与逻辑改动分离              |
| 本地无 buf/protoc               | 生成物已入库，平时开发不需要；改 proto 时用 go tool 固定版本运行  |

## 9. 演进预留（v1 不做，设计不堵死）

- **多租户**：users 表预留 tenant_id；拦截器链预留元数据透传位；AGENTS.md 已声明新增资源表必须评估租户字段
- **对象存储**：Storage 接口即插点位，新增 S3 实现零改业务层
- **OTel**：拦截器链 metrics/log 位与 trace 同构，后续加 tracer 拦截器即可
- **第二个 Go 服务**：直接 import `packages/api-go` 复用契约

---

_本计划由访谈锁定，修改需回改决策表并注明理由。执行进度在各阶段 PR 中体现。_
