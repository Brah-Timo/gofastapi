// Package hooks provides the lifecycle hook system for gofastapi.
//
// Hooks are user-defined functions executed at specific points in the
// CRUD request lifecycle. They allow you to inject business logic without
// modifying the framework internals.
//
// Available hook points (in execution order):
//
//	BeforeCreate  → runs before INSERT, receives *T  — return error to abort
//	AfterCreate   → runs after  INSERT, receives *T  — errors are logged only
//	BeforeUpdate  → runs before UPDATE, receives *T  — return error to abort
//	AfterUpdate   → runs after  UPDATE, receives *T  — errors are logged only
//	BeforeDelete  → runs before DELETE, receives *T  — return error to abort
//	AfterDelete   → runs after  DELETE, receives *T  — errors are logged only
//	AfterFind     → runs after  SELECT (Show only), receives *T
//
// Multiple hooks of the same type run in registration order.
// A hook returning a non-nil error from a "Before" hook aborts the operation.
// Errors from "After" hooks are silently ignored (the operation already succeeded).
//
// Example:
//
//	gofastapi.CRUD[User]("/users", db,
//	    crud.WithBeforeCreate[User](func(u *User, ctx crud.Context) error {
//	        if u.Role == "superadmin" {
//	            return errors.New("cannot create superadmin via API")
//	        }
//	        u.Role = "member" // enforce default
//	        return nil
//	    }),
//	    crud.WithAfterCreate[User](func(u *User, ctx crud.Context) error {
//	        go sendWelcomeEmail(u.Email) // fire-and-forget
//	        return nil
//	    }),
//	)
package hooks

import "fmt"

// ─────────────────────────────────────────────────────────────────────────────
// HookType — the event name
// ─────────────────────────────────────────────────────────────────────────────

// HookType identifies when a hook is fired.
type HookType string

const (
	// BeforeCreate fires before a new record is written to the database.
	BeforeCreate HookType = "before_create"
	// AfterCreate fires after a new record has been written.
	AfterCreate HookType = "after_create"
	// BeforeUpdate fires before an existing record is overwritten.
	BeforeUpdate HookType = "before_update"
	// AfterUpdate fires after an existing record has been overwritten.
	AfterUpdate HookType = "after_update"
	// BeforeDelete fires before a record is removed.
	BeforeDelete HookType = "before_delete"
	// AfterDelete fires after a record has been removed.
	AfterDelete HookType = "after_delete"
	// AfterFind fires after a single record has been fetched (Show endpoint).
	AfterFind HookType = "after_find"
)

// ─────────────────────────────────────────────────────────────────────────────
// Context — minimal interface used by hooks (mirrors crud.Context)
// ─────────────────────────────────────────────────────────────────────────────

// Context is the request context available to all hook functions.
// It mirrors crud.Context to avoid an import cycle.
type Context interface {
	Set(key string, value any)
	Get(key string) (any, bool)
	ClientIP() string
}

// ─────────────────────────────────────────────────────────────────────────────
// HookFunc — the user-defined callback type
// ─────────────────────────────────────────────────────────────────────────────

// HookFunc is the signature of a lifecycle hook function.
// item is a pointer to the model being processed (modifications are allowed).
// ctx is the request context; use it to access headers, auth claims, etc.
// Returning a non-nil error from a Before hook aborts the operation.
type HookFunc[T any] func(item *T, ctx Context) error

// ─────────────────────────────────────────────────────────────────────────────
// Registry — stores and executes hooks for a single model type T
// ─────────────────────────────────────────────────────────────────────────────

// Registry stores all hooks registered for type T.
// It is created once per Handler and is safe for concurrent reads after setup.
type Registry[T any] struct {
	hooks map[HookType][]HookFunc[T]
}

// NewRegistry creates an empty Registry for type T.
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		hooks: make(map[HookType][]HookFunc[T]),
	}
}

// Register adds fn to the list of hooks for hookType.
// Hooks run in registration order.
func (r *Registry[T]) Register(hookType HookType, fn HookFunc[T]) {
	r.hooks[hookType] = append(r.hooks[hookType], fn)
}

// Run executes all hooks registered for hookType in order.
//
// Semantics:
//   - Before hooks: if any hook returns a non-nil error, Run returns it
//     immediately (subsequent hooks are skipped and the operation is aborted).
//   - After hooks: errors are wrapped and returned but the caller (Handler)
//     logs them and continues — the operation has already succeeded.
func (r *Registry[T]) Run(hookType HookType, item *T, ctx Context) error {
	fns, ok := r.hooks[hookType]
	if !ok || len(fns) == 0 {
		return nil
	}

	isBefore := hookType == BeforeCreate ||
		hookType == BeforeUpdate ||
		hookType == BeforeDelete

	var errs []error
	for _, fn := range fns {
		if err := fn(item, ctx); err != nil {
			if isBefore {
				// Abort immediately on Before errors.
				return fmt.Errorf("hook %s: %w", hookType, err)
			}
			errs = append(errs, err)
		}
	}

	// For After hooks, return the first error if any.
	if len(errs) > 0 {
		return fmt.Errorf("hook %s: %w", hookType, errs[0])
	}
	return nil
}

// Len returns the number of hooks registered for hookType.
func (r *Registry[T]) Len(hookType HookType) int {
	return len(r.hooks[hookType])
}

// Clear removes all hooks. Useful in tests.
func (r *Registry[T]) Clear(hookType HookType) {
	delete(r.hooks, hookType)
}

// ClearAll removes every registered hook.
func (r *Registry[T]) ClearAll() {
	r.hooks = make(map[HookType][]HookFunc[T])
}
