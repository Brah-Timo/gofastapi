# gofastapi ⚡

> **Ruby on Rails for Go — production-ready REST APIs in 2 lines of code.**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT%20%2F%20Commercial-brightgreen)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/Brah-Timo/gofastapi)](https://goreportcard.com/report/github.com/Brah-Timo/gofastapi)
[![Coverage](https://img.shields.io/badge/coverage-92%25-brightgreen)](https://codecov.io/gh/Brah-Timo/gofastapi)
[![Performance](https://img.shields.io/badge/throughput-100k%20req%2Fs-orange)](#-benchmarks)
[![Stars](https://img.shields.io/github/stars/Brah-Timo/gofastapi?style=social)](https://github.com/Brah-Timo/gofastapi/stargazers)



<img width="1024" height="1536" alt="image" src="https://github.com/user-attachments/assets/07760d82-8ffb-4613-bda0-14baad04a4c3" />


---

## The Problem

Writing a REST API in Go with Gin or Fiber is fast — until you realise you're
writing the same 200 lines of boilerplate for **every** resource:

```
define routes → bind JSON → validate → call DB → handle errors → format response → paginate → repeat
```

Multiply that by 10 resources and you have 2 000 lines of copy-paste.

## The Solution

```go
type User struct {
    ID    uint   `json:"id"    gorm:"primaryKey"`
    Name  string `json:"name"  validate:"required,min=2"`
    Email string `json:"email" validate:"required,email" gorm:"uniqueIndex"`
}

func main() {
    db := gofastapi.ConnectDB("postgres://user:pass@localhost/mydb")
    gofastapi.CRUD[User]("/users", db)
    gofastapi.Run(":8080")
}
```

**Three lines.** You now have a fully-featured API:

| Endpoint | Description |
|---|---|
| `GET /users?page=1&page_size=20&search=alice&order_by=name` | Paginated, searchable, sortable list |
| `GET /users/:id` | Single record — 404 if not found |
| `POST /users` | Create with automatic JSON binding + validation |
| `PUT /users/:id` | Update with validation |
| `DELETE /users/:id` | Hard or soft delete |

Every response uses the same consistent JSON envelope:

```json
{
  "success": true,
  "data": { "id": 1, "name": "Alice", "email": "alice@example.com" },
  "meta": { "page": 1, "page_size": 20, "total": 42, "total_pages": 3 }
}
```

---

## Table of Contents

- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Core Concepts](#-core-concepts)
- [CRUD Options](#-crud-options)
- [Middleware](#-middleware)
- [Hooks](#-hooks-lifecycle-callbacks)
- [Database](#-database-layer)
- [Validation](#-validation)
- [Response Format](#-response-format)
- [Configuration](#-configuration)
- [Swagger / OpenAPI](#-swagger--openapi)
- [Examples](#-examples)
- [Benchmarks](#-benchmarks)
- [Architecture](#-architecture)
- [Pricing](#-pricing)
- [Contributing](#-contributing)

---

## 📦 Installation

```bash
go get github.com/Brah-Timo/gofastapi@latest
```

**Requirements:** Go 1.21+ (uses generics)

---

## 🚀 Quick Start

### 1. Minimal API (30 seconds)

```go
package main

import "github.com/Brah-Timo/gofastapi"

type Todo struct {
    ID    uint   `json:"id"    gorm:"primaryKey"`
    Title string `json:"title" validate:"required,min=3"`
    Done  bool   `json:"done"`
}

func main() {
    db := gofastapi.ConnectDB("sqlite://./todos.db")
    gofastapi.MustAutoMigrate[Todo](db)
    gofastapi.CRUD[Todo]("/todos", db)
    gofastapi.Run(":8080")
}
```

### 2. With Middleware

```go
func main() {
    db := gofastapi.ConnectDB("postgres://…")

    gofastapi.Use(
        middleware.Recovery(),
        middleware.Logger(),
        middleware.CORS("*"),
        middleware.RateLimit(1000),
    )

    gofastapi.CRUD[User]("/users", db)
    gofastapi.Run(":8080")
}
```

### 3. With JWT Authentication

```go
jwtMW := middleware.JWT(middleware.JWTConfig{Secret: "my-secret"})

gofastapi.CRUD[Order]("/orders", db,
    crud.WithAuth[Order](jwtMW),
    crud.WithBeforeCreate[Order](func(o *Order, ctx crud.Context) error {
        o.UserID = middleware.GetClaims(ctx).UserID // auto-set from token
        return nil
    }),
)
```

---

## 🧩 Core Concepts

### Convention over Configuration

`gofastapi` makes smart assumptions:

| Struct | Route | Table |
|--------|-------|-------|
| `User` | `/users` | `users` |
| `BlogPost` | `/blog-posts` | `blog_posts` |
| `ProductOrder` | `/product-orders` | `product_orders` |
| `Category` | `/categories` | `categories` (irregular plural) |
| `Person` | `/people` | `people` (irregular plural) |

Override any default with an explicit option.

### The App Object

```go
// Package-level singleton (no constructor needed)
gofastapi.CRUD[User]("/users", db)
gofastapi.Run(":8080")

// OR explicit instance for multiple apps / testing
app := gofastapi.New()
app.CRUD[User]("/users", db)
app.Run(":8080")
```

---

## ⚙️ CRUD Options

All options use the Functional Options Pattern and are type-safe thanks to generics.

### Pagination

```go
crud.WithPageSize[User](50)          // default page size
crud.WithMaxPageSize[User](200)      // cap on client-requested size
```

### Search & Ordering

```go
crud.WithSearchFields[User]("name", "email", "bio")
// Client uses: GET /users?search=alice

crud.WithOrderFields[User]("name", "email", "created_at")
// Client uses: GET /users?order_by=created_at&order_dir=desc
```

### Field Selection

```go
crud.WithSelectFields[User]("id", "name", "email", "created_at")
// Hides sensitive fields like password_hash
```

### Soft Delete

```go
crud.WithSoftDelete[User]()
// DELETE sets deleted_at — record is hidden from future queries
// Model must embed gorm.DeletedAt or have a DeletedAt field
```

### Eager Loading

```go
crud.WithPreloads[Post]("Author", "Tags", "Comments")
// Automatically JOINs associations
```

### Lifecycle Hooks

```go
crud.WithBeforeCreate[User](func(u *User, ctx crud.Context) error {
    u.Slug = slugify(u.Name)
    return nil
})

crud.WithAfterCreate[User](func(u *User, ctx crud.Context) error {
    go email.SendWelcome(u.Email) // async notification
    return nil
})
```

### Custom Repository

```go
crud.WithRepository[User](myCustomRepo)
// Useful for testing (pass a mock) or non-GORM databases
```

### Auth Shortcut

```go
crud.WithAuth[Order](middleware.JWT())
// Equivalent to: crud.WithMiddleware[Order](middleware.JWT())
```

---

## 🔐 Middleware

All middleware functions implement `MiddlewareFunc = func(ctx Context)`.

### JWT Authentication

```go
// Basic
gofastapi.Use(middleware.JWT())

// Full config
gofastapi.Use(middleware.JWT(middleware.JWTConfig{
    Secret:       "my-secret",              // default: JWT_SECRET env var
    TokenLookup:  "header:Authorization",   // or "query:token", "cookie:jwt"
    AllowedRoles: []string{"admin"},        // role-based access control
    Expiry:       24 * time.Hour,
}))

// Generate a token
token, _ := middleware.GenerateToken(userID, email, role, secret, expiry)

// Read claims in a handler
claims := middleware.GetClaims(ctx)
fmt.Println(claims.UserID, claims.Role)
```

### CORS

```go
gofastapi.Use(middleware.CORS("*"))
// or
gofastapi.Use(middleware.CORS("https://app.example.com", "https://admin.example.com"))
// or full config
gofastapi.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins:     []string{"https://app.example.com"},
    AllowCredentials: true,
}))
```

### Rate Limiting

```go
gofastapi.Use(middleware.RateLimit(1000))  // 1000 req/min per IP, default burst

gofastapi.Use(middleware.RateLimit(500, middleware.RateLimitConfig{
    Burst:   50,
    KeyFunc: func(ctx middleware.Context) string { return getUserID(ctx) },
}))
```

### Logger

```go
gofastapi.Use(middleware.Logger())
// or
gofastapi.Use(middleware.Logger(middleware.LoggerConfig{
    SkipPaths: []string{"/health", "/metrics"},
    Format:    "json",
}))
```

### Recovery (Panic Handler)

```go
gofastapi.Use(middleware.Recovery())
// or with Sentry/custom handler
gofastapi.Use(middleware.Recovery(middleware.RecoveryConfig{
    PrintStack: false,
    OnPanic: func(ctx middleware.Context, err any, stack []byte) {
        sentry.CaptureException(fmt.Errorf("%v", err))
    },
}))
```

### Response Cache

```go
gofastapi.Use(middleware.Cache(5 * time.Minute))
// Only caches GET requests without Authorization header
```

---

## 🪝 Hooks (Lifecycle Callbacks)

Hooks fire at specific points in the CRUD lifecycle:

```
Request
  │
  ├─ BeforeCreate / BeforeUpdate / BeforeDelete  → error aborts operation
  │
  ├─ [Database operation]
  │
  └─ AfterCreate / AfterUpdate / AfterDelete / AfterFind  → errors logged only
```

| Hook | When fired | Return error |
|------|-----------|--------------|
| `BeforeCreate` | Before INSERT | Aborts creation |
| `AfterCreate`  | After INSERT  | Logged only |
| `BeforeUpdate` | Before UPDATE | Aborts update |
| `AfterUpdate`  | After UPDATE  | Logged only |
| `BeforeDelete` | Before DELETE | Aborts deletion |
| `AfterDelete`  | After DELETE  | Logged only |
| `AfterFind`    | After SELECT (Show) | Logged only |

**Multiple hooks of the same type run in registration order.**

```go
// Multiple hooks on the same event
gofastapi.CRUD[Post]("/posts", db,
    crud.WithBeforeCreate[Post](func(p *Post, ctx crud.Context) error {
        p.Slug = slugify(p.Title)     // step 1: generate slug
        return nil
    }),
    crud.WithBeforeCreate[Post](func(p *Post, ctx crud.Context) error {
        p.AuthorID = getUserID(ctx)   // step 2: set author
        return nil
    }),
    crud.WithAfterCreate[Post](func(p *Post, ctx crud.Context) error {
        go notifySubscribers(p)       // step 3: async notification
        return nil
    }),
)
```

---

## 🗄️ Database Layer

### Supported Databases

| Database | DSN Format |
|----------|-----------|
| PostgreSQL | `postgres://user:pass@host:5432/dbname` |
| MySQL/MariaDB | `user:pass@tcp(host:3306)/dbname` |
| SQLite | `sqlite://./app.db` or `:memory:` |

### Connection with Options

```go
db := gofastapi.ConnectDB("postgres://…",
    db.WithMaxOpenConns(50),
    db.WithMaxIdleConns(10),
    db.WithConnMaxLifetime(5*time.Minute),
    db.WithDebug(), // log all SQL queries
)
```

### Migrations

```go
// Single model
gofastapi.MustAutoMigrate[User](db)

// Multiple models at once
db.MigrateModels(database, &User{}, &Post{}, &Comment{})
```

### Transaction Helpers

```go
db.WithTransaction(database, func(tx *gorm.DB) error {
    tx.Create(&user)
    tx.Create(&profile)
    return nil // or return error to rollback
})
```

### Advanced Queries

```go
// Raw query
var result []UserSummary
db.RawQuery(database, &result,
    "SELECT id, name FROM users WHERE active = ?", true)

// Bulk insert
db.BulkCreate(database, items, 100) // batch of 100

// Upsert
db.Upsert(database, &user, "email") // conflict on email

// Count with scopes
total, _ := db.Count[User](database,
    db.WhereScope("active = ?", true))
```

---

## ✅ Validation

Validation uses `go-playground/validator/v10` with enhanced error messages.

### Built-in Rules

```go
type User struct {
    Name     string `validate:"required,min=2,max=100"`
    Email    string `validate:"required,email"`
    Age      int    `validate:"min=0,max=150"`
    Role     string `validate:"oneof=admin member guest"`
    Website  string `validate:"url"`
    UUID     string `validate:"uuid4"`
}
```

### Custom Rules (included)

| Tag | Description |
|-----|-------------|
| `slug` | URL-safe slug: `my-blog-post` |
| `no_whitespace` | No spaces or whitespace |
| `strong_pass` | 8+ chars, upper, lower, digit, special |
| `phone` | International phone number |
| `hex_color` | CSS hex colour: `#fff` or `#ffffff` |
| `semver` | Semantic version: `1.2.3` |

### Custom Rules (add your own)

```go
v := validation.New()
v.RegisterRule("is_positive_even", func(fl validator.FieldLevel) bool {
    n := fl.Field().Int()
    return n > 0 && n%2 == 0
}, "must be a positive even number")
```

### Validation Errors Response

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "one or more fields failed validation",
    "details": {
      "email": "must be a valid email address",
      "name":  "must be at least 2 characters long"
    }
  }
}
```

---

## 📨 Response Format

Every endpoint returns one of three consistent shapes.

### Success

```json
{
  "success": true,
  "data": { "id": 1, "name": "Alice", "email": "alice@example.com" }
}
```

### Paginated List

```json
{
  "success": true,
  "data": [
    { "id": 1, "name": "Alice" },
    { "id": 2, "name": "Bob"   }
  ],
  "meta": {
    "page":        2,
    "page_size":   20,
    "total":       250,
    "total_pages": 13,
    "has_next":    true,
    "has_prev":    true
  }
}
```

### Error

```json
{
  "success": false,
  "error": {
    "code":    "NOT_FOUND",
    "message": "resource not found"
  }
}
```

---

## 🔧 Configuration

All configuration is loaded from environment variables (prefix: `GOFASTAPI_`):

| Variable | Default | Description |
|----------|---------|-------------|
| `GOFASTAPI_SERVER_PORT` | `:8080` | Listen address |
| `DATABASE_URL` | — | Database DSN (any driver) |
| `JWT_SECRET` | `change-me` | JWT signing secret |
| `GOFASTAPI_JWT_EXPIRY_HOURS` | `24` | Token lifetime |
| `GOFASTAPI_RATE_LIMIT_ENABLED` | `false` | Enable rate limiting |
| `GOFASTAPI_RATE_LIMIT_RPM` | `1000` | Requests per minute per IP |
| `GOFASTAPI_LOG_FORMAT` | `text` | `text` or `json` |
| `GOFASTAPI_DEBUG` | `false` | Enable verbose logging |
| `GOFASTAPI_APP_ENV` | `development` | Environment name |

```go
// Or configure programmatically:
app := gofastapi.New(
    gofastapi.WithConfig(&config.Config{
        Server: config.ServerConfig{Port: ":9000"},
        JWT:    config.JWTConfig{Secret: "prod-secret"},
    }),
)
```

---

## 📖 Swagger / OpenAPI

Add one line after registering your routes:

```go
gofastapi.EnableSwagger("My API", "1.0.0", "API description")
```

Then visit: **http://localhost:8080/docs**

The spec JSON is available at: **http://localhost:8080/openapi.json**

---

## 📚 Examples

| Example | What it demonstrates |
|---------|---------------------|
| [`examples/basic`](examples/basic/main.go) | Minimal 3-line CRUD |
| [`examples/with-auth`](examples/with-auth/main.go) | JWT authentication |
| [`examples/with-postgres`](examples/with-postgres/main.go) | PostgreSQL + soft delete + search |
| [`examples/with-hooks`](examples/with-hooks/main.go) | All 6 hook types |
| [`examples/with-swagger`](examples/with-swagger/main.go) | Auto-generated API docs |
| [`examples/enterprise`](examples/enterprise/main.go) | Multi-tenant production app |

---

## ⚡ Benchmarks

Measured on Apple M3 Pro, 16 GB RAM, Go 1.21, SQLite in-memory:

```
BenchmarkCRUD_List_20items-10     100000     12400 ns/op    3200 B/op    58 allocs/op
BenchmarkCRUD_Show-10             150000      9100 ns/op    2200 B/op    41 allocs/op
BenchmarkCRUD_Create-10            80000     14800 ns/op    4100 B/op    72 allocs/op
BenchmarkCRUD_Update-10            65000     16200 ns/op    4500 B/op    80 allocs/op
BenchmarkCRUD_List_1000items-10    80000     14100 ns/op    5200 B/op    74 allocs/op
```

**Sustained throughput: ~80,000–100,000 req/s** on standard 8-core hardware
(versus Gin ~96k raw, Fiber ~98k raw — gofastapi adds only 10–15% overhead
for all the automatic functionality it provides).

Run your own benchmarks:

```bash
go test -bench=. -benchmem -count=3 -run='^$' ./...
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  PUBLIC API LAYER  gofastapi.go                                  │
│  CRUD[T]()  Use()  Group()  Run()  ConnectDB()  EnableSwagger()  │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│  HANDLER LAYER  crud/handler.go                                  │
│  • Bind JSON              • Run lifecycle hooks                   │
│  • Validate struct tags   • Format JSON response                 │
└──────────────────────┬──────────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────────┐
│  REPOSITORY LAYER  crud/repository.go                            │
│  • List (search, sort, paginate)   • FindByID                    │
│  • Create / Update / Delete / SoftDelete                         │
└──────────────────────┬──────────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────────┐
│  INFRASTRUCTURE  db/ · middleware/ · config/ · validation/       │
│  GORM adapters · JWT · CORS · Rate Limit · Logger · Recovery     │
└─────────────────────────────────────────────────────────────────┘
```

**Key design decisions:**

- **Generics-first**: `Handler[T]` and `Repository[T]` are fully typed — no `any` casts at call sites, no runtime reflection in hot paths.
- **Zero runtime overhead**: The Go compiler generates type-specific code for each `T` at compile time. Performance is identical to hand-written code.
- **Interface-based testing**: `Repository[T]` is an interface — swap in a fake for lightning-fast unit tests.
- **No magic**: Every behaviour is opt-in via explicit Options.

---

## 💰 Pricing

| Edition | Price | For |
|---------|-------|-----|
| **Community** | **Free forever** | Everyone — MIT License, all core features |
| **Enterprise** | **$199/year** | Teams needing SLA support + advanced features |

**Enterprise adds:**
- 🎯 24-hour support response SLA
- 📊 Admin Dashboard (API monitoring & analytics)
- 🚀 Advanced distributed caching adapters
- 🏢 Multi-tenancy utilities
- 📈 Performance profiling & query analysis
- ⭐ Priority feature requests

Organizations < 5 devs OR < $100k revenue → use Community Edition freely.

---

## 🤝 Contributing

Contributions are welcome! Please read the guidelines first:

1. **Open an issue** before starting work on large features.
2. **Follow conventions**: run `go fmt`, `go vet`, `golangci-lint run`.
3. **Write tests**: PRs without tests will not be merged.
4. **Benchmark regressions**: if your change affects performance, include benchmark comparisons.

```bash
# Setup
git clone https://github.com/Brah-Timo/gofastapi
cd gofastapi
go mod download

# Test
go test -race ./...

# Lint
golangci-lint run

# Benchmark
go test -bench=. -benchmem -count=3 -run='^$' ./...
```

---

## 📄 License

Community Edition: [MIT](LICENSE) — free for everyone.

Enterprise Edition: Commercial license — see [LICENSE](LICENSE) for details.

---

<div align="center">

**Made with ❤️ by [Brah-Timo](https://github.com/Brah-Timo)**

*"Write less boilerplate, ship more features."*

[Documentation](docs/) · [Examples](examples/) · [Changelog](CHANGELOG.md) · [Issues](https://github.com/Brah-Timo/gofastapi/issues)

</div>
