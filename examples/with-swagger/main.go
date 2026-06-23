// Example: Auto-generated Swagger / OpenAPI 3.0 documentation.
//
// Run:
//
//	go run ./examples/with-swagger
//
// Then open: http://localhost:8080/docs
package main

import (
	"gorm.io/gorm"

	"github.com/Brah-Timo/gofastapi"
	"github.com/Brah-Timo/gofastapi/crud"
	"github.com/Brah-Timo/gofastapi/middleware"
)

type Customer struct {
	gorm.Model
	FirstName string `json:"first_name" validate:"required,min=2"`
	LastName  string `json:"last_name"  validate:"required,min=2"`
	Email     string `json:"email"      validate:"required,email" gorm:"uniqueIndex"`
	Phone     string `json:"phone"      validate:"phone"`
	Company   string `json:"company"`
}

type Invoice struct {
	gorm.Model
	CustomerID uint    `json:"customer_id" validate:"required"`
	Amount     float64 `json:"amount"      validate:"required,gt=0"`
	Currency   string  `json:"currency"    validate:"required,oneof=USD EUR GBP"`
	Status     string  `json:"status"      gorm:"default:draft"`
	Notes      string  `json:"notes"`
}

func main() {
	db := gofastapi.ConnectDB("sqlite://./swagger-example.db")
	gofastapi.MustAutoMigrate[Customer](db)
	gofastapi.MustAutoMigrate[Invoice](db)

	gofastapi.Use(
		middleware.Recovery(),
		middleware.Logger(),
		middleware.CORS("*"),
	)

	// Register CRUD endpoints
	gofastapi.CRUD[Customer]("/api/customers", db,
		crud.WithPageSize[Customer](25),
		crud.WithSearchFields[Customer]("first_name", "last_name", "email", "company"),
	)

	gofastapi.CRUD[Invoice]("/api/invoices", db,
		crud.WithPageSize[Invoice](20),
		crud.WithSoftDelete[Invoice](),
	)

	// Mount Swagger UI at /docs — call after all routes are registered
	gofastapi.EnableSwagger(
		"Billing API",
		"2.0.0",
		"A simple billing API with customers and invoices. "+
			"Built with gofastapi — zero boilerplate REST for Go.",
	)

	// Health endpoint
	gofastapi.Group("/health").GET("", func(ctx middleware.Context) {
		ctx.JSON(200, map[string]any{
			"status":  "healthy",
			"version": "2.0.0",
		})
	})

	gofastapi.Run(":8080")
}
