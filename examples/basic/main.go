// Example: Basic CRUD in 5 lines.
//
// Run:
//
//	go run ./examples/basic
//
// Then try:
//
//	curl http://localhost:8080/users
//	curl -X POST http://localhost:8080/users \
//	     -H "Content-Type: application/json" \
//	     -d '{"name":"Alice","email":"alice@example.com","age":30}'
//	curl http://localhost:8080/users/1
//	curl -X PUT http://localhost:8080/users/1 \
//	     -H "Content-Type: application/json" \
//	     -d '{"name":"Alice Updated","email":"alice@example.com","age":31}'
//	curl -X DELETE http://localhost:8080/users/1
package main

import (
	"github.com/Brah-Timo/gofastapi"
)

// User is the data model — this is all you need to define.
type User struct {
	ID    uint   `json:"id"    gorm:"primaryKey"`
	Name  string `json:"name"  validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email" gorm:"uniqueIndex"`
	Age   int    `json:"age"   validate:"min=0,max=150"`
}

func main() {
	// 1. Connect to SQLite (zero config for demos / CI)
	db := gofastapi.ConnectDB("sqlite://./app.db")

	// 2. Auto-create / migrate the users table
	gofastapi.MustAutoMigrate[User](db)

	// 3. Register 5 REST endpoints in one line
	gofastapi.CRUD[User]("/users", db)

	// 4. Start the server
	gofastapi.Run(":8080")

	// That's it. You now have:
	//   GET    /users         → paginated list
	//   GET    /users/:id     → single user
	//   POST   /users         → create with validation
	//   PUT    /users/:id     → update with validation
	//   DELETE /users/:id     → delete
}
