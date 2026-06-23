// Package middleware — panic recovery middleware.
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

// RecoveryConfig configures the recovery middleware.
type RecoveryConfig struct {
	// PrintStack controls whether the stack trace is printed to stdout.
	PrintStack bool
	// OnPanic is called when a panic is recovered. Use it for error reporting.
	OnPanic func(ctx Context, err any, stack []byte)
}

// Recovery returns a middleware that recovers from panics, logs the stack
// trace, and returns a 500 Internal Server Error response.
//
// This should be the outermost middleware in your chain.
//
//	gofastapi.Use(middleware.Recovery())
func Recovery(configs ...RecoveryConfig) MiddlewareFunc {
	cfg := RecoveryConfig{
		PrintStack: true,
		OnPanic:    nil,
	}
	if len(configs) > 0 {
		cfg = configs[0]
	}

	return func(ctx Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()

				if cfg.PrintStack {
					fmt.Printf("[PANIC RECOVERED]\n%v\n\n%s\n", r, stack)
				}

				if cfg.OnPanic != nil {
					cfg.OnPanic(ctx, r, stack)
				}

				ctx.JSON(http.StatusInternalServerError, map[string]any{
					"success": false,
					"error": map[string]string{
						"code":    "INTERNAL_SERVER_ERROR",
						"message": "an unexpected error occurred",
					},
				})
				ctx.Abort()
			}
		}()
		ctx.Next()
	}
}
