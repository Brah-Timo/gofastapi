# Changelog

All notable changes to `gofastapi` are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [Unreleased]

### Added
- `WithAfterFind` hook for post-process single-record responses
- `JWTOptional` middleware for endpoints where auth is helpful but not required
- `BulkCreate` helper in `db/gorm_adapter.go`
- `Restore` helper to un-delete soft-deleted records

---

## [1.0.0] — 2026-06-23

### Added — Core framework
- `gofastapi.CRUD[T]()` — one-liner that registers 5 REST endpoints for any struct
- `gofastapi.Run()` with graceful shutdown (SIGINT / SIGTERM)
- `gofastapi.Use()` for global middleware registration
- `gofastapi.Group()` for URL-prefix route groups
- `gofastapi.ConnectDB()` with automatic driver detection (postgres, mysql, sqlite)
- `gofastapi.AutoMigrate[T]()` and `MustAutoMigrate[T]()`
- `gofastapi.EnableSwagger()` for zero-config OpenAPI 3.0 UI

### Added — CRUD layer (`crud/`)
- Generic `Handler[T]` with full HTTP lifecycle
- Generic `Repository[T]` interface + `GORMRepository[T]` default implementation
- Functional Options API (`WithPageSize`, `WithSoftDelete`, `WithSearchFields`, etc.)
- Full validation via `go-playground/validator/v10` with JSON field names in errors
- Hooks system: `BeforeCreate`, `AfterCreate`, `BeforeUpdate`, `AfterUpdate`, `BeforeDelete`, `AfterDelete`, `AfterFind`
- Smart error classification: DB errors → HTTP status codes

### Added — Database layer (`db/`)
- `AutoConnect()` with automatic driver detection
- `GORMRepository[T]` with pagination, search, ordering, preloads
- PostgreSQL, MySQL, SQLite adapters
- `WithTransaction()` and `WithTransactionCtx()`
- `BulkCreate[T]()`, `Count[T]()`, `Upsert[T]()`, `Restore[T]()`
- `sqlx` adapter for named queries

### Added — Middleware (`middleware/`)
- `JWT()` — HMAC HS256 authentication with role-based access control
- `JWTOptional()` — optional JWT authentication
- `CORS()` — Cross-Origin Resource Sharing with wildcard subdomain support
- `RateLimit()` — per-IP token-bucket rate limiter (go-cache + rate package)
- `Logger()` — structured request/response logging (text or JSON format)
- `Recovery()` — panic recovery with stack trace logging + custom handler
- `Cache()` — in-memory TTL response cache

### Added — Response (`response/`)
- Unified `APIResponse[T]` envelope for all responses
- `Success()`, `Created()`, `OK()` helpers
- `Paginated()` with full metadata (total, total_pages, has_next, has_prev)
- `Error()`, `ErrorMsg()`, `ValidationErrors()`, `NotFound()`, `Forbidden()`

### Added — Validation (`validation/`)
- `Validator` wrapping `go-playground/validator/v10`
- Human-readable error messages with JSON field names
- Built-in custom rules: `slug`, `no_whitespace`, `strong_pass`, `phone`, `hex_color`, `semver`
- `RegisterRule()` for adding project-specific rules

### Added — Config (`config/`)
- `LoadFromEnv()` with `GOFASTAPI_*` prefix
- Alias support: `DATABASE_URL`, `PORT`, `JWT_SECRET`
- `Defaults()` for programmatic config construction
- `IsDevelopment()`, `IsProduction()` helpers

### Added — Internal (`internal/`)
- `reflect` — cached struct metadata (`TypeInfo`, `FieldInfo`)
- `naming` — automatic route/table naming with English pluralisation
- `utils` — environment helpers, string/number/time utilities

### Added — Examples
- `examples/basic` — 5-line CRUD
- `examples/with-auth` — JWT authentication
- `examples/with-postgres` — PostgreSQL + soft delete + search
- `examples/with-hooks` — lifecycle hooks (slug, notifications, protection)
- `examples/with-swagger` — auto-generated API docs
- `examples/enterprise` — full multi-tenant production setup

### Added — CI/CD
- GitHub Actions: test matrix (Go 1.21, 1.22), lint, benchmarks, security scan
- Auto-release workflow on tag push

### Performance
- List (20 items):  ~12 400 ns/op,  3 200 B/op
- Create:           ~14 800 ns/op,  4 100 B/op
- Show (by ID):     ~ 9 100 ns/op,  2 200 B/op
- Sustained throughput: ~80 000–100 000 req/s on 8-core hardware

---

## Pre-release history

- `v0.3.0` — Added Swagger UI, Response package, Enterprise examples
- `v0.2.0` — Added middleware suite, validation, hooks system
- `v0.1.0` — Initial proof-of-concept: generic CRUD handler + GORM repo
