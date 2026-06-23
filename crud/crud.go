// Package crud is the heart of gofastapi.
//
// It ties together the Handler, Repository, Validator, and Hooks subsystems
// into a cohesive whole, and provides the top-level CRUD[T] function that
// the main gofastapi package re-exports.
//
// # Architecture
//
//	┌────────────────────────────────────────────────────┐
//	│  HTTP Request                                       │
//	└──────────────┬─────────────────────────────────────┘
//	               │
//	┌──────────────▼─────────────────────────────────────┐
//	│  Handler[T]                                         │
//	│  • Bind & validate                                  │
//	│  • Run lifecycle hooks (Before/After)               │
//	│  • Call Repository[T]                               │
//	│  • Render JSON response                             │
//	└──────────────┬─────────────────────────────────────┘
//	               │
//	┌──────────────▼─────────────────────────────────────┐
//	│  Repository[T]  (default: GORMRepository[T])        │
//	│  • List / FindByID / Create / Update / Delete        │
//	│  • Search, pagination, ordering                      │
//	└────────────────────────────────────────────────────┘
//
// # Generics contract
//
// T can be any struct. No interface needs to be implemented.
// gofastapi introspects T via the internal/reflect package to determine
// the primary key field and database column names.
//
// # Thread safety
//
// Handler[T] instances are immutable after construction and are safe for
// concurrent use by multiple goroutines.
package crud

// Re-export the Context type alias so callers can use crud.Context without
// importing the internal handler package separately.
// (Context is already defined in handler.go of this package.)

// Re-export hook constants for convenience:
//
//	crud.BeforeCreate  →  hooks.BeforeCreate
//	crud.AfterCreate   →  hooks.AfterCreate
//	…etc.
import "github.com/Brah-Timo/gofastapi/hooks"

const (
	// HookBeforeCreate is fired before a new record is written to the database.
	HookBeforeCreate = hooks.BeforeCreate
	// HookAfterCreate is fired after a new record is written.
	HookAfterCreate = hooks.AfterCreate
	// HookBeforeUpdate is fired before an existing record is overwritten.
	HookBeforeUpdate = hooks.BeforeUpdate
	// HookAfterUpdate is fired after an existing record is overwritten.
	HookAfterUpdate = hooks.AfterUpdate
	// HookBeforeDelete is fired before a record is removed.
	HookBeforeDelete = hooks.BeforeDelete
	// HookAfterDelete is fired after a record is removed.
	HookAfterDelete = hooks.AfterDelete
	// HookAfterFind is fired after a single record is fetched (Show only).
	HookAfterFind = hooks.AfterFind
)
