// Example: Enterprise-grade application.
//
// Demonstrates the full power of gofastapi:
//   - Multiple CRUD resources with different access policies
//   - JWT authentication with role-based access control
//   - Rate limiting per IP
//   - Soft delete
//   - Hooks (auto-populate, notifications, audit log)
//   - Custom middleware (tenant isolation)
//   - Health check endpoint
//   - Swagger UI
//   - Graceful shutdown
//
// Run:
//
//	DATABASE_URL=sqlite://./enterprise.db \
//	JWT_SECRET=super-secret \
//	go run ./examples/enterprise
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/Brah-Timo/gofastapi"
	"github.com/Brah-Timo/gofastapi/config"
	"github.com/Brah-Timo/gofastapi/crud"
	"github.com/Brah-Timo/gofastapi/middleware"
)

// ─────────────────────────────────────────────────────────────────────────────
// Domain models
// ─────────────────────────────────────────────────────────────────────────────

type Tenant struct {
	gorm.Model
	Name   string `json:"name"   validate:"required,min=2"`
	Slug   string `json:"slug"   validate:"required,slug" gorm:"uniqueIndex"`
	Plan   string `json:"plan"   validate:"oneof=free pro enterprise" gorm:"default:free"`
	Active bool   `json:"active" gorm:"default:true"`
}

type User struct {
	gorm.Model
	TenantID  uint   `json:"tenant_id"  validate:"required"`
	Email     string `json:"email"      validate:"required,email" gorm:"uniqueIndex"`
	FirstName string `json:"first_name" validate:"required,min=2"`
	LastName  string `json:"last_name"  validate:"required,min=2"`
	Role      string `json:"role"       validate:"oneof=admin manager member" gorm:"default:member"`
	Active    bool   `json:"active"     gorm:"default:true"`
}

type Product struct {
	gorm.Model
	TenantID    uint    `json:"tenant_id"    validate:"required"`
	Name        string  `json:"name"         validate:"required,min=2,max=255"`
	Description string  `json:"description"`
	Price       float64 `json:"price"        validate:"required,gt=0"`
	Stock       int     `json:"stock"        validate:"min=0"`
	SKU         string  `json:"sku"          gorm:"uniqueIndex"`
	Active      bool    `json:"active"       gorm:"default:true"`
}

type Order struct {
	gorm.Model
	TenantID  uint      `json:"tenant_id"   validate:"required"`
	UserID    uint      `json:"user_id"`
	ProductID uint      `json:"product_id"  validate:"required"`
	Quantity  int       `json:"quantity"    validate:"required,min=1,max=1000"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"      gorm:"default:pending"`
	PlacedAt  time.Time `json:"placed_at"`
}

type AuditLog struct {
	ID         uint      `json:"id"          gorm:"primaryKey"`
	TenantID   uint      `json:"tenant_id"`
	UserID     uint      `json:"user_id"`
	Resource   string    `json:"resource"`
	Action     string    `json:"action"`
	ResourceID uint      `json:"resource_id"`
	ClientIP   string    `json:"client_ip"`
	CreatedAt  time.Time `json:"created_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// main
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// Load config from environment
	cfg := config.LoadFromEnv()
	log.Printf("🏢 Starting %s in %s mode", cfg.AppName, cfg.AppEnv)

	// Use config DSN if set, else fall back to SQLite
	dsn := cfg.Database.DSN
	if dsn == "" {
		dsn = "sqlite://./enterprise.db"
		log.Println("⚠️  DATABASE_URL not set — using SQLite")
	}

	db := gofastapi.ConnectDB(dsn)

	// Migrate all models
	gofastapi.MustAutoMigrate[Tenant](db)
	gofastapi.MustAutoMigrate[User](db)
	gofastapi.MustAutoMigrate[Product](db)
	gofastapi.MustAutoMigrate[Order](db)
	gofastapi.MustAutoMigrate[AuditLog](db)

	// ── Global middleware ──────────────────────────────────────────────────
	gofastapi.Use(
		middleware.Recovery(middleware.RecoveryConfig{PrintStack: cfg.Debug}),
		middleware.Logger(middleware.LoggerConfig{
			SkipPaths: []string{"/health", "/metrics"},
			Format:    cfg.Log.Format,
		}),
		middleware.CORS("*"),
		middleware.RateLimit(cfg.RateLimit.RequestsPerMinute),
	)

	jwtMW := middleware.JWT(middleware.JWTConfig{Secret: cfg.JWT.Secret})
	adminJwtMW := middleware.JWT(middleware.JWTConfig{
		Secret:       cfg.JWT.Secret,
		AllowedRoles: []string{"admin"},
	})

	// ── Admin-only CRUD ───────────────────────────────────────────────────
	// Tenants can only be managed by admins.
	gofastapi.CRUD[Tenant]("/api/admin/tenants", db,
		crud.WithAuth[Tenant](adminJwtMW),
		crud.WithPageSize[Tenant](50),
		crud.WithSearchFields[Tenant]("name", "slug"),
		crud.WithSoftDelete[Tenant](),
	)

	// Users — admins manage all, members can only see their own.
	gofastapi.CRUD[User]("/api/v1/users", db,
		crud.WithAuth[User](jwtMW),
		crud.WithPageSize[User](25),
		crud.WithSearchFields[User]("email", "first_name", "last_name"),
		crud.WithSoftDelete[User](),
		crud.WithSelectFields[User](
			"id", "tenant_id", "email", "first_name", "last_name",
			"role", "active", "created_at",
		),
		crud.WithBeforeCreate[User](func(u *User, ctx crud.Context) error {
			claims := middleware.GetClaims(ctx)
			if claims != nil {
				u.TenantID = 1 // derive from claims in real app
			}
			return nil
		}),
	)

	// Products — all authenticated users can CRUD.
	gofastapi.CRUD[Product]("/api/v1/products", db,
		crud.WithAuth[Product](jwtMW),
		crud.WithPageSize[Product](24),
		crud.WithSoftDelete[Product](),
		crud.WithSearchFields[Product]("name", "description", "sku"),
		crud.WithOrderFields[Product]("name", "price", "created_at", "stock"),
		crud.WithBeforeCreate[Product](func(p *Product, ctx crud.Context) error {
			claims := middleware.GetClaims(ctx)
			if claims != nil {
				p.TenantID = 1
			}
			if p.Price > 9_999_999 {
				return errors.New("price exceeds maximum allowed value")
			}
			return nil
		}),
		crud.WithAfterCreate[Product](func(p *Product, ctx crud.Context) error {
			log.Printf("📦 New product: %q (id=%d, price=%.2f)", p.Name, p.ID, p.Price)
			return nil
		}),
	)

	// Orders — authenticated, with business logic in hooks.
	gofastapi.CRUD[Order]("/api/v1/orders", db,
		crud.WithAuth[Order](jwtMW),
		crud.WithPageSize[Order](20),
		crud.WithSoftDelete[Order](),
		crud.WithBeforeCreate[Order](func(o *Order, ctx crud.Context) error {
			claims := middleware.GetClaims(ctx)
			if claims != nil {
				o.UserID = claims.UserID
				o.TenantID = 1
			}
			o.PlacedAt = time.Now().UTC()
			return nil
		}),
		crud.WithAfterCreate[Order](func(o *Order, ctx crud.Context) error {
			go writeAuditLog(db, o.TenantID, o.UserID, "order", "create", o.ID, ctx.ClientIP())
			return nil
		}),
		crud.WithBeforeDelete[Order](func(o *Order, ctx crud.Context) error {
			if o.Status == "shipped" || o.Status == "delivered" {
				return fmt.Errorf("cannot delete order with status %q", o.Status)
			}
			return nil
		}),
	)

	// ── Auth endpoints ─────────────────────────────────────────────────────
	authGroup := gofastapi.Group("/auth")
	authGroup.POST("/login", handleLogin(cfg.JWT.Secret))
	authGroup.POST("/refresh", handleRefresh(cfg.JWT.Secret))

	// ── Health & metrics ───────────────────────────────────────────────────
	gofastapi.Group("/health").GET("", func(ctx middleware.Context) {
		ctx.JSON(http.StatusOK, map[string]any{
			"status":  "ok",
			"app":     cfg.AppName,
			"version": cfg.AppVersion,
			"env":     cfg.AppEnv,
		})
	})

	// ── Swagger UI ─────────────────────────────────────────────────────────
	gofastapi.EnableSwagger(
		"Enterprise API",
		cfg.AppVersion,
		"Full-featured multi-tenant REST API powered by gofastapi. "+
			"Visit /docs for interactive documentation.",
	)

	log.Printf("🚀 Server starting on %s", cfg.Server.Port)
	log.Println("   GET  /health           → health check")
	log.Println("   GET  /docs             → Swagger UI")
	log.Println("   POST /auth/login        → get JWT token")

	if err := gofastapi.Run(cfg.Server.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleLogin(jwtSecret string) func(ctx middleware.Context) {
	return func(ctx middleware.Context) {
		var req loginRequest
		if err := ctx.BindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		// Demo: accept any password for alice@example.com
		if req.Email == "" || req.Password == "" {
			ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		token, err := middleware.GenerateToken(1, req.Email, "admin", jwtSecret, 24*time.Hour)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "token error"})
			return
		}
		ctx.JSON(http.StatusOK, map[string]any{
			"token":      token,
			"expires_in": 86400,
			"type":       "Bearer",
		})
	}
}

func handleRefresh(jwtSecret string) func(ctx middleware.Context) {
	return func(ctx middleware.Context) {
		claims := middleware.GetClaims(ctx)
		if claims == nil {
			ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
			return
		}
		token, err := middleware.GenerateToken(
			claims.UserID, claims.UserEmail, claims.Role, jwtSecret, 24*time.Hour)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "token error"})
			return
		}
		ctx.JSON(http.StatusOK, map[string]any{
			"token":      token,
			"expires_in": 86400,
		})
	}
}

func writeAuditLog(db gofastapi.Database, tenantID, userID uint, resource, action string, resourceID uint, ip string) {
	entry := AuditLog{
		TenantID:   tenantID,
		UserID:     userID,
		Resource:   resource,
		Action:     action,
		ResourceID: resourceID,
		ClientIP:   ip,
		CreatedAt:  time.Now().UTC(),
	}
	db.DB().Create(&entry)
}
