# Database Guide

## Supported Drivers

| Driver | DSN format |
|--------|-----------|
| PostgreSQL | `postgres://user:pass@host:5432/dbname?sslmode=disable` |
| MySQL/MariaDB | `user:pass@tcp(host:3306)/dbname?parseTime=True` |
| SQLite | `sqlite://./app.db` or `:memory:` |

## Connecting

```go
// Auto-detect driver from DSN
db := gofastapi.ConnectDB("postgres://…")

// With connection pool options
db := gofastapi.ConnectDB("postgres://…",
    db.WithMaxOpenConns(50),
    db.WithMaxIdleConns(10),
    db.WithConnMaxLifetime(5*time.Minute),
    db.WithDebug(),                           // log SQL
    db.WithSlowThreshold(100*time.Millisecond), // warn on slow queries
)
```

## Migrations

```go
// Single model
gofastapi.MustAutoMigrate[User](database)

// Multiple models
db.MigrateModels(database, &User{}, &Post{}, &Comment{}, &Tag{})
```

AutoMigrate creates tables, adds missing columns, and adds missing indexes.
It does **not** delete columns or change existing data.

## Using Raw GORM

Access the underlying `*gorm.DB` for advanced queries:

```go
gormDB := database.DB()

// Custom query
var users []User
gormDB.Where("age > ? AND active = ?", 18, true).
      Order("created_at DESC").
      Limit(10).
      Find(&users)

// Associations
gormDB.Preload("Posts.Tags").Find(&users)
```

## Transactions

```go
db.WithTransaction(database, func(tx *gorm.DB) error {
    if err := tx.Create(&user).Error; err != nil {
        return err // auto-rollback
    }
    if err := tx.Create(&profile).Error; err != nil {
        return err // auto-rollback
    }
    return nil // auto-commit
})
```

## Raw Queries

```go
// SELECT query with struct scanning
var results []struct {
    Name  string
    Total int
}
db.RawQuery(database, &results,
    "SELECT name, COUNT(*) as total FROM orders GROUP BY name")

// Execute (INSERT/UPDATE/DELETE/DDL)
db.RawExec(database,
    "UPDATE users SET last_login = ? WHERE id = ?", time.Now(), userID)
```

## Soft Delete

1. Embed `gorm.DeletedAt` in your model:

```go
type Post struct {
    gorm.Model  // includes gorm.DeletedAt
    Title string
}
```

2. Enable soft delete in the CRUD options:

```go
gofastapi.CRUD[Post]("/posts", db,
    crud.WithSoftDelete[Post](),
)
```

3. Restore a soft-deleted record:

```go
db.Restore[Post](database, postID)
```

## Bulk Operations

```go
// Insert 1000 items in batches of 100
items := make([]Product, 1000)
// ... fill items ...
db.BulkCreate(database, items, 100)
```

## Upsert

```go
// Insert or update on email conflict
db.Upsert(database, &user, "email")
```

## Connection Health

```go
// Check if DB is responding
if !db.IsHealthy(database) {
    log.Fatal("database is unreachable")
}

// Connection pool statistics
stats := db.Stats(database)
fmt.Printf("open: %d, idle: %d\n", stats.OpenConnections, stats.Idle)
```

## SQLite for Local Development

```go
// File-based SQLite
db := gofastapi.ConnectDB("sqlite://./myapp.db")

// In-memory (tests / CI)
db := gofastapi.ConnectDB(":memory:")
// or
db, _ := db.InMemory()
```
