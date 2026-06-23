# Hooks — Lifecycle Callbacks

Hooks let you inject business logic at specific points in the CRUD lifecycle
without modifying framework internals.

## Hook Points

```
HTTP Request
     │
     ├── BeforeCreate  ← return error to abort
     ├── [INSERT]
     └── AfterCreate   ← errors logged only
     
     ├── BeforeUpdate  ← return error to abort
     ├── [UPDATE]
     └── AfterUpdate   ← errors logged only
     
     ├── BeforeDelete  ← return error to abort
     ├── [DELETE]
     └── AfterDelete   ← errors logged only
     
     ├── [SELECT] (Show endpoint)
     └── AfterFind     ← errors logged only
```

## Hook Signature

```go
type HookFunc[T any] func(item *T, ctx hooks.Context) error
```

- `item` — pointer to the model (modifications are reflected in the response)
- `ctx` — request context (access claims, headers, IP, custom values)

## Examples

### Auto-populate fields

```go
crud.WithBeforeCreate[BlogPost](func(p *BlogPost, ctx crud.Context) error {
    p.Slug = slugify(p.Title)
    p.AuthorID = middleware.RequireUserID(ctx)
    return nil
})
```

### Prevent forbidden operations

```go
crud.WithBeforeDelete[Order](func(o *Order, ctx crud.Context) error {
    if o.Status == "shipped" {
        return errors.New("cannot delete a shipped order")
    }
    return nil
})
```

### Send async notifications

```go
crud.WithAfterCreate[User](func(u *User, ctx crud.Context) error {
    go func() {
        email.SendWelcome(u.Email, u.Name)
        slack.Notify(fmt.Sprintf("New user: %s", u.Email))
    }()
    return nil // returning nil even for async work
})
```

### Enforce business rules

```go
crud.WithBeforeCreate[Product](func(p *Product, ctx crud.Context) error {
    if p.Price > 1_000_000 {
        return errors.New("price cannot exceed 1,000,000")
    }
    if p.Stock < 0 {
        return errors.New("stock cannot be negative")
    }
    return nil
})
```

### Audit log

```go
crud.WithAfterCreate[Order](func(o *Order, ctx crud.Context) error {
    go auditLog.Record(audit.Entry{
        Action:     "order.create",
        ResourceID: o.ID,
        UserID:     middleware.RequireUserID(ctx),
        IP:         ctx.ClientIP(),
    })
    return nil
})
```

### Transform response (AfterFind)

```go
crud.WithAfterFind[User](func(u *User, ctx crud.Context) error {
    // Mask partial email for privacy
    parts := strings.Split(u.Email, "@")
    if len(parts) == 2 {
        u.Email = parts[0][:1] + "***@" + parts[1]
    }
    return nil
})
```

## Multiple Hooks of the Same Type

Hooks of the same type run in registration order:

```go
gofastapi.CRUD[Post]("/posts", db,
    // Hook 1: validate
    crud.WithBeforeCreate[Post](func(p *Post, ctx crud.Context) error {
        if len(p.Title) < 5 {
            return errors.New("title too short")
        }
        return nil
    }),
    // Hook 2: populate (only runs if hook 1 passes)
    crud.WithBeforeCreate[Post](func(p *Post, ctx crud.Context) error {
        p.Slug = slugify(p.Title)
        p.AuthorID = middleware.RequireUserID(ctx)
        return nil
    }),
)
```

## Error Semantics

| Hook type | Error behaviour |
|-----------|----------------|
| `BeforeCreate/Update/Delete` | Returns 400 to client, **aborts** the operation |
| `AfterCreate/Update/Delete/Find` | Logged internally, **does not** affect HTTP response |
