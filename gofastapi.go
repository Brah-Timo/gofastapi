// Package gofastapi provides a zero-boilerplate, production-ready REST API
// framework for Go 1.21+. It leverages generics to auto-generate full CRUD
// endpoints from any struct, eliminating repetitive boilerplate while
// preserving full control through a rich Functional-Options API.
//
// Philosophy: "Convention over Configuration" — sane defaults for everything,
// override only what you need.
//
// Basic usage (5 lines → 5 endpoints):
//
//	type User struct {
//	    ID    uint   `json:"id"    gorm:"primaryKey"`
//	    Name  string `json:"name"  validate:"required,min=2"`
//	    Email string `json:"email" validate:"required,email" gorm:"uniqueIndex"`
//	}
//
//	func main() {
//	    db := gofastapi.ConnectDB("postgres://user:pass@localhost/mydb")
//	    gofastapi.AutoMigrate[User](db)
//	    gofastapi.CRUD[User]("/users", db)
//	    gofastapi.Run(":8080")
//	}
//
// The above registers:
//
//	GET    /users       → paginated list (search, sort, filter)
//	GET    /users/:id   → single record (404 if absent)
//	POST   /users       → create with full validation
//	PUT    /users/:id   → update with full validation
//	DELETE /users/:id   → hard or soft delete
package gofastapi

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Brah-Timo/gofastapi/config"
	"github.com/Brah-Timo/gofastapi/crud"
	"github.com/Brah-Timo/gofastapi/db"
	"github.com/Brah-Timo/gofastapi/middleware"
	"github.com/Brah-Timo/gofastapi/router"
	"github.com/Brah-Timo/gofastapi/swagger"
)

// ─────────────────────────────────────────────────────────────────────────────
// App — the central application object
// ─────────────────────────────────────────────────────────────────────────────

// App represents a fully configured gofastapi application instance.
// Use New() to create a named instance, or rely on the package-level
// singleton (defaultApp) through the top-level helper functions.
type App struct {
	router *router.Router
	cfg    *config.Config
	db     db.Database
	logger *log.Logger
}

// New creates a new App with default configuration loaded from environment
// variables. Pass AppOption functions to override any default.
func New(opts ...AppOption) *App {
	a := &App{
		router: router.New(),
		cfg:    config.LoadFromEnv(),
		logger: log.New(os.Stdout, "[gofastapi] ", log.LstdFlags|log.Lshortfile),
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// ─────────────────────────────────────────────────────────────────────────────
// AppOption — functional options for App
// ─────────────────────────────────────────────────────────────────────────────

// AppOption is a functional option for App configuration.
type AppOption func(*App)

// WithConfig overrides the configuration loaded from environment.
func WithConfig(cfg *config.Config) AppOption {
	return func(a *App) { a.cfg = cfg }
}

// WithDB injects a pre-configured database connection into the App.
func WithDB(database db.Database) AppOption {
	return func(a *App) { a.db = database }
}

// WithLogger sets a custom logger for the App.
func WithLogger(l *log.Logger) AppOption {
	return func(a *App) { a.logger = l }
}

// ─────────────────────────────────────────────────────────────────────────────
// CRUD — the magic one-liner
// ─────────────────────────────────────────────────────────────────────────────

// CRUD registers five REST endpoints for type T on the default App singleton.
// T must be a struct with at least an ID field (any primary-key type).
//
// Generated routes (relative to prefix):
//
//	GET    /         List   — paginated, searchable, sortable
//	GET    /:id      Show   — single record; 404 on miss
//	POST   /         Create — bind + validate + BeforeCreate + persist + AfterCreate
//	PUT    /:id      Update — fetch + bind + validate + BeforeUpdate + save + AfterUpdate
//	DELETE /:id      Delete — fetch + BeforeDelete + remove + AfterDelete
func CRUD[T any](prefix string, database db.Database, opts ...crud.Option[T]) {
	AppCRUD[T](defaultApp, prefix, database, opts...)
}

// AppCRUD registers five REST endpoints for type T on an App instance.
// Go does not allow generic methods on non-generic receiver types, so this
// is a package-level helper that accepts the App as its first argument.
//
//	gofastapi.AppCRUD[User](app, "/users", database)
func AppCRUD[T any](a *App, prefix string, database db.Database, opts ...crud.Option[T]) {
	h := crud.NewHandler[T](database, opts...)
	g := a.router.Group(prefix)
	// *router.Context implements crud.Context, so we bridge the signatures here.
	g.GET("", func(ctx *router.Context) { h.List(ctx) })
	g.GET("/:id", func(ctx *router.Context) { h.Show(ctx) })
	g.POST("", func(ctx *router.Context) { h.Create(ctx) })
	g.PUT("/:id", func(ctx *router.Context) { h.Update(ctx) })
	g.DELETE("/:id", func(ctx *router.Context) { h.Delete(ctx) })
}

// ─────────────────────────────────────────────────────────────────────────────
// Middleware helpers
// ─────────────────────────────────────────────────────────────────────────────

// Use adds one or more middleware functions to the global middleware chain of
// the default App singleton. Middleware is applied to every request.
func Use(mw ...router.MiddlewareFunc) {
	defaultApp.router.Use(mw...)
}

// Use adds middleware to this App instance.
func (a *App) Use(mw ...router.MiddlewareFunc) {
	a.router.Use(mw...)
}

// ─────────────────────────────────────────────────────────────────────────────
// Router helpers
// ─────────────────────────────────────────────────────────────────────────────

// Group creates a route group with a shared URL prefix on the default App.
func Group(prefix string, mw ...router.MiddlewareFunc) *router.RouterGroup {
	return defaultApp.router.Group(prefix, mw...)
}

// Group creates a route group on this App instance.
func (a *App) Group(prefix string, mw ...router.MiddlewareFunc) *router.RouterGroup {
	return a.router.Group(prefix, mw...)
}

// Handler returns the underlying http.Handler so the App can be mounted
// inside any standard HTTP server or used in tests via httptest.
func Handler() http.Handler {
	return defaultApp.router.Handler()
}

// Handler returns the http.Handler for this App instance.
func (a *App) Handler() http.Handler {
	return a.router.Handler()
}

// ─────────────────────────────────────────────────────────────────────────────
// Database helpers
// ─────────────────────────────────────────────────────────────────────────────

// ConnectDB establishes a database connection and returns a Database interface.
// The driver is detected automatically from the DSN prefix:
//
//	"postgres://"  or "host="   → PostgreSQL via pgx
//	"mysql://"                  → MySQL / MariaDB
//	"sqlite://"    or ":memory:" → SQLite (local dev / tests)
//
// Panics on connection failure to surface misconfiguration early.
func ConnectDB(dsn string) db.Database {
	d, err := db.AutoConnect(dsn)
	if err != nil {
		log.Fatalf("[gofastapi] ConnectDB failed: %v", err)
	}
	return d
}

// AutoMigrate runs GORM's AutoMigrate for type T, creating or altering the
// corresponding table to match the struct definition.
func AutoMigrate[T any](database db.Database) error {
	return db.Migrate[T](database)
}

// MustAutoMigrate is like AutoMigrate but panics on error.
func MustAutoMigrate[T any](database db.Database) {
	if err := db.Migrate[T](database); err != nil {
		log.Fatalf("[gofastapi] AutoMigrate failed: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Swagger
// ─────────────────────────────────────────────────────────────────────────────

// EnableSwagger mounts the Swagger UI at /docs on the default App.
// Call this after all CRUD/routes have been registered.
func EnableSwagger(title, version, description string) {
	defaultApp.EnableSwagger(title, version, description)
}

// EnableSwagger mounts the Swagger UI at /docs on this App instance.
func (a *App) EnableSwagger(title, version, description string) {
	h := swagger.NewHandler(title, version, description, a.router.Spec())
	// *router.Context satisfies both inline interfaces used by swagger.Handler.
	a.router.GET("/docs/*any", func(ctx *router.Context) { h.ServeHTTP(ctx) })
	a.router.GET("/openapi.json", func(ctx *router.Context) { h.ServeSpec(ctx) })
}

// ─────────────────────────────────────────────────────────────────────────────
// Server lifecycle
// ─────────────────────────────────────────────────────────────────────────────

// Run starts the HTTP server on the default App singleton.
// It blocks until an OS signal (SIGINT/SIGTERM) is received, then performs a
// graceful shutdown waiting up to GracefulTimeout seconds for active requests.
func Run(addr string) error {
	return defaultApp.Run(addr)
}

// Run starts the HTTP server for this App instance with graceful shutdown.
func (a *App) Run(addr string) error {
	srv := &http.Server{
		Addr:           addr,
		Handler:        a.router.Handler(),
		ReadTimeout:    time.Duration(a.cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(a.cfg.Server.WriteTimeout) * time.Second,
		MaxHeaderBytes: a.cfg.Server.MaxHeaderBytes,
	}

	// Channel to receive startup / fatal errors.
	errCh := make(chan error, 1)

	go func() {
		a.logger.Printf("🚀 Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for OS signal or server error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		a.logger.Printf("⚡ Signal %v received — graceful shutdown…", sig)
	}

	timeout := time.Duration(a.cfg.Server.GracefulTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	a.logger.Println("✅ Server stopped cleanly")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Package-level singleton
// ─────────────────────────────────────────────────────────────────────────────

// defaultApp is the implicit singleton used by package-level helpers
// (CRUD, Use, Group, Run, …). It is initialised once at startup so users can
// call gofastapi.CRUD[…] without ever calling New().
var defaultApp = New(
// Add sensible default middleware: structured logger + panic recovery.
// Users can call Use() to append more.
)

func init() {
	// middleware.MiddlewareFunc is func(middleware.Context); router.MiddlewareFunc is
	// func(*router.Context). Since *router.Context satisfies middleware.Context we
	// bridge the signatures with explicit lambdas.
	recovery := middleware.Recovery()
	logger := middleware.Logger()
	defaultApp.router.Use(
		func(ctx *router.Context) { recovery(ctx) },
		func(ctx *router.Context) { logger(ctx) },
	)
}
