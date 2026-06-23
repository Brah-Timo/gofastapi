# Swagger / OpenAPI Documentation

## Enable Swagger UI

Call `EnableSwagger` **after** all routes are registered:

```go
func main() {
    db := gofastapi.ConnectDB("postgres://…")
    gofastapi.CRUD[User]("/users", db)
    gofastapi.CRUD[Post]("/posts", db)

    // Must be called LAST
    gofastapi.EnableSwagger("My API", "1.0.0", "API description")

    gofastapi.Run(":8080")
}
```

Then open: **http://localhost:8080/docs**

The raw OpenAPI JSON is at: **http://localhost:8080/openapi.json**

## Enterprise License Note

Advanced Swagger features (request body schemas, response schemas, example values)
are available in the Enterprise Edition.
