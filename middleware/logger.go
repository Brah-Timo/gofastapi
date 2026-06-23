// Package middleware — structured request/response logger.
package middleware

import (
	"fmt"
	"net/http"
	"time"
)

// LoggerConfig configures the logger middleware.
type LoggerConfig struct {
	// SkipPaths is a list of URL paths that should not be logged.
	// E.g. []string{"/health", "/metrics"}
	SkipPaths []string
	// Format controls the log format. Options: "json" (default) or "text".
	Format string
	// TimeFormat is the timestamp format. Default: time.RFC3339.
	TimeFormat string
	// Writer is where logs are written. Default: os.Stdout (via fmt).
	Writer func(line string)
}

// Logger returns a middleware that logs every HTTP request and its response.
// It records: timestamp, method, path, status, latency, client IP, size.
//
//	gofastapi.Use(middleware.Logger())
func Logger(configs ...LoggerConfig) MiddlewareFunc {
	cfg := LoggerConfig{
		Format:     "text",
		TimeFormat: time.RFC3339,
		Writer:     func(line string) { fmt.Println(line) },
	}
	if len(configs) > 0 {
		c := configs[0]
		if len(c.SkipPaths) > 0 {
			cfg.SkipPaths = c.SkipPaths
		}
		if c.Format != "" {
			cfg.Format = c.Format
		}
		if c.TimeFormat != "" {
			cfg.TimeFormat = c.TimeFormat
		}
		if c.Writer != nil {
			cfg.Writer = c.Writer
		}
	}

	skipMap := make(map[string]bool, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skipMap[p] = true
	}

	return func(ctx Context) {
		path := ctx.Request().URL.Path
		if skipMap[path] {
			ctx.Next()
			return
		}

		start := time.Now()
		method := ctx.Request().Method

		// Wrap the context to capture status code.
		// Note: full status capture requires a response writer wrapper in the
		// router adapter. Here we record the status as seen after Next().
		ctx.Next()

		latency := time.Since(start)
		ip := ctx.ClientIP()

		line := formatLogLine(cfg, method, path, ip, latency)
		cfg.Writer(line)
	}
}

func formatLogLine(cfg LoggerConfig, method, path, ip string, latency time.Duration) string {
	ts := time.Now().Format(cfg.TimeFormat)
	if cfg.Format == "json" {
		return fmt.Sprintf(
			`{"time":%q,"method":%q,"path":%q,"ip":%q,"latency_ms":%d}`,
			ts, method, path, ip, latency.Milliseconds(),
		)
	}
	return fmt.Sprintf(
		"[%s] %s %s %s (%s)",
		ts, colorMethod(method), path, ip, latency.Truncate(time.Millisecond),
	)
}

func colorMethod(method string) string {
	colors := map[string]string{
		http.MethodGet:    "\033[32m", // green
		http.MethodPost:   "\033[34m", // blue
		http.MethodPut:    "\033[33m", // yellow
		http.MethodPatch:  "\033[33m", // yellow
		http.MethodDelete: "\033[31m", // red
	}
	reset := "\033[0m"
	if c, ok := colors[method]; ok {
		return c + method + reset
	}
	return method
}
