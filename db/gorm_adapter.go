// Package db — GORM adapter helpers.
// This file provides additional GORM-specific helpers that make it easier
// to work with the GORMDatabase in application code.
package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ─────────────────────────────────────────────────────────────────────────────
// Scoping helpers — build reusable query fragments
// ─────────────────────────────────────────────────────────────────────────────

// Scope is an alias for GORM's func(*gorm.DB) *gorm.DB.
// Use it to build composable query modifiers:
//
//	active := db.Scope(func(q *gorm.DB) *gorm.DB {
//	    return q.Where("deleted_at IS NULL")
//	})
type Scope = func(*gorm.DB) *gorm.DB

// WhereScope returns a Scope that adds a WHERE condition.
func WhereScope(query any, args ...any) Scope {
	return func(q *gorm.DB) *gorm.DB {
		return q.Where(query, args...)
	}
}

// OrderScope returns a Scope that adds an ORDER BY clause.
func OrderScope(column, direction string) Scope {
	// Whitelist direction to prevent SQL injection.
	if direction != "asc" && direction != "desc" {
		direction = "asc"
	}
	return func(q *gorm.DB) *gorm.DB {
		return q.Order(fmt.Sprintf("%s %s", column, direction))
	}
}

// PaginateScope returns a Scope that applies LIMIT / OFFSET pagination.
func PaginateScope(page, pageSize int) Scope {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	return func(q *gorm.DB) *gorm.DB {
		return q.Offset(offset).Limit(pageSize)
	}
}

// PreloadScope returns a Scope that preloads an association.
func PreloadScope(association string, conditions ...any) Scope {
	return func(q *gorm.DB) *gorm.DB {
		if len(conditions) > 0 {
			return q.Preload(association, conditions[0])
		}
		return q.Preload(association)
	}
}

// SelectScope returns a Scope that limits returned columns.
func SelectScope(columns ...string) Scope {
	return func(q *gorm.DB) *gorm.DB {
		if len(columns) == 0 {
			return q
		}
		return q.Select(columns)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Transaction helpers
// ─────────────────────────────────────────────────────────────────────────────

// WithTransaction executes fn inside a database transaction.
// If fn returns an error the transaction is rolled back; otherwise it is
// committed. The *gorm.DB passed to fn is the transaction object.
//
//	err := db.WithTransaction(database, func(tx *gorm.DB) error {
//	    tx.Create(&user)
//	    tx.Create(&profile)
//	    return nil
//	})
func WithTransaction(database Database, fn func(tx *gorm.DB) error) error {
	return database.DB().Transaction(fn)
}

// WithTransactionCtx is like WithTransaction but propagates a context
// for cancellation and deadline support.
func WithTransactionCtx(ctx context.Context, database Database, fn func(tx *gorm.DB) error) error {
	return database.DB().WithContext(ctx).Transaction(fn)
}

// ─────────────────────────────────────────────────────────────────────────────
// Upsert helper
// ─────────────────────────────────────────────────────────────────────────────

// Upsert inserts model if the primary key doesn't exist, or updates it
// if it does. Uses GORM's ON CONFLICT DO UPDATE semantics.
//
//	err := db.Upsert(database, &user, "email")  // conflict on email column
func Upsert[T any](database Database, model *T, conflictColumns ...string) error {
	cols := make([]clause.Column, len(conflictColumns))
	for i, c := range conflictColumns {
		cols[i] = clause.Column{Name: c}
	}
	return database.DB().
		Clauses(clause.OnConflict{
			Columns:   cols,
			DoUpdates: clause.AssignmentColumns(conflictColumns),
		}).
		Create(model).Error
}

// ─────────────────────────────────────────────────────────────────────────────
// Bulk operations
// ─────────────────────────────────────────────────────────────────────────────

// BulkCreate inserts a slice of models in a single database round-trip.
// batchSize controls how many rows are inserted per batch (0 → all at once).
func BulkCreate[T any](database Database, items []T, batchSize int) error {
	if batchSize <= 0 {
		return database.DB().Create(&items).Error
	}
	return database.DB().CreateInBatches(&items, batchSize).Error
}

// ─────────────────────────────────────────────────────────────────────────────
// Soft-delete helpers
// ─────────────────────────────────────────────────────────────────────────────

// WithTrashed returns a *gorm.DB that includes soft-deleted records.
// Use it when you need to see all records regardless of deletion status.
func WithTrashed(database Database) *gorm.DB {
	return database.DB().Unscoped()
}

// Restore un-deletes a soft-deleted record by setting its deleted_at to NULL.
// T must embed gorm.DeletedAt (or have a DeletedAt field with the right type).
func Restore[T any](database Database, id any) error {
	return database.DB().Model(new(T)).
		Unscoped().
		Where("id = ?", id).
		Update("deleted_at", nil).Error
}

// ─────────────────────────────────────────────────────────────────────────────
// Count helper
// ─────────────────────────────────────────────────────────────────────────────

// Count returns the total number of records for model T matching the optional
// scopes.
func Count[T any](database Database, scopes ...Scope) (int64, error) {
	var count int64
	q := database.DB().Model(new(T))
	for _, s := range scopes {
		q = s(q)
	}
	err := q.Count(&count).Error
	return count, err
}

// ─────────────────────────────────────────────────────────────────────────────
// Raw query helpers
// ─────────────────────────────────────────────────────────────────────────────

// RawQuery executes a raw SQL query and scans the results into dest.
// dest should be a pointer to a slice of structs or a map.
//
//	var results []UserSummary
//	err := db.RawQuery(database, &results,
//	    "SELECT id, name FROM users WHERE active = ?", true)
func RawQuery(database Database, dest any, sql string, args ...any) error {
	return database.DB().Raw(sql, args...).Scan(dest).Error
}

// RawExec executes a raw SQL statement (INSERT / UPDATE / DELETE / DDL).
func RawExec(database Database, sql string, args ...any) error {
	return database.DB().Exec(sql, args...).Error
}
