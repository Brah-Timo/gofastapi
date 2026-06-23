// Example: Full CRUD with PostgreSQL.
//
// Prerequisites:
//
//	docker run -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=myapp \
//	           -p 5432:5432 postgres:16-alpine
//
// Run:
//
//	DATABASE_URL=postgres://postgres:pass@localhost/myapp \
//	go run ./examples/with-postgres
package main

import (
	"log"
	"os"

	"github.com/Brah-Timo/gofastapi"
	"github.com/Brah-Timo/gofastapi/crud"
	"github.com/Brah-Timo/gofastapi/middleware"
	"gorm.io/gorm"
)

// Product is the model.
type Product struct {
	gorm.Model          // adds ID, CreatedAt, UpdatedAt, DeletedAt
	Name        string  `json:"name"        validate:"required,min=2,max=255"`
	Description string  `json:"description"`
	Price       float64 `json:"price"       validate:"required,gt=0"`
	Stock       int     `json:"stock"       validate:"min=0"`
	SKU         string  `json:"sku"         gorm:"uniqueIndex" validate:"required"`
	CategoryID  uint    `json:"category_id"`
}

// Category groups products.
type Category struct {
	gorm.Model
	Name        string `json:"name"        validate:"required,min=2"`
	Description string `json:"description"`
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to PostgreSQL with custom pool settings.
	database := gofastapi.ConnectDB(dsn)

	// Auto-migrate tables.
	gofastapi.MustAutoMigrate[Category](database)
	gofastapi.MustAutoMigrate[Product](database)

	// Global middleware stack.
	gofastapi.Use(
		middleware.Recovery(),
		middleware.Logger(),
		middleware.CORS("*"),
		middleware.RateLimit(2000), // 2000 req/min per IP
	)

	// Categories — fully open.
	gofastapi.CRUD[Category]("/api/categories", database,
		crud.WithPageSize[Category](50),
		crud.WithSearchFields[Category]("name"),
	)

	// Products — soft delete + rich search.
	gofastapi.CRUD[Product]("/api/products", database,
		crud.WithPageSize[Product](24),
		crud.WithSoftDelete[Product](),
		crud.WithSearchFields[Product]("name", "description", "sku"),
		crud.WithOrderFields[Product]("name", "price", "created_at", "stock"),
		crud.WithSelectFields[Product]("id", "name", "description", "price", "stock", "sku", "category_id", "created_at"),
		crud.WithBeforeCreate[Product](func(p *Product, ctx crud.Context) error {
			if p.Price > 1_000_000 {
				return ErrPriceTooHigh
			}
			return nil
		}),
	)

	// Health check endpoint.
	gofastapi.Group("/health").GET("", func(ctx middleware.Context) {
		ctx.JSON(200, map[string]any{
			"status":   "ok",
			"database": "connected",
		})
	})

	// Swagger UI.
	gofastapi.EnableSwagger("Product Catalogue API", "1.0.0",
		"A simple product catalogue API powered by gofastapi and PostgreSQL.")

	log.Println("🚀 Product API on :8080")
	gofastapi.Run(":8080")
}

var ErrPriceTooHigh = errorString("price cannot exceed 1,000,000")

type errorString string

func (e errorString) Error() string { return string(e) }
