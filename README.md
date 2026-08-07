# 智索

## 技术栈

- **后端**: Go, Gin (router), GORM + MySQL, Redis/内存缓存, Prometheus, slog
- **前端**: React, TypeScript, Vite, TailwindCSS
- **文档**: Swagger/OpenAPI (swaggo)
- **部署**: 单二进制 (go:embed)

## 快速开始

```bash
# 初始化前端
make build-web

# 编译后端
make build-server

# 构建全部
make build

# 开发模式（终端 1：前端热更新）
make dev-web

# 开发模式（终端 2：后端开发服务器）
make dev-server

# 指定配置文件启动
go run ./cmd/server/ -config config/production.yaml
```

### 配置

配置使用 YAML 文件，默认路径 `config/config.yaml`（查找顺序：`-config` 指定路径 → `config.yaml` → `config/development.yaml` → 代码默认值）。

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `server.port` | `8080` | 服务端口 |
| `server.read_header_timeout` | `5` | 读请求头超时（秒） |
| `server.read_timeout` | `30` | 读请求超时（秒） |
| `server.write_timeout` | `30` | 写响应超时（秒） |
| `server.idle_timeout` | `120` | 连接空闲超时（秒） |
| `mysql.host` | `127.0.0.1` | 数据库主机 |
| `mysql.port` | `3306` | 数据库端口 |
| `mysql.user` | `root` | 数据库用户 |
| `mysql.password` | `password` | 数据库密码 |
| `mysql.name` | `zhisuo` | 数据库名 |
| `mysql.charset` | `utf8mb4` | 字符集 |
| `mysql.parse_time` | `True` | 自动解析 time.Time |
| `mysql.loc` | `Local` | 时区 |
| `mysql.max_open_conns` | `25` | 连接池最大打开连接数 |
| `mysql.max_idle_conns` | `5` | 连接池最大空闲连接数 |
| `mysql.conn_max_lifetime` | `300` | 连接最大生命周期（秒） |
| `cache.enabled` | `true` | 全局缓存开关：`false` 时绕过所有缓存装饰器（直连 DB） |
| `cache.type` | `memory` | 缓存后端 (`memory`/`redis`)，仅 `enabled=true` 时生效 |
| `cache.ttl` | `300` | 缓存默认 TTL（秒） |
| `redis.host` | `127.0.0.1` | Redis 主机 |
| `redis.port` | `6379` | Redis 端口 |
| `redis.db` | `0` | Redis 数据库编号 |
| `rate_limit.rps` | `10` | 每 IP 每秒请求数 |
| `rate_limit.burst` | `50` | 每 IP 令牌桶容量 |
| `pagination.default_page_size` | `20` | 列表默认页大小 |
| `pagination.max_page_size` | `100` | 列表页大小上限 |
| `log.level` | `info` | 日志级别 (debug/info/warn/error) |
| `log.file` | `logs/app.log` | 日志文件路径 |
| `log.max_size` | `100` | 单个日志文件最大 MB |
| `log.max_age` | `30` | 日志保留天数 |
| `log.max_backups` | `7` | 保留的旧日志文件数 |

### Swagger 文档

启动后访问 `http://localhost:8080/swagger/index.html`。更新 handler 注解后重新生成：

```bash
swag init -g cmd/server/main.go --output docs
```

## 项目结构

```
├── config/                      # 配置文件（按环境独立完整文件）
│   ├── development.yaml         #   开发环境
│   └── production.yaml          #   生产环境
├── docs/                        # Swagger 生成产物（swag init）
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── cmd/server/main.go           # 入口（组合根）：装配 + 优雅关闭 + AutoMigrate
├── internal/
│   ├── port/                    # 共享接口与约定
│   │   ├── tx.go                # Tx / TxManager
│   │   ├── cache.go             # Cache + CacheListInvalidator
│   │   ├── user.go              # UserService
│   │   ├── article.go           # ArticleService
│   │   ├── response.go          # 统一响应封装
│   │   └── types.go             # ErrorCoder / Page / PageConfig / 业务错误码
│   ├── user/                    # 业务模块：用户
│   │   ├── entity/
│   │   ├── usecase/
│   │   │   ├── port.go
│   │   │   └── user.go
│   │   └── adapter/
│   │       ├── handler/
│   │       ├── repository/
│   │       └── service/
│   ├── article/                 # 业务模块：文章
│   │   └── (同 user 结构，含 repository/article_cache.go 缓存装饰器)
│   ├── comment/                 # 业务模块：评论
│   │   └── (同 user 结构)
│   ├── infrastructure/          # 基础设施
│   │   ├── config.go
│   │   ├── database.go          # GORM 连接 + 连接池
│   │   ├── db.go                # TxManager
│   │   ├── migrate.go           # AutoMigrate + 版本字段归一化
│   │   ├── health.go            # /healthz + /readyz
│   │   ├── ratelimit.go         # 每 IP 令牌桶限流
│   │   ├── idempotency.go       # Idempotency-Key 中间件
│   │   ├── logger.go            # InitLogger + TraceHandler（trace/span 注入）
│   │   ├── metrics.go           # Prometheus + exemplar
│   │   ├── cache/               # 缓存后端：memory + redis
│   │   └── router.go
│   └── web/                     # 前端构建产物
│       ├── embed.go
│       └── static/
├── web/                         # 前端源码
│   ├── src/
│   ├── vite.config.ts
│   └── package.json
└── Makefile
```

## 架构

**Clean Architecture + Bounded Context**，每个业务模块垂直独立：

- `entity` → `usecase` → `adapter`（单向依赖）
- 模块间通过 `internal/port/` 接口交互
- 跨模块事务通过 `TxManager.WithTx()` 编排

详细规范见 [AGENTS.md](AGENTS.md)。

## API

### Users

| 路径 | 说明 |
|------|------|
| POST `/api/v1/users/create` | 创建用户 |
| POST `/api/v1/users/delete` | 删除用户 |
| POST `/api/v1/users/get` | 用户详情 |
| POST `/api/v1/users/list` | 用户列表（分页） |
| POST `/api/v1/users/update` | 更新用户（乐观锁） |

### Articles

| 路径 | 说明 |
|------|------|
| POST `/api/v1/articles/create` | 创建文章 |
| POST `/api/v1/articles/delete` | 删除文章 |
| POST `/api/v1/articles/get` | 文章详情 |
| POST `/api/v1/articles/list` | 文章列表（分页） |
| POST `/api/v1/articles/by-user` | 指定用户的文章列表（分页） |
| POST `/api/v1/articles/update` | 更新文章（乐观锁） |

### Comments

| 路径 | 说明 |
|------|------|
| POST `/api/v1/comments/create` | 创建评论 |
| POST `/api/v1/comments/delete` | 删除评论 |
| POST `/api/v1/comments/list` | 评论列表（分页） |

> 所有接口统一使用 POST 方法，参数通过 JSON body 传递。

### 运维端点

| 路径 | 说明 |
|------|------|
| GET `/healthz` | 存活探针（Liveness） |
| GET `/readyz` | 就绪探针（Readiness，DB ping） |
| GET `/metrics` | Prometheus 指标 |
| GET `/swagger/index.html` | Swagger UI |

### 分页

列表接口接受可选 `page` / `page_size` 字段，返回统一分页结构：

```json
{"code":0, "message":"", "data":{"items":[...], "total":42, "page":1, "page_size":20}}
```

页大小有上限（`pagination.max_page_size`，默认 100），超限自动钳制。

### 幂等

创建类请求可携带 `Idempotency-Key` 请求头实现恰好一次语义：
- 首次请求执行并缓存完整响应，重试/重放返回完全相同的结果
- 同一 key 并发请求返回 `1006`（in-flight），客户端可稍后重试

```bash
curl -X POST http://localhost:8080/api/v1/users/create \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: order-123' \
  -d '{"username":"jane","email":"jane@x.com"}'
```

### 乐观锁（并发安全）

`update` 接口接受可选 `version` 字段（来自先前读取的实体）。传入非零版本时，若数据库中版本已变化则更新失败并返回 `1004`：

```bash
# 1. 读取实体，得到 version=1
# 2. 携带该版本更新
curl -X POST http://localhost:8080/api/v1/articles/update \
  -H 'Content-Type: application/json' \
  -d '{"id":1, "version":1, "title":"新标题", "content":"新内容"}'
```

### 响应格式

所有接口 HTTP 状态码恒为 200，通过 `code` 区分业务成功/失败：

```json
// 成功
{"code":0, "message":"", "data":{...}}
// 业务错误
{"code":1001, "message":"参数错误", "data":null}
```

| code | 含义 |
|------|------|
| `0` | 成功 |
| `1001` | 参数错误 |
| `1002` | 未授权 |
| `1003` | 资源不存在 |
| `1004` | 版本冲突（乐观锁） |
| `1005` | 请求过于频繁（限流） |
| `1006` | 幂等请求处理中（in-flight） |
| `1999` | 内部错误 |
| `2001` | 用户不存在 |
| `3001` | 文章不存在 |
| `4001` | 评论不存在 |

### Swagger

| 路径 | 说明 |
|------|------|
| GET `/swagger/index.html` | Swagger UI |
| GET `/swagger/doc.json` | OpenAPI JSON 规范 |
| GET `/swagger/doc.yaml` | OpenAPI YAML 规范 |

## 生产化特性

- **优雅关闭**: 收到 SIGTERM/SIGINT 后停止接收新请求，10 秒内排空在途请求
- **健康检查**: `/healthz` 存活探针、`/readyz` 就绪探针（探测数据库连通性）
- **限流**: 每 IP 令牌桶（`golang.org/x/time/rate`），超限返回 `1005`
- **幂等**: `Idempotency-Key` 头实现创建类请求恰好一次
- **乐观锁**: 所有更新走 `Version` 字段条件更新，防并发覆盖
- **列表缓存**: 文章列表缓存（短 TTL + 写入按前缀失效），单实体读缓存 + 穿透保护
- **轻量链路**: 每个请求携带 `trace_id`/`span_id`/`parent_span_id` 贯穿日志与 Prometheus exemplar
- **统一错误码**: 业务错误经 `ErrorCoder` 哨兵映射为全局错误码，内部错误细节不泄漏给客户端
- **Schema 迁移**: 启动时 `AutoMigrate` 建表，并自动将历史 `NULL` 版本行归一化为 `0`
