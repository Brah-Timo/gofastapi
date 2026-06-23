// Package config provides environment-aware configuration loading for gofastapi.
//
// Configuration is loaded in order of precedence (highest first):
//  1. Environment variables (prefixed with GOFASTAPI_)
//  2. A config file (.yaml / .json / .toml) specified via GOFASTAPI_CONFIG_FILE
//  3. Built-in defaults
//
// Example .env / environment:
//
//	GOFASTAPI_SERVER_PORT=:9000
//	GOFASTAPI_DATABASE_DSN=postgres://user:pass@localhost/mydb
//	GOFASTAPI_JWT_SECRET=super-secret-key
//	GOFASTAPI_DEBUG=true
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Config — the top-level configuration struct
// ─────────────────────────────────────────────────────────────────────────────

// Config holds all runtime configuration for a gofastapi application.
type Config struct {
	// Server contains HTTP server settings.
	Server ServerConfig `json:"server"`
	// Database contains database connection settings.
	Database DatabaseConfig `json:"database"`
	// JWT contains authentication token settings.
	JWT JWTConfig `json:"jwt"`
	// RateLimit contains rate limiting settings.
	RateLimit RateLimitConfig `json:"rate_limit"`
	// Log contains logging configuration.
	Log LogConfig `json:"log"`
	// Debug enables verbose logging and development mode features.
	Debug bool `json:"debug"`
	// AppName is a human-readable name for the application.
	AppName string `json:"app_name"`
	// AppVersion is the current version string (shown in health checks).
	AppVersion string `json:"app_version"`
	// AppEnv is the environment name: "development", "staging", "production".
	AppEnv string `json:"app_env"`
}

// ServerConfig holds HTTP server knobs.
type ServerConfig struct {
	// Port is the listen address (e.g. ":8080" or "0.0.0.0:443").
	Port string `json:"port"`
	// ReadTimeout is the max duration for reading the request (seconds).
	ReadTimeout int `json:"read_timeout"`
	// WriteTimeout is the max duration for writing the response (seconds).
	WriteTimeout int `json:"write_timeout"`
	// MaxHeaderBytes is the max size of request headers.
	MaxHeaderBytes int `json:"max_header_bytes"`
	// GracefulTimeout is the max seconds to wait for in-flight requests on shutdown.
	GracefulTimeout int `json:"graceful_timeout"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	// DSN is the database connection string. Supports postgres://, mysql://, sqlite://.
	DSN string `json:"dsn"`
	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int `json:"max_open_conns"`
	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int `json:"max_idle_conns"`
	// ConnMaxLifetimeSec is the connection max lifetime in seconds.
	ConnMaxLifetimeSec int `json:"conn_max_lifetime_sec"`
	// Debug enables SQL query logging.
	Debug bool `json:"debug"`
}

// ConnMaxLifetime returns ConnMaxLifetimeSec as a time.Duration.
func (d DatabaseConfig) ConnMaxLifetime() time.Duration {
	return time.Duration(d.ConnMaxLifetimeSec) * time.Second
}

// JWTConfig holds JSON Web Token settings.
type JWTConfig struct {
	// Secret is the HMAC signing key for JWT tokens.
	Secret string `json:"secret"`
	// ExpiryHours is the token lifetime in hours (default: 24).
	ExpiryHours int `json:"expiry_hours"`
	// Issuer is the `iss` claim value (optional).
	Issuer string `json:"issuer"`
}

// Expiry returns ExpiryHours as a time.Duration.
func (j JWTConfig) Expiry() time.Duration {
	if j.ExpiryHours <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(j.ExpiryHours) * time.Hour
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	// Enabled controls whether rate limiting is active.
	Enabled bool `json:"enabled"`
	// RequestsPerMinute is the sustained rate per IP.
	RequestsPerMinute int `json:"requests_per_minute"`
	// Burst is the maximum instantaneous burst above the sustained rate.
	Burst int `json:"burst"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	// Level is the minimum log level: "debug", "info", "warn", "error".
	Level string `json:"level"`
	// Format is the output format: "json" or "text".
	Format string `json:"format"`
	// Output is the log destination: "stdout", "stderr", or a file path.
	Output string `json:"output"`
}

// ─────────────────────────────────────────────────────────────────────────────
// LoadFromEnv — primary config loader
// ─────────────────────────────────────────────────────────────────────────────

// LoadFromEnv loads configuration from environment variables.
// Variables are prefixed with GOFASTAPI_ (case-insensitive).
//
// Full list of supported variables:
//
//	GOFASTAPI_SERVER_PORT              (default: ":8080")
//	GOFASTAPI_SERVER_READ_TIMEOUT      (default: 30)
//	GOFASTAPI_SERVER_WRITE_TIMEOUT     (default: 30)
//	GOFASTAPI_SERVER_MAX_HEADER_BYTES  (default: 1048576)
//	GOFASTAPI_SERVER_GRACEFUL_TIMEOUT  (default: 5)
//	GOFASTAPI_DATABASE_DSN             (default: "")
//	GOFASTAPI_DATABASE_MAX_OPEN_CONNS  (default: 25)
//	GOFASTAPI_DATABASE_MAX_IDLE_CONNS  (default: 5)
//	GOFASTAPI_JWT_SECRET               (default: "change-me")
//	GOFASTAPI_JWT_EXPIRY_HOURS         (default: 24)
//	GOFASTAPI_RATE_LIMIT_ENABLED       (default: false)
//	GOFASTAPI_RATE_LIMIT_RPM           (default: 1000)
//	GOFASTAPI_LOG_LEVEL                (default: "info")
//	GOFASTAPI_LOG_FORMAT               (default: "text")
//	GOFASTAPI_DEBUG                    (default: false)
//	GOFASTAPI_APP_ENV                  (default: "development")
//	DATABASE_URL                       (alias for GOFASTAPI_DATABASE_DSN)
func LoadFromEnv() *Config {
	cfg := Defaults()

	// Server
	cfg.Server.Port = envStr("GOFASTAPI_SERVER_PORT", envStr("PORT", ":8080"))
	cfg.Server.ReadTimeout = envInt("GOFASTAPI_SERVER_READ_TIMEOUT", 30)
	cfg.Server.WriteTimeout = envInt("GOFASTAPI_SERVER_WRITE_TIMEOUT", 30)
	cfg.Server.MaxHeaderBytes = envInt("GOFASTAPI_SERVER_MAX_HEADER_BYTES", 1<<20)
	cfg.Server.GracefulTimeout = envInt("GOFASTAPI_SERVER_GRACEFUL_TIMEOUT", 5)

	// Database — support both DATABASE_URL and GOFASTAPI_DATABASE_DSN
	cfg.Database.DSN = envStr("GOFASTAPI_DATABASE_DSN",
		envStr("DATABASE_URL", ""))
	cfg.Database.MaxOpenConns = envInt("GOFASTAPI_DATABASE_MAX_OPEN_CONNS", 25)
	cfg.Database.MaxIdleConns = envInt("GOFASTAPI_DATABASE_MAX_IDLE_CONNS", 5)
	cfg.Database.ConnMaxLifetimeSec = envInt("GOFASTAPI_DATABASE_CONN_MAX_LIFETIME", 300)
	cfg.Database.Debug = envBool("GOFASTAPI_DATABASE_DEBUG", false)

	// JWT
	cfg.JWT.Secret = envStr("GOFASTAPI_JWT_SECRET", envStr("JWT_SECRET", "change-me-in-production"))
	cfg.JWT.ExpiryHours = envInt("GOFASTAPI_JWT_EXPIRY_HOURS", 24)
	cfg.JWT.Issuer = envStr("GOFASTAPI_JWT_ISSUER", "")

	// Rate limiting
	cfg.RateLimit.Enabled = envBool("GOFASTAPI_RATE_LIMIT_ENABLED", false)
	cfg.RateLimit.RequestsPerMinute = envInt("GOFASTAPI_RATE_LIMIT_RPM", 1000)
	cfg.RateLimit.Burst = envInt("GOFASTAPI_RATE_LIMIT_BURST", 1000)

	// Logging
	cfg.Log.Level = envStr("GOFASTAPI_LOG_LEVEL", "info")
	cfg.Log.Format = envStr("GOFASTAPI_LOG_FORMAT", "text")
	cfg.Log.Output = envStr("GOFASTAPI_LOG_OUTPUT", "stdout")

	// App metadata
	cfg.Debug = envBool("GOFASTAPI_DEBUG", false)
	cfg.AppName = envStr("GOFASTAPI_APP_NAME", "gofastapi-app")
	cfg.AppVersion = envStr("GOFASTAPI_APP_VERSION", "1.0.0")
	cfg.AppEnv = envStr("GOFASTAPI_APP_ENV", envStr("APP_ENV", "development"))

	return cfg
}

// Defaults returns a Config populated with built-in defaults.
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            ":8080",
			ReadTimeout:     30,
			WriteTimeout:    30,
			MaxHeaderBytes:  1 << 20,
			GracefulTimeout: 5,
		},
		Database: DatabaseConfig{
			MaxOpenConns:       25,
			MaxIdleConns:       5,
			ConnMaxLifetimeSec: 300,
		},
		JWT: JWTConfig{
			Secret:      "change-me-in-production",
			ExpiryHours: 24,
		},
		RateLimit: RateLimitConfig{
			Enabled:           false,
			RequestsPerMinute: 1000,
			Burst:             1000,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		Debug:      false,
		AppName:    "gofastapi-app",
		AppVersion: "1.0.0",
		AppEnv:     "development",
	}
}

// IsDevelopment reports whether the app is running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development" || c.AppEnv == "dev"
}

// IsProduction reports whether the app is running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production" || c.AppEnv == "prod"
}

// String returns a human-readable summary of the configuration.
// Sensitive values (DSN, JWT secret) are redacted.
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{app=%s v%s env=%s port=%s db=%s}",
		c.AppName, c.AppVersion, c.AppEnv, c.Server.Port,
		redactDSN(c.Database.DSN),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return def
}

func redactDSN(dsn string) string {
	if dsn == "" {
		return "<not set>"
	}
	// Replace password segment with ***.
	for _, scheme := range []string{"postgres://", "postgresql://", "mysql://"} {
		if strings.HasPrefix(dsn, scheme) {
			rest := dsn[len(scheme):]
			if atIdx := strings.Index(rest, "@"); atIdx != -1 {
				userInfo := rest[:atIdx]
				if colonIdx := strings.Index(userInfo, ":"); colonIdx != -1 {
					redacted := scheme + userInfo[:colonIdx+1] + "***" + rest[atIdx:]
					return redacted
				}
			}
		}
	}
	return "<set>"
}
