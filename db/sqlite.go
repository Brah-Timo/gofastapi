// Package db — SQLite driver helpers.
// SQLite is perfect for local development and testing.
// It requires CGO (the mattn/go-sqlite3 driver uses CGO by default).
// For a pure-Go alternative swap to modernc.org/sqlite.
package db

import (
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// connectSQLite opens a GORM connection to SQLite.
// DSN variants accepted:
//
//	":memory:"            → in-memory database (lost on close)
//	"sqlite://./app.db"   → file-based, strips the scheme prefix
//	"./app.db"            → file-based, relative path
//	"file:app.db?cache=shared" → SQLite URI format
func connectSQLite(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
	// Strip "sqlite://" prefix if present.
	path := strings.TrimPrefix(dsn, "sqlite://")
	return gorm.Open(sqlite.Open(path), cfg)
}

// ConnectSQLite opens a connection to a SQLite database.
//
//	db, err := db.ConnectSQLite(":memory:")    // test / demo
//	db, err := db.ConnectSQLite("./myapp.db")  // development
func ConnectSQLite(path string, opts ...Option) (Database, error) {
	if path == "" {
		path = ":memory:"
	}
	return AutoConnect(path, opts...)
}

// InMemory opens an in-memory SQLite database. Convenient for tests.
//
//	db := db.MustInMemory()
//	defer db.Close()
func InMemory(opts ...Option) (Database, error) {
	return ConnectSQLite(":memory:", opts...)
}

// MustInMemory is like InMemory but panics on error.
func MustInMemory(opts ...Option) Database {
	d, err := InMemory(opts...)
	if err != nil {
		panic("db.MustInMemory: " + err.Error())
	}
	return d
}
