// Example: CRUD protected by JWT Authentication.
//
// Run:
//
//	JWT_SECRET=my-secret go run ./examples/with-auth
//
// 1. Get a token (via /auth/login):
//
//	curl -X POST http://localhost:8080/auth/login \
//	     -H "Content-Type: application/json" \
//	     -d '{"email":"admin@example.com","password":"admin123"}'
//
// 2. Use the token:
//
//	curl http://localhost:8080/api/v1/orders \
//	     -H "Authorization: Bearer <token>"
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Brah-Timo/gofastapi"
	"github.com/Brah-Timo/gofastapi/crud"
	"github.com/Brah-Timo/gofastapi/middleware"
)

// Order is the protected resource.
type Order struct {
	ID        uint    `json:"id"         gorm:"primaryKey"`
	UserID    uint    `json:"user_id"    gorm:"not null"`
	ProductID uint    `json:"product_id" validate:"required"`
	Quantity  int     `json:"quantity"   validate:"required,min=1"`
	Total     float64 `json:"total"`
	Status    string  `json:"status"     gorm:"default:pending"`
}

// User is the authentication model.
type User struct {
	ID       uint   `json:"id"       gorm:"primaryKey"`
	Email    string `json:"email"    validate:"required,email" gorm:"uniqueIndex"`
	Password string `json:"password" validate:"required,min=8"`
	Role     string `json:"role"     gorm:"default:member"`
}

// LoginRequest is the payload for /auth/login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func main() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
		log.Println("⚠️  JWT_SECRET not set — using dev secret")
	}

	// Setup
	db := gofastapi.ConnectDB("sqlite://./auth-example.db")
	gofastapi.MustAutoMigrate[User](db)
	gofastapi.MustAutoMigrate[Order](db)

	// Global middleware: logger + recovery
	gofastapi.Use(
		middleware.Recovery(),
		middleware.Logger(),
		middleware.CORS("*"),
	)

	// Public routes
	authGroup := gofastapi.Group("/auth")
	authGroup.POST("/login", func(ctx middleware.Context) {
		var req LoginRequest
		if err := ctx.BindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		// In production: look up user, verify bcrypt hash.
		// Here we use a hardcoded demo user.
		if req.Email != "admin@example.com" || req.Password != "admin123" {
			ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		token, err := middleware.GenerateToken(1, req.Email, "admin", jwtSecret, 0)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
			return
		}
		ctx.JSON(http.StatusOK, map[string]string{
			"token":   token,
			"type":    "Bearer",
			"message": "login successful",
		})
	})

	// Protected CRUD — requires valid JWT
	jwtMW := middleware.JWT(middleware.JWTConfig{Secret: jwtSecret})
	gofastapi.CRUD[Order]("/api/v1/orders", db,
		crud.WithAuth[Order](jwtMW),
		crud.WithPageSize[Order](20),
		crud.WithBeforeCreate[Order](func(o *Order, ctx crud.Context) error {
			// Automatically assign the authenticated user's ID.
			claims := middleware.GetClaims(ctx)
			if claims != nil {
				o.UserID = claims.UserID
			}
			return nil
		}),
	)

	log.Println("🚀 Server running on :8080")
	log.Println("   POST /auth/login          → get token")
	log.Println("   GET  /api/v1/orders        → list orders (auth required)")
	gofastapi.Run(":8080")
}
