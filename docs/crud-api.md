# CRUD API Reference

## gofastapi.CRUD[T]

```go
func CRUD[T any](prefix string, database db.Database, opts ...crud.Option[T])
```

Registers five REST endpoints for type T. T can be any struct — no interface
needs to be implemented.

### Generated Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `{prefix}` | List items (paginated) |
| `GET` | `{prefix}/:id` | Get single item |
| `POST` | `{prefix}` | Create item |
| `PUT` | `{prefix}/:id` | Update item |
| `DELETE` | `{prefix}/:id` | Delete item |

### Query Parameters (List endpoint)

| Parameter | Default | Description |
|-----------|---------|-------------|
| `page` | `1` | 1-based page number |
| `page_size` | `20` | Items per page |
| `search` | — | Free-text search (requires `WithSearchFields`) |
| `order_by` | `id` | Column to sort by |
| `order_dir` | `asc` | Sort direction: `asc` or `desc` |

## Available Options

### WithPageSize

```go
crud.WithPageSize[User](50)
```

Sets the default page size returned by the List endpoint.
The client can override this with `?page_size=N` up to the MaxPageSize cap.

### WithMaxPageSize

```go
crud.WithMaxPageSize[User](500)
```

Sets the maximum page size the client may request.
Requests exceeding this value are silently capped.

### WithSoftDelete

```go
crud.WithSoftDelete[User]()
```

Enables soft-delete: DELETE requests set `deleted_at` instead of removing the row.
The model must embed `gorm.DeletedAt`.

### WithSearchFields

```go
crud.WithSearchFields[User]("name", "email", "bio")
```

Configures which columns the `?search=` parameter is applied to.
The search is case-insensitive LIKE matching.

### WithOrderFields

```go
crud.WithOrderFields[User]("name", "created_at", "email")
```

Whitelists columns that clients may use in `?order_by=`.
When empty, any column without special characters is permitted.

### WithSelectFields

```go
crud.WithSelectFields[User]("id", "name", "email", "created_at")
```

Limits which columns are returned in responses.
Use to hide sensitive fields (e.g. password hashes).

### WithPreloads

```go
crud.WithPreloads[Post]("Author", "Tags")
```

Eagerly loads GORM associations on every request.

### WithRepository

```go
crud.WithRepository[User](myCustomRepo)
```

Replaces the default GORM repository with a custom implementation.
Useful for testing with mocks or non-GORM backends.

### WithAuth

```go
jwtMW := middleware.JWT()
crud.WithAuth[Order](jwtMW)
```

Protects all five endpoints with JWT authentication middleware.

### WithMiddleware

```go
crud.WithMiddleware[Product](loggingMW, auditMW)
```

Adds handler-scoped middleware that runs only for this CRUD group.

## Hook Options

See [Hooks Guide](hooks.md) for full documentation.

```go
crud.WithBeforeCreate[T](fn)
crud.WithAfterCreate[T](fn)
crud.WithBeforeUpdate[T](fn)
crud.WithAfterUpdate[T](fn)
crud.WithBeforeDelete[T](fn)
crud.WithAfterDelete[T](fn)
crud.WithAfterFind[T](fn)
```
