# Middleware Reference

## JWT Authentication

```go
import "github.com/Brah-Timo/gofastapi/middleware"

// Minimal — reads JWT_SECRET from env
gofastapi.Use(middleware.JWT())

// Full config
gofastapi.Use(middleware.JWT(middleware.JWTConfig{
    Secret:       "super-secret",
    TokenLookup:  "header:Authorization",  // or "query:token", "cookie:jwt"
    ContextKey:   "user",
    AllowedRoles: []string{"admin", "manager"},
    Expiry:       24 * time.Hour,
}))

// Generate a token
token, err := middleware.GenerateToken(userID, email, role, secret, expiry)

// Read claims in any handler
claims := middleware.GetClaims(ctx)  // returns *JWTClaims or nil
userID := middleware.RequireUserID(ctx)
```

## CORS

```go
// Allow all origins
gofastapi.Use(middleware.CORS("*"))

// Allow specific origins
gofastapi.Use(middleware.CORS(
    "https://app.example.com",
    "https://admin.example.com",
))

// Full config
gofastapi.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins:     []string{"https://app.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           3600,
}))
```

## Rate Limiting

```go
// 1000 requests per minute per IP
gofastapi.Use(middleware.RateLimit(1000))

// Custom key function (e.g. per user ID)
gofastapi.Use(middleware.RateLimit(500, middleware.RateLimitConfig{
    Burst:   100,
    KeyFunc: func(ctx middleware.Context) string {
        claims := middleware.GetClaims(ctx)
        if claims != nil {
            return fmt.Sprintf("user:%d", claims.UserID)
        }
        return ctx.ClientIP()
    },
}))
```

## Logger

```go
// Default text format
gofastapi.Use(middleware.Logger())

// JSON format, skip health checks
gofastapi.Use(middleware.Logger(middleware.LoggerConfig{
    SkipPaths: []string{"/health", "/metrics"},
    Format:    "json",
}))
```

## Recovery (Panic Handler)

```go
// Default: print stack trace, respond 500
gofastapi.Use(middleware.Recovery())

// Custom: report to Sentry, suppress stack trace in logs
gofastapi.Use(middleware.Recovery(middleware.RecoveryConfig{
    PrintStack: false,
    OnPanic: func(ctx middleware.Context, err any, stack []byte) {
        sentry.CaptureException(fmt.Errorf("%v", err))
    },
}))
```

## Response Cache

```go
// Cache GET responses for 5 minutes
gofastapi.Use(middleware.Cache(5 * time.Minute))

// Authenticated requests are NOT cached by default
// Add to specific handlers by using per-route middleware
```

## Writing Custom Middleware

```go
func MyMiddleware() middleware.MiddlewareFunc {
    return func(ctx middleware.Context) {
        // Before: inspect/modify request
        start := time.Now()
        ctx.Set("start_time", start)

        // Call next handler
        ctx.Next()

        // After: inspect/modify response (post-handler)
        elapsed := time.Since(start)
        log.Printf("Request took %s", elapsed)
    }
}

// Register globally
gofastapi.Use(MyMiddleware())

// Register on specific routes
gofastapi.CRUD[Order]("/orders", db,
    crud.WithMiddleware[Order](MyMiddleware()),
)
```
