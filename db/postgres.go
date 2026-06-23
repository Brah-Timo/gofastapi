// Package db — PostgreSQL driver helpers.
package db

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// connectPostgres opens a GORM connection to PostgreSQL using the pgx driver.
// The DSN can be a URL ("postgres://…") or a key=value connection string.
func connectPostgres(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
	// Normalise postgres:// → postgresql:// because pgx accepts both but
	// some tools emit one or the other.
	normalised := strings.Replace(dsn, "postgres://", "postgresql://", 1)
	return gorm.Open(postgres.New(postgres.Config{
		DSN:                  normalised,
		PreferSimpleProtocol: false, // use extended protocol for full type safety
	}), cfg)
}

// PostgresDSN builds a PostgreSQL connection string from discrete components.
// Any empty field is omitted so that its default is used.
//
//	dsn := PostgresDSN("localhost", 5432, "mydb", "user", "pass", "disable")
//	// → "host=localhost port=5432 dbname=mydb user=user password=pass sslmode=disable"
func PostgresDSN(host string, port int, dbname, user, password, sslmode string) string {
	var parts []string
	if host != "" {
		parts = append(parts, "host="+host)
	}
	if port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", port))
	}
	if dbname != "" {
		parts = append(parts, "dbname="+dbname)
	}
	if user != "" {
		parts = append(parts, "user="+user)
	}
	if password != "" {
		parts = append(parts, "password="+password)
	}
	if sslmode != "" {
		parts = append(parts, "sslmode="+sslmode)
	} else {
		parts = append(parts, "sslmode=disable")
	}
	return strings.Join(parts, " ")
}

// PostgresURL builds a postgres:// URL from discrete components.
//
//	url := PostgresURL("user", "pass", "localhost", 5432, "mydb")
//	// → "postgres://user:pass@localhost:5432/mydb"
func PostgresURL(user, password, host string, port int, dbname string) string {
	auth := user
	if password != "" {
		auth = user + ":" + password
	}
	return fmt.Sprintf("postgres://%s@%s:%d/%s", auth, host, port, dbname)
}

// ConnectPostgres opens a connection to PostgreSQL using the provided DSN.
// This is a convenience wrapper around AutoConnect for users who already
// know they are using PostgreSQL.
func ConnectPostgres(dsn string, opts ...Option) (Database, error) {
	return AutoConnect(dsn, opts...)
}
