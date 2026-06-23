// Package db — sqlx adapter.
//
// This file provides optional sqlx-based helpers for users who prefer
// sqlx's named queries and struct scanning over GORM's ORM style.
// The sqlx.DB is created from the same underlying *sql.DB as GORM so
// both can coexist on the same connection pool.
package db

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// SQLxDB wraps sqlx.DB on top of a GORMDatabase, sharing the same
// connection pool.
type SQLxDB struct {
	*sqlx.DB
}

// ToSQLx converts an existing Database into a SQLxDB that shares the same
// underlying connection pool.
//
//	gdb := db.MustConnect("postgres://…")
//	xdb := db.ToSQLx(gdb, "postgres")
func ToSQLx(database Database, driverName string) (*SQLxDB, error) {
	xdb := sqlx.NewDb(database.SQL(), driverName)
	return &SQLxDB{xdb}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Named query helpers
// ─────────────────────────────────────────────────────────────────────────────

// NamedQuery executes a named SQL query (`:param` style) and scans results
// into dest (must be a pointer to a slice of structs or a single struct).
//
//	var users []User
//	err := xdb.NamedQuery(ctx, &users,
//	    "SELECT * FROM users WHERE status = :status", map[string]any{"status": "active"})
func (x *SQLxDB) NamedQuery(ctx context.Context, dest any, query string, arg any) error {
	rows, err := x.DB.NamedQueryContext(ctx, query, arg)
	if err != nil {
		return fmt.Errorf("sqlx.NamedQuery: %w", err)
	}
	defer rows.Close()
	return sqlx.StructScan(rows, dest)
}

// Get executes a query and scans a single row into dest.
// Returns sql.ErrNoRows if no row is found.
func (x *SQLxDB) Get(ctx context.Context, dest any, query string, args ...any) error {
	return x.DB.GetContext(ctx, dest, query, args...)
}

// Select executes a query and scans all rows into dest (pointer to slice).
func (x *SQLxDB) Select(ctx context.Context, dest any, query string, args ...any) error {
	return x.DB.SelectContext(ctx, dest, query, args...)
}
