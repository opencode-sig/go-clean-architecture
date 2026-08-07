Before any Go coding, review, debugging, troubleshooting, or setup task, load the `samber/cc-skills-golang@golang-how-to` skill first — it routes to whichever other Go skills the task needs.

## Required Go skills

The following Go skills MUST always be applied when working on this project. Load them at the start of every Go-related task, regardless of whether the user explicitly mentions them.

- `golang-code-style`
- `golang-data-structures`
- `golang-design-patterns`
- `golang-documentation`
- `golang-error-handling`
- `golang-modernize`
- `golang-naming`
- `golang-safety`
- `golang-security`
- `golang-testing`
- `golang-troubleshooting`
- `golang-observability`
- `golang-swagger`

## Project architecture

This project follows **Clean Architecture + Bounded Context** — each business module is vertically sliced with strict dependency direction.

### Directory structure

```
cmd/server/main.go          # Composition root — wiring only
internal/
├── port/                    # Shared cross-module interfaces (Tx, Cache, UserService, ArticleService)
├── <module>/                # Each business module is self-contained
│   ├── entity/              # Domain entities — pure Go, zero dependencies
│   ├── usecase/
│   │   ├── port.go          # Repository interface + external service dependencies
│   │   └── <module>.go      # Use case — business logic
│   └── adapter/
│       ├── handler/         # HTTP handler — calls use case
│       ├── repository/      # MySQL implementation — implements port
│       └── service/         # Cross-module adapter — implements port.*Service
├── infrastructure/          # Shared infrastructure
│   ├── config.go            # YAML config loading
│   ├── database.go          # DB connection
│   ├── db.go                # TxManager
│   ├── logger.go            # InitLogger + TraceHandler + lumberjack
│   ├── metrics.go           # Prometheus metrics + middleware
│   ├── cache/               # Cache backends: memory + Redis (port.Cache impls)
│   └── router.go            # Route registration (Gin)
└── web/                     # Embedded frontend artifact
    ├── embed.go             # //go:embed static
    └── static/              # Built frontend output (gitignored except placeholder)

web/                         # Frontend development (React + Vite + TailwindCSS)
├── src/
├── vite.config.ts           # outDir: ../internal/web/static
└── package.json
```

### Module rules

- Each module goes under `internal/<module>/` with its own `entity/`, `usecase/`, `adapter/`
- **Dependency direction**: `handler → usecase → entity` (inner layers never import outer)
- Modules **MUST NOT** import each other's internal packages
- Cross-module interaction goes through `internal/port/*` interfaces
- `usecase/port.go` contains **only** the `Repository` interface (no structs, no constructors)
- Use case structs + constructors + sentinel errors live in `usecase/<module>.go`
- Business errors are sentinel errors defined in the usecase package (e.g. `ErrArticleNotFound`)
- Repositories wrap sentinels: `fmt.Errorf("%w: %d", usecase.ErrXxx, id)` — never plain strings
- Handlers map errors with `errors.Is(err, usecase.ErrXxx)` — **never** import `database/sql`
- Service adapters' `Exists` methods return `(false, nil)` only for not-found sentinels; all other errors are propagated

### New module checklist (clone `internal/user` as the template)

1. `internal/<module>/entity/<module>.go` — pure Go struct, no deps
2. `internal/<module>/usecase/port.go` — `Repository` interface only; `WithTx(tx port.Tx) Repository`
3. `internal/<module>/usecase/<module>.go` — sentinel errors + `<Name>UseCase` + `New<Name>UseCase`
4. `internal/<module>/adapter/repository/<module>_mysql.go` — `New<Name>MySQL(db *gorm.DB)`, wraps sentinels (see Repository conventions)
5. `internal/<module>/adapter/handler/<module>_handler.go` — named request structs, swaggo annotations, envelope responses
6. If other modules need existence checks: add `internal/<module>/adapter/service/<module>_service.go` implementing `port.*Service` with `WithTx`
7. If single-entity reads are hot: add `internal/<module>/adapter/repository/<module>_cache.go` (see Cache conventions)
8. Wire in `cmd/server/main.go` with an import alias per layer (e.g. `articleUsecase`)
9. Cross-module transactions: `txManager.Run(ctx, func(tx port.Tx) error { ... })`, bind via `WithTx` — see `comment/usecase/comment.go` `Create` as the reference example
10. Regenerate swagger docs if handler annotations changed

### Cross-module service adapters

- `internal/<module>/adapter/service/` implements `port.*Service`
- The service adapter wraps the module's `usecase.Repository` and delegates to it
- Service adapters support `WithTx(tx port.Tx)` for transactional cross-module calls

### Transaction (Unit of Work)

- `infrastructure.TxManager` implements `port.TxManager` — `Begin/Commit/Rollback` + `Run(ctx, fn)` helper
- Every module's `Repository` interface has `WithTx(tx port.Tx) Repository`
- `port.*Service` interfaces also declare `WithTx(tx port.Tx)`, so service adapters can join transactions
- A transaction is a `*gorm.DB` session: `Begin` returns `port.Tx` wrapping it, repo `WithTx` does `tx.(*gorm.DB)`
- Cross-module transactions use `txManager.Run(ctx, func(tx port.Tx) error { ... })` — use cases depend on `port.TxManager`, never on `infrastructure`
- Reference example: `comment/usecase/comment.go` `Create` (transactional user/article existence checks + insert)

### Repository conventions

- All persistence uses **GORM** (`gorm.io/gorm` + `gorm.io/driver/mysql`); GORM never leaks past the repository layer
- `New<Name>MySQL(db *gorm.DB)` constructor takes the shared GORM handle
- `WithTx(tx port.Tx)` returns a new repository instance holding `tx.(*gorm.DB)`
- Queries use GORM chained builders (`Where`, `Order`, `First`, `Find`); map `gorm.ErrRecordNotFound` to the usecase sentinel
- **Delete must check `result.RowsAffected == 0`** — GORM does not error on zero-row delete; wrap the sentinel manually
- **List reads are paginated** — `FindAll/FindByUserID/FindByArticleID` take `(ctx, limit, offset)` and return `([]entity, total int64, err)`; use cases return `port.Page` via `port.NewPage`
- **Updates use optimistic locking** on the `Version` field — `UpdateColumns` with `WHERE id = ? AND version = ?` bumps `version + 1`; zero rows affected maps to the module's `ErrXVersionConflict` sentinel. Single-entity timestamps use `NOW()` via GORM expression
- `CreatedAt/UpdatedAt` are auto-managed by GORM (fields already named accordingly in entities)

### Error mapping conventions

- Sentinel errors are created with `port.NewCodedError(code, msg)` — they implement `port.ErrorCoder`, so `errors.Is` and `errors.As` both work
- Handlers map errors with **one** call: `port.ResponseError(c, err)`; it resolves the code from the sentinel and never leaks internal messages. Do NOT hand-roll `errors.Is` chains with `port.Error(...)` per sentinel
- Version conflicts map to code 1004 (`port.CodeVersionConflict`), rate limit to 1005, idempotency in-flight to 1006 (100x = common)

### Cache conventions

- Cache is a cross-cutting concern: the `port.Cache` interface is in `internal/port/cache.go`; backends (memory, Redis) live in `internal/infrastructure/cache/`; a factory (`NewCache`) picks the backend from `cfg.CacheType`. **The global switch is `cfg.CacheEnabled`** (`cache.enabled` in YAML, default true): `main.go` only wraps the repository with the cache decorator when it is true — `false` connects straight to the DB (no no-op backend needed). `cache.type` stays an enum (`memory`/`redis`) for backend selection; they are orthogonal. Redis topology is chosen by `cfg.RedisMode` (`single`/`cluster`): both modes read seed addresses uniformly from `cfg.RedisAddrs` (`redis.addrs`) — `newUniversalClient` builds `redis.NewClient` (first addr + `RedisDB`) or `redis.NewClusterClient` (all addrs) — and `NewRedis` accepts `redis.UniversalClient` so the wrapper is topology-agnostic
- Caching decorates the repository layer: `internal/<module>/adapter/repository/<module>_cache.go` implements the module's own `usecase.Repository` and wraps the MySQL one, so use cases are cache-agnostic
- Cache is Cache-aside: reads populate on miss, mutations invalidate the affected key after a successful write
- **`WithTx` must bypass the cache** — a cached decorator's `WithTx` returns the underlying repository's `WithTx` (transactions need uncommitted data)
- Single-entity `FindByID` reads use TTL cache; list reads (`FindAll`, `FindByUserID`) use a **short list TTL** (`listTTL`, capped at 15s min) keyed by `article:list:<scope>:<id>:<limit>:<offset>`
- List caches are invalidated on writes via the optional `CacheListInvalidator` capability (`DeleteByPrefix`, `article:list:`); backends without it degrade to short TTL
- Cache keys follow `module:entity:id` convention (e.g. `article:id:1`)
- Use a jittered TTL (`jitterTTL`, ±10%) to avoid thundering-herd expiry, and cache a short-lived empty marker (`nil` value) for not-found results to guard against cache penetration
- Cache backend failures must degrade to the database: if `Get` returns anything other than `port.ErrCacheMiss`, fall through to the repo (never fail the request because of the cache)
- Reference example: `internal/article/adapter/repository/article_cache.go`

### Handler conventions

- Handlers only parse request, validate input, call use case, return response
- Handlers use `*gin.Context` — `c.ShouldBindJSON()` for input
- **All responses use the unified envelope from `internal/port/response.go`** — HTTP status is always 200
  - Success: `port.Success(c, data)` → `{"code":0, "message":"", "data":...}`
  - Business error: `port.Error(c, code, msg)` → `{"code":1001, "message":"...", "data":null}`
  - Internal error: `port.ErrorInternal(c, msg)` → code 1999
  - Prefer `port.ResponseError(c, err)` for usecase errors (see Error conventions)
- Pagination request structs add optional `page`/`page_size` fields; handlers validate via `pageCfg.WithDefaults(...)` (injected `port.PageConfig`)
- **All HTTP methods use POST** unless explicitly overridden — no GET/PUT/DELETE
- Exception: `/healthz`, `/readyz`, `/swagger/*any`, `/metrics`, and the root catch-all use GET

### Infra middleware (production hardening)

- **Graceful shutdown**: `main.go` runs `srv.ListenAndServe` in a goroutine, waits on `signal.NotifyContext`, then `srv.Shutdown` with a 10s timeout
- **Server timeouts**: `ReadHeaderTimeout`/`ReadTimeout`/`WriteTimeout`/`IdleTimeout` come from `cfg`
- **Health**: `GET /healthz` (liveness) and `GET /readyz` (DB ping) via `NewHealthHandler(db)`
- **Rate limiting**: per-IP token bucket `NewIPRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)` as a middleware; 429-style business response code 1005
- **Idempotency**: `IdempotencyMiddleware(db)` makes POSTs with `Idempotency-Key` exactly-once. First request claims the key (`in_flight` INSERT), stores the completed response, then replays it; concurrent claims return code 1006 in-flight
- **Lightweight tracing**: `requestID()` seeds `trace_id`/`span_id`/`parent_span_id` from `X-Trace-ID` (or generates); slog's `TraceHandler` attaches them to every log record. Metrics exemplars use `trace_id`
- **Schema migration**: `infrastructure.Migrate(db, entities...)` runs `AutoMigrate` + normalizes pre-version `NULL` rows to 0 on startup

### Swagger / OpenAPI

- Handlers are annotated with swaggo (`@Summary`, `@Param`, `@Success`, `@Router`) for auto-generated OpenAPI docs
- After changing handler annotations, regenerate docs:
  ```bash
  swag init -g cmd/server/main.go --output docs
  ```
- Request body types MUST be named structs (swag cannot generate schema for anonymous types)
- Paginated responses use `port.Response{data=port.Page}`
- Add `example`, `minimum`, `enums` struct tags to request types for richer Swagger UI output
- Swagger UI is served at `GET /swagger/index.html` (excluded from logging/metrics middleware)

### Prometheus / Metrics

### Logging

- Use `slog` (standard library) directly — no wrapper package
- `infrastructure.InitLogger(cfg)` is called once in `main.go`
- Outputs JSON to both stdout and file (`logs/app.log`)
- Uses `github.com/natefinch/lumberjack` for log rotation (size + daily)
- Use case / adapter code calls `slog.Info()`, `slog.Error()`, etc. globally
- Within request contexts, use `slog.InfoContext(ctx, ...)` to auto-attach `request_id`/`trace_id`/`span_id`/`parent_span_id` (seeded by the `requestID` middleware)
- Test helpers can silence logs with `slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))`

### Frontend

- Source code in `web/` root directory (React + Vite + TailwindCSS)
- Build output goes to `internal/web/static/`
- `make build-web` rebuilds frontend
- `make build` builds both frontend and Go binary
