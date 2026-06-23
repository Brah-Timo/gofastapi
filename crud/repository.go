// Package crud — generic repository layer.
//
// The Repository interface abstracts all database access so that:
//   - Handlers depend only on the interface (easily mockable in tests).
//   - The default GORM implementation can be swapped for any custom backend.
package crud

import (
	"context"
	"fmt"
	"strings"

	"github.com/Brah-Timo/gofastapi/db"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// ListParams — parameters for the List operation
// ─────────────────────────────────────────────────────────────────────────────

// ListParams carries all query modifiers for the List endpoint.
type ListParams struct {
	// Page is the 1-based page number. Defaults to 1.
	Page int
	// PageSize is the number of items per page. Defaults to 20.
	PageSize int
	// Search is a free-text search term applied to configured search fields.
	Search string
	// OrderBy is the column name to order by.
	OrderBy string
	// OrderDir is "asc" or "desc". Defaults to "asc".
	OrderDir string
	// Filters holds additional equality filters: column → value.
	Filters map[string]string
	// Preloads holds association names to eagerly load.
	Preloads []string
	// SelectFields limits the columns returned. Empty = all columns.
	SelectFields []string
}

// ─────────────────────────────────────────────────────────────────────────────
// Repository interface
// ─────────────────────────────────────────────────────────────────────────────

// Repository defines the minimal set of database operations the Handler uses.
// Implement this interface to use a custom data source (e.g. an external API,
// an in-memory store, or a non-GORM database driver).
type Repository[T any] interface {
	// List returns a page of items matching the params plus the total count.
	List(ctx context.Context, params ListParams) (items []T, total int64, err error)
	// FindByID returns the item with the given primary key.
	// Returns ErrNotFound if no such record exists.
	FindByID(ctx context.Context, id any) (T, error)
	// Create inserts a new item and populates its auto-generated fields (ID, …).
	Create(ctx context.Context, item *T) error
	// Update saves all fields of item.
	Update(ctx context.Context, item *T) error
	// Delete permanently removes the item with id.
	Delete(ctx context.Context, id any) error
	// SoftDelete sets deleted_at on the item with id.
	// The struct must have a gorm.DeletedAt field.
	SoftDelete(ctx context.Context, id any) error
	// Count returns the total number of items matching params.
	Count(ctx context.Context, params ListParams) (int64, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// GORMRepository — default GORM implementation
// ─────────────────────────────────────────────────────────────────────────────

// GORMRepository[T] is the default Repository implementation backed by GORM.
// It is created automatically by NewHandler unless overridden with WithRepository.
type GORMRepository[T any] struct {
	gdb          *gorm.DB
	searchFields []string
	allowedOrder []string
}

// NewGORMRepository creates a new GORMRepository for type T.
func NewGORMRepository[T any](database db.Database) *GORMRepository[T] {
	return &GORMRepository[T]{
		gdb: database.DB(),
	}
}

// NewRepository is a convenience constructor that returns the Repository
// interface (same as NewGORMRepository but typed as the interface).
func NewRepository[T any](database db.Database) Repository[T] {
	return NewGORMRepository[T](database)
}

// ─────────────────────────────────────────────────────────────────────────────
// List
// ─────────────────────────────────────────────────────────────────────────────

func (r *GORMRepository[T]) List(ctx context.Context, p ListParams) ([]T, int64, error) {
	var items []T
	var total int64

	q := r.gdb.WithContext(ctx).Model(new(T))

	// Selective columns.
	if len(p.SelectFields) > 0 {
		q = q.Select(p.SelectFields)
	}

	// Search across configured text fields.
	if p.Search != "" && len(r.searchFields) > 0 {
		conditions := make([]string, len(r.searchFields))
		args := make([]any, len(r.searchFields))
		for i, f := range r.searchFields {
			conditions[i] = fmt.Sprintf("LOWER(%s) LIKE ?", f)
			args[i] = "%" + strings.ToLower(p.Search) + "%"
		}
		q = q.Where(strings.Join(conditions, " OR "), args...)
	}

	// Extra equality filters.
	for col, val := range p.Filters {
		q = q.Where(col+" = ?", val)
	}

	// Total count (before pagination).
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("repository.List count: %w", err)
	}

	// Ordering — whitelist check to prevent injection.
	if p.OrderBy != "" {
		if r.isAllowedOrder(p.OrderBy) {
			dir := "ASC"
			if strings.ToLower(p.OrderDir) == "desc" {
				dir = "DESC"
			}
			q = q.Order(p.OrderBy + " " + dir)
		}
	} else {
		q = q.Order("id ASC")
	}

	// Pagination.
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.Page < 1 {
		p.Page = 1
	}
	offset := (p.Page - 1) * p.PageSize
	q = q.Offset(offset).Limit(p.PageSize)

	// Eager load associations.
	for _, assoc := range p.Preloads {
		q = q.Preload(assoc)
	}

	if err := q.Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("repository.List find: %w", err)
	}

	return items, total, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FindByID
// ─────────────────────────────────────────────────────────────────────────────

func (r *GORMRepository[T]) FindByID(ctx context.Context, id any) (T, error) {
	var item T
	err := r.gdb.WithContext(ctx).First(&item, "id = ?", id).Error
	if err != nil {
		if IsNotFound(err) {
			return item, ErrNotFound
		}
		return item, fmt.Errorf("repository.FindByID: %w", err)
	}
	return item, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────────────────────────────────────

func (r *GORMRepository[T]) Create(ctx context.Context, item *T) error {
	if err := r.gdb.WithContext(ctx).Create(item).Error; err != nil {
		return fmt.Errorf("repository.Create: %w", ClassifyDBError(err))
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────────────────────

func (r *GORMRepository[T]) Update(ctx context.Context, item *T) error {
	if err := r.gdb.WithContext(ctx).Save(item).Error; err != nil {
		return fmt.Errorf("repository.Update: %w", ClassifyDBError(err))
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────────────────────

func (r *GORMRepository[T]) Delete(ctx context.Context, id any) error {
	var item T
	if err := r.gdb.WithContext(ctx).Delete(&item, "id = ?", id).Error; err != nil {
		return fmt.Errorf("repository.Delete: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SoftDelete
// ─────────────────────────────────────────────────────────────────────────────

func (r *GORMRepository[T]) SoftDelete(ctx context.Context, id any) error {
	var item T
	// GORM automatically sets deleted_at when the model has gorm.DeletedAt.
	if err := r.gdb.WithContext(ctx).Delete(&item, "id = ?", id).Error; err != nil {
		return fmt.Errorf("repository.SoftDelete: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Count
// ─────────────────────────────────────────────────────────────────────────────

func (r *GORMRepository[T]) Count(ctx context.Context, p ListParams) (int64, error) {
	var count int64
	q := r.gdb.WithContext(ctx).Model(new(T))
	if p.Search != "" && len(r.searchFields) > 0 {
		conditions := make([]string, len(r.searchFields))
		args := make([]any, len(r.searchFields))
		for i, f := range r.searchFields {
			conditions[i] = fmt.Sprintf("LOWER(%s) LIKE ?", f)
			args[i] = "%" + strings.ToLower(p.Search) + "%"
		}
		q = q.Where(strings.Join(conditions, " OR "), args...)
	}
	for col, val := range p.Filters {
		q = q.Where(col+" = ?", val)
	}
	err := q.Count(&count).Error
	return count, err
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

func (r *GORMRepository[T]) isAllowedOrder(column string) bool {
	if len(r.allowedOrder) == 0 {
		// No whitelist → allow everything (users should configure this in prod).
		// Strip dangerous characters as a safety net.
		return !strings.ContainsAny(column, ";'\"()\\")
	}
	for _, c := range r.allowedOrder {
		if strings.EqualFold(c, column) {
			return true
		}
	}
	return false
}
