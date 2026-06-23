// Package db provides the database abstraction layer for gofastapi.
//
// It wraps GORM (the default) and exposes a minimal Database interface that
// the CRUD layer depends on, making the underlying engine swappable without
// touching business logic.
//
// Driver detection is automatic based on the DSN prefix:
//
//	"postgres://" | "postgresql://" | "host=" → PostgreSQL via pgx
//	"mysql://"    | "user=…@tcp("             → MySQL / MariaDB
//	"sqlite://"   | ":memory:"  | "file:"     → SQLite
//
// # Usage
//
//	db, err := db.AutoConnect("postgres://user:pass@localhost/mydb")
//	if err != nil { log.Fatal(err) }
//	defer db.Close()
//
//	// Run GORM automigrate for a struct
//	err = db.Migrate[User]()
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ─────────────────────────────────────────────────────────────────────────────
// Database interface — the contract the CRUD layer depends on
// ─────────────────────────────────────────────────────────────────────────────

// Database is the minimal interface the rest of gofastapi needs.
// It intentionally exposes only what is necessary so that custom
// implementations (e.g., mock databases in tests) are easy to write.
type Database interface {
	// DB returns the underlying *gorm.DB for advanced queries.
	DB() *gorm.DB
	// SQL returns the *sql.DB for raw queries or connection pool control.
	SQL() *sql.DB
	// Ping verifies that the connection is alive.
	Ping(ctx context.Context) error
	// Close releases all database resources.
	Close() error
	// Driver returns the detected driver name ("postgres", "mysql", "sqlite").
	Driver() string
}

// ─────────────────────────────────────────────────────────────────────────────
// GORMDatabase — concrete implementation
// ─────────────────────────────────────────────────────────────────────────────

// GORMDatabase wraps a *gorm.DB and satisfies the Database interface.
type GORMDatabase struct {
	gdb    *gorm.DB
	sqlDB  *sql.DB
	driver string
}

func (g *GORMDatabase) DB() *gorm.DB   { return g.gdb }
func (g *GORMDatabase) SQL() *sql.DB   { return g.sqlDB }
func (g *GORMDatabase) Driver() string { return g.driver }

func (g *GORMDatabase) Ping(ctx context.Context) error {
	return g.sqlDB.PingContext(ctx)
}

func (g *GORMDatabase) Close() error {
	return g.sqlDB.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// Connection options
// ─────────────────────────────────────────────────────────────────────────────

// Options holds tunable connection pool and logging settings.
type Options struct {
	// MaxOpenConns is the maximum number of open database connections.
	// Default: 25.
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections.
	// Default: 5.
	MaxIdleConns int
	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	// Default: 5 minutes.
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum amount of time an idle connection is kept.
	// Default: 1 minute.
	ConnMaxIdleTime time.Duration
	// Debug enables GORM's verbose SQL logging. Useful in development.
	// Default: false.
	Debug bool
	// SlowThreshold marks queries slower than this duration as warnings.
	// Default: 200 ms.
	SlowThreshold time.Duration
}

func defaultOptions() Options {
	return Options{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		Debug:           false,
		SlowThreshold:   200 * time.Millisecond,
	}
}

// Option is a functional option for database connection.
type Option func(*Options)

// WithMaxOpenConns sets the maximum number of open database connections.
func WithMaxOpenConns(n int) Option { return func(o *Options) { o.MaxOpenConns = n } }

// WithMaxIdleConns sets the maximum number of idle connections.
func WithMaxIdleConns(n int) Option { return func(o *Options) { o.MaxIdleConns = n } }

// WithConnMaxLifetime sets the maximum connection lifetime.
func WithConnMaxLifetime(d time.Duration) Option {
	return func(o *Options) { o.ConnMaxLifetime = d }
}

// WithDebug enables verbose SQL logging.
func WithDebug() Option { return func(o *Options) { o.Debug = true } }

// WithSlowThreshold sets the slow-query warning threshold.
func WithSlowThreshold(d time.Duration) Option {
	return func(o *Options) { o.SlowThreshold = d }
}

// ─────────────────────────────────────────────────────────────────────────────
// AutoConnect — driver detection + connection
// ─────────────────────────────────────────────────────────────────────────────

// ErrUnknownDriver is returned when AutoConnect cannot determine the driver
// from the DSN.
var ErrUnknownDriver = errors.New("db: cannot determine driver from DSN — prefix with 'postgres://', 'mysql://', or 'sqlite://'")

// AutoConnect automatically determines the database driver from the DSN and
// opens a connection with sensible defaults.
//
//	dsn = "postgres://user:pass@localhost:5432/mydb"  → PostgreSQL
//	dsn = "mysql://user:pass@tcp(localhost:3306)/mydb" → MySQL
//	dsn = "sqlite://./app.db"                          → SQLite
//	dsn = ":memory:"                                    → SQLite in-memory
func AutoConnect(dsn string, opts ...Option) (Database, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	driver := detectDriver(dsn)
	if driver == "" {
		return nil, ErrUnknownDriver
	}

	var gdb *gorm.DB
	var err error

	logLevel := logger.Silent
	if o.Debug {
		logLevel = logger.Info
	}

	gormCfg := &gorm.Config{
		Logger:  logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time { return time.Now().UTC() },
	}

	switch driver {
	case "postgres":
		gdb, err = connectPostgres(dsn, gormCfg)
	case "mysql":
		gdb, err = connectMySQL(dsn, gormCfg)
	case "sqlite":
		gdb, err = connectSQLite(dsn, gormCfg)
	default:
		return nil, fmt.Errorf("db: unsupported driver %q", driver)
	}

	if err != nil {
		return nil, fmt.Errorf("db: connect(%s): %w", driver, err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("db: get sql.DB: %w", err)
	}

	// Connection pool settings.
	sqlDB.SetMaxOpenConns(o.MaxOpenConns)
	sqlDB.SetMaxIdleConns(o.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(o.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(o.ConnMaxIdleTime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("db: ping failed: %w", err)
	}

	return &GORMDatabase{
		gdb:    gdb,
		sqlDB:  sqlDB,
		driver: driver,
	}, nil
}

// MustConnect is like AutoConnect but panics instead of returning an error.
// Useful in main() where a failed DB connection should abort startup.
func MustConnect(dsn string, opts ...Option) Database {
	d, err := AutoConnect(dsn, opts...)
	if err != nil {
		panic(fmt.Sprintf("db.MustConnect: %v", err))
	}
	return d
}

// ─────────────────────────────────────────────────────────────────────────────
// Migrate
// ─────────────────────────────────────────────────────────────────────────────

// Migrate runs GORM's AutoMigrate for type T.
// It creates the table if it doesn't exist, adds missing columns, and adds
// missing indexes. It does NOT delete columns or change existing data.
func Migrate[T any](database Database) error {
	var model T
	return database.DB().AutoMigrate(&model)
}

// MigrateModels runs GORM's AutoMigrate for a slice of model instances.
// Useful when you want to migrate all models in one call.
func MigrateModels(database Database, models ...any) error {
	return database.DB().AutoMigrate(models...)
}

// ─────────────────────────────────────────────────────────────────────────────
// Driver detection (internal)
// ─────────────────────────────────────────────────────────────────────────────

func detectDriver(dsn string) string {
	lower := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lower, "postgres://"),
		strings.HasPrefix(lower, "postgresql://"),
		strings.HasPrefix(lower, "host="),
		strings.Contains(lower, "sslmode="):
		return "postgres"
	case strings.HasPrefix(lower, "mysql://"),
		strings.Contains(lower, "@tcp("),
		strings.Contains(lower, "parseTime="):
		return "mysql"
	case strings.HasPrefix(lower, "sqlite://"),
		lower == ":memory:",
		strings.HasPrefix(lower, "file:"),
		strings.HasSuffix(lower, ".db"),
		strings.HasSuffix(lower, ".sqlite"),
		strings.HasSuffix(lower, ".sqlite3"):
		return "sqlite"
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Health & Stats helpers (public utility functions)
// ─────────────────────────────────────────────────────────────────────────────

// Stats returns the current database connection pool statistics.
func Stats(database Database) sql.DBStats {
	return database.SQL().Stats()
}

// IsHealthy pings the database and returns true if it responds.
func IsHealthy(database Database) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return database.Ping(ctx) == nil
}
