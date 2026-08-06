# 智索

## 技术栈

- **后端**: Go, Gin (router), MySQL, Prometheus
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

配置使用 YAML 文件，默认路径 `config/config.yaml`。

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `server.port` | `8080` | 服务端口 |
| `mysql.host` | `127.0.0.1` | 数据库主机 |
| `mysql.port` | `3306` | 数据库端口 |
| `mysql.user` | `root` | 数据库用户 |
| `mysql.password` | `password` | 数据库密码 |
| `mysql.name` | `zhisuo` | 数据库名 |
| `mysql.charset` | `utf8mb4` | 字符集 |
| `mysql.parse_time` | `True` | 自动解析 time.Time |
| `mysql.loc` | `Local` | 时区 |
| `log.level` | `info` | 日志级别 (debug/info/warn/error) |
| `log.file` | `logs/app.log` | 日志文件路径 |
| `log.max_size` | `100` | 单个日志文件最大 MB |
| `log.max_age` | `30` | 日志保留天数 |
| `log.max_backups` | `7` | 保留的旧日志文件数 |

> 查找顺序：`-config` 指定路径 → `config.yaml` → `config/development.yaml` → 代码默认值

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
├── cmd/server/main.go           # 入口（组合根）
├── internal/
│   ├── port/                    # 共享接口
│   │   ├── tx.go                # Tx / TxManager
│   │   ├── user.go              # UserService
│   │   ├── article.go           # ArticleService
│   │   └── response.go          # 统一响应封装
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
│   │   └── (同 user 结构)
│   ├── comment/                 # 业务模块：评论
│   │   └── (同 user 结构)
│   ├── infrastructure/          # 基础设施
│   │   ├── config.go
│   │   ├── database.go
│   │   ├── db.go                # Querier + TxManager
│   │   ├── logger.go            # InitLogger + TraceHandler
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
| POST `/api/v1/users/list` | 用户列表 |
| POST `/api/v1/users/update` | 更新用户 |

### Articles

| 路径 | 说明 |
|------|------|
| POST `/api/v1/articles/create` | 创建文章 |
| POST `/api/v1/articles/delete` | 删除文章 |
| POST `/api/v1/articles/get` | 文章详情 |
| POST `/api/v1/articles/list` | 文章列表 |
| POST `/api/v1/articles/update` | 更新文章 |

### Comments

| 路径 | 说明 |
|------|------|
| POST `/api/v1/comments/create` | 创建评论 |
| POST `/api/v1/comments/delete` | 删除评论 |
| POST `/api/v1/comments/list` | 评论列表 |

> 所有接口统一使用 POST 方法，参数通过 JSON body 传递。

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
