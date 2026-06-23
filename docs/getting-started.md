# Getting Started with gofastapi

## Prerequisites

- Go 1.21 or later
- A database: PostgreSQL, MySQL, or SQLite (SQLite works with zero setup)

## Installation

```bash
go get github.com/Brah-Timo/gofastapi@latest
```

## Your First API (2 minutes)

Create a new file `main.go`:

```go
package main

import "github.com/Brah-Timo/gofastapi"

type Task struct {
    ID       uint   `json:"id"       gorm:"primaryKey"`
    Title    string `json:"title"    validate:"required,min=3"`
    Done     bool   `json:"done"     gorm:"default:false"`
    Priority int    `json:"priority" validate:"min=1,max=5"`
}

func main() {
    db := gofastapi.ConnectDB("sqlite://./tasks.db")
    gofastapi.MustAutoMigrate[Task](db)
    gofastapi.CRUD[Task]("/tasks", db)
    gofastapi.Run(":8080")
}
```

Run it:

```bash
go run main.go
```

Test it:

```bash
# Create a task
curl -X POST http://localhost:8080/tasks \
     -H "Content-Type: application/json" \
     -d '{"title":"Buy groceries","priority":3}'

# List tasks
curl "http://localhost:8080/tasks?page=1&page_size=10"

# Get one task
curl http://localhost:8080/tasks/1

# Update
curl -X PUT http://localhost:8080/tasks/1 \
     -H "Content-Type: application/json" \
     -d '{"title":"Buy groceries","priority":3,"done":true}'

# Delete
curl -X DELETE http://localhost:8080/tasks/1
```

## Adding Middleware

```go
import (
    "github.com/Brah-Timo/gofastapi"
    "github.com/Brah-Timo/gofastapi/middleware"
)

func main() {
    db := gofastapi.ConnectDB("sqlite://./tasks.db")
    gofastapi.MustAutoMigrate[Task](db)

    // Add middleware BEFORE registering routes
    gofastapi.Use(
        middleware.Recovery(),    // handle panics
        middleware.Logger(),      // log every request
        middleware.CORS("*"),     // allow all origins
    )

    gofastapi.CRUD[Task]("/tasks", db)
    gofastapi.Run(":8080")
}
```

## Adding Validation

Add `validate` struct tags to your model:

```go
type Task struct {
    ID       uint   `json:"id"       gorm:"primaryKey"`
    Title    string `json:"title"    validate:"required,min=3,max=200"`
    Done     bool   `json:"done"`
    Priority int    `json:"priority" validate:"required,min=1,max=5"`
    Tags     string `json:"tags"     validate:"omitempty,max=500"`
}
```

Validation errors are returned as structured JSON automatically:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "one or more fields failed validation",
    "details": {
      "title":    "must be at least 3 characters long",
      "priority": "is required"
    }
  }
}
```

## What's Next?

- [CRUD API Reference](crud-api.md)
- [Middleware Guide](middleware.md)
- [Database Guide](database.md)
- [Hooks Guide](hooks.md)
- [Swagger/OpenAPI](swagger.md)
- [Examples](../examples/)
