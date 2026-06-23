// Package crud — Functional Options for Handler[T].
//
// All public WithXxx functions return an Option[T] that can be passed to
// gofastapi.CRUD[T] or crud.NewHandler[T] to configure the handler.
//
// Example:
//
//	gofastapi.CRUD[Product]("/products", db,
//	    crud.WithPageSize[Product](50),
//	    crud.WithSoftDelete[Product](),
//	    crud.WithSearchFields[Product]("name", "description"),
//	    crud.WithBeforeCreate[Product](func(p *Product, ctx crud.Context) error {
//	        p.Slug = slugify(p.Name)
//	        return nil
//	    }),
//	)
package crud

import (
	"github.com/Brah-Timo/gofastapi/hooks"
)

// Option[T] is a function that mutates a Handler[T] configuration.
type Option[T any] func(*Handler[T])

// ─────────────────────────────────────────────────────────────────────────────
// Pagination options
// ─────────────────────────────────────────────────────────────────────────────

// WithPageSize sets the default page size returned by the List endpoint.
// The client can override this with the `page_size` query parameter, subject
// to the MaxPageSize cap.
func WithPageSize[T any](size int) Option[T] {
	return func(h *Handler[T]) {
		h.cfg.DefaultPageSize = size
	}
}

// WithMaxPageSize sets the maximum page size the client is allowed to request.
// Requests exceeding this value are silently capped.
func WithMaxPageSize[T any](size int) Option[T] {
	return func(h *Handler[T]) {
		h.cfg.MaxPageSize = size
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Delete options
// ─────────────────────────────────────────────────────────────────────────────

// WithSoftDelete enables soft-delete semantics: DELETE requests set
// the `deleted_at` column instead of removing the row.
// The model struct must embed gorm.DeletedAt for this to work.
func WithSoftDelete[T any]() Option[T] {
	return func(h *Handler[T]) {
		h.cfg.EnableSoftDelete = true
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Search & ordering options
// ─────────────────────────────────────────────────────────────────────────────

// WithSearchFields configures which database columns the `?search=` query
// parameter is applied to. The search is case-insensitive LIKE matching.
// When not set, free-text search is disabled.
//
//	crud.WithSearchFields[User]("name", "email", "bio")
func WithSearchFields[T any](fields ...string) Option[T] {
	return func(h *Handler[T]) {
		h.cfg.SearchFields = fields
	}
}

// WithOrderFields whitelists the columns that clients may use in `?order_by=`.
// When not set, any column name (without special characters) is permitted.
//
//	crud.WithOrderFields[User]("name", "created_at", "email")
func WithOrderFields[T any](fields ...string) Option[T] {
	return func(h *Handler[T]) {
		h.cfg.AllowedOrderFields = fields
	}
}

// WithSelectFields limits which columns are returned in List and Show.
// Use this to hide sensitive fields (e.g. password hashes) from responses.
//
//	crud.WithSelectFields[User]("id", "name", "email", "created_at")
func WithSelectFields[T any](fields ...string) Option[T] {
	return func(h *Handler[T]) {
		h.cfg.SelectFields = fields
	}
}

// WithPreloads configures GORM associations to eagerly load.
//
//	crud.WithPreloads[Post]("Author", "Tags")
func WithPreloads[T any](associations ...string) Option[T] {
	return func(h *Handler[T]) {
		h.cfg.Preloads = associations
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Repository override
// ─────────────────────────────────────────────────────────────────────────────

// WithRepository replaces the default GORM repository with a custom one.
// Use this for testing (pass a mock) or when you need a non-GORM backend.
//
//	crud.WithRepository[User](myCustomRepo)
func WithRepository[T any](repo Repository[T]) Option[T] {
	return func(h *Handler[T]) {
		h.repo = repo
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Hook options
// ─────────────────────────────────────────────────────────────────────────────

// WithBeforeCreate registers a hook that runs before a resource is created.
// If the hook returns an error the creation is aborted with 400.
//
//	crud.WithBeforeCreate[User](func(u *User, ctx crud.Context) error {
//	    u.Role = "member" // set default role
//	    return nil
//	})
func WithBeforeCreate[T any](fn hooks.HookFunc[T]) Option[T] {
	return func(h *Handler[T]) {
		h.hooksReg.Register(hooks.BeforeCreate, fn)
	}
}

// WithAfterCreate registers a hook that runs after a resource is created.
// Errors returned from AfterCreate hooks are logged but do not change the
// HTTP response (the record has already been persisted).
//
//	crud.WithAfterCreate[User](func(u *User, ctx crud.Context) error {
//	    emailService.SendWelcome(u.Email)
//	    return nil
//	})
func WithAfterCreate[T any](fn hooks.HookFunc[T]) Option[T] {
	return func(h *Handler[T]) {
		h.hooksReg.Register(hooks.AfterCreate, fn)
	}
}

// WithBeforeUpdate registers a hook that runs before a resource is updated.
func WithBeforeUpdate[T any](fn hooks.HookFunc[T]) Option[T] {
	return func(h *Handler[T]) {
		h.hooksReg.Register(hooks.BeforeUpdate, fn)
	}
}

// WithAfterUpdate registers a hook that runs after a resource is updated.
func WithAfterUpdate[T any](fn hooks.HookFunc[T]) Option[T] {
	return func(h *Handler[T]) {
		h.hooksReg.Register(hooks.AfterUpdate, fn)
	}
}

// WithBeforeDelete registers a hook that runs before a resource is deleted.
// Return an error to abort the deletion.
func WithBeforeDelete[T any](fn hooks.HookFunc[T]) Option[T] {
	return func(h *Handler[T]) {
		h.hooksReg.Register(hooks.BeforeDelete, fn)
	}
}

// WithAfterDelete registers a hook that runs after a resource is deleted.
func WithAfterDelete[T any](fn hooks.HookFunc[T]) Option[T] {
	return func(h *Handler[T]) {
		h.hooksReg.Register(hooks.AfterDelete, fn)
	}
}

// WithAfterFind registers a hook that runs after a single record is fetched
// by Show. Use it to transform or enrich the response.
func WithAfterFind[T any](fn hooks.HookFunc[T]) Option[T] {
	return func(h *Handler[T]) {
		h.hooksReg.Register(hooks.AfterFind, fn)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Middleware option
// ─────────────────────────────────────────────────────────────────────────────

// WithMiddleware adds handler-scoped middleware that runs on every request to
// this specific CRUD group only. Applied in the order provided.
//
//	crud.WithMiddleware[Order](middleware.JWT(), myAuthZ)
func WithMiddleware[T any](mw ...MiddlewareFunc) Option[T] {
	return func(h *Handler[T]) {
		h.middlewares = append(h.middlewares, mw...)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth shortcut
// ─────────────────────────────────────────────────────────────────────────────

// WithAuth is a convenience option that protects all five endpoints of this
// CRUD group with JWT authentication middleware.
// It is equivalent to crud.WithMiddleware[T](middleware.JWT()).
//
// Make sure the JWT_SECRET environment variable is set before using this.
//
//	gofastapi.CRUD[Order]("/orders", db, crud.WithAuth[Order]())
func WithAuth[T any](jwtMiddleware MiddlewareFunc) Option[T] {
	return WithMiddleware[T](jwtMiddleware)
}
