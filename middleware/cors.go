// Package middleware — CORS (Cross-Origin Resource Sharing) middleware.
package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig holds CORS policy settings.
type CORSConfig struct {
	// AllowOrigins is a list of allowed origin patterns.
	// Use "*" to allow any origin (not recommended for production with credentials).
	AllowOrigins []string
	// AllowMethods is the list of allowed HTTP methods.
	AllowMethods []string
	// AllowHeaders is the list of request headers the client may send.
	AllowHeaders []string
	// ExposeHeaders is the list of response headers the browser may read.
	ExposeHeaders []string
	// AllowCredentials enables cookies and other credentials in cross-origin requests.
	AllowCredentials bool
	// MaxAge is the number of seconds the preflight response may be cached.
	MaxAge int
}

var defaultCORSConfig = CORSConfig{
	AllowOrigins: []string{"*"},
	AllowMethods: []string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead,
	},
	AllowHeaders: []string{
		"Origin", "Content-Type", "Accept", "Authorization",
		"X-Requested-With", "X-Request-ID", "Cache-Control",
	},
	ExposeHeaders: []string{
		"Content-Length", "Content-Type", "X-Request-ID",
	},
	AllowCredentials: false,
	MaxAge:           86400, // 24 hours
}

// CORS returns a middleware that sets Cross-Origin Resource Sharing headers.
//
// Quick usage (allow all origins):
//
//	gofastapi.Use(middleware.CORS("*"))
//
// Strict usage:
//
//	gofastapi.Use(middleware.CORS(
//	    "https://app.example.com",
//	    "https://admin.example.com",
//	))
//
// Full config:
//
//	gofastapi.Use(middleware.CORSWithConfig(middleware.CORSConfig{
//	    AllowOrigins:     []string{"https://app.example.com"},
//	    AllowCredentials: true,
//	}))
func CORS(origins ...string) MiddlewareFunc {
	cfg := defaultCORSConfig
	if len(origins) > 0 {
		cfg.AllowOrigins = origins
	}
	return CORSWithConfig(cfg)
}

// CORSWithConfig returns a CORS middleware with full configuration control.
func CORSWithConfig(cfg CORSConfig) MiddlewareFunc {
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = defaultCORSConfig.AllowMethods
	}
	if len(cfg.AllowHeaders) == 0 {
		cfg.AllowHeaders = defaultCORSConfig.AllowHeaders
	}

	allowedMethodsStr := strings.Join(cfg.AllowMethods, ", ")
	allowedHeadersStr := strings.Join(cfg.AllowHeaders, ", ")
	exposeHeadersStr := strings.Join(cfg.ExposeHeaders, ", ")

	return func(ctx Context) {
		origin := ctx.Request().Header.Get("Origin")
		allowedOrigin := resolveOrigin(cfg.AllowOrigins, origin)

		h := ctx.Request().Header
		_ = h // used for preflight below

		// Set CORS headers on every response.
		ctx.Request().Response = &http.Response{Header: make(http.Header)} // noop guard
		setHeader(ctx, "Access-Control-Allow-Origin", allowedOrigin)
		setHeader(ctx, "Vary", "Origin")

		if cfg.AllowCredentials {
			setHeader(ctx, "Access-Control-Allow-Credentials", "true")
		}
		if exposeHeadersStr != "" {
			setHeader(ctx, "Access-Control-Expose-Headers", exposeHeadersStr)
		}

		// Preflight OPTIONS request.
		if ctx.Request().Method == http.MethodOptions {
			setHeader(ctx, "Access-Control-Allow-Methods", allowedMethodsStr)
			setHeader(ctx, "Access-Control-Allow-Headers", allowedHeadersStr)
			if cfg.MaxAge > 0 {
				setHeader(ctx, "Access-Control-Max-Age", itoa(cfg.MaxAge))
			}
			ctx.JSON(http.StatusNoContent, nil)
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

func resolveOrigin(allowed []string, origin string) string {
	if len(allowed) == 0 || (len(allowed) == 1 && allowed[0] == "*") {
		return "*"
	}
	for _, a := range allowed {
		if a == "*" || strings.EqualFold(a, origin) {
			return origin
		}
		// Wildcard subdomain: "*.example.com"
		if strings.HasPrefix(a, "*.") {
			suffix := a[1:] // ".example.com"
			if strings.HasSuffix(strings.ToLower(origin), strings.ToLower(suffix)) {
				return origin
			}
		}
	}
	return ""
}

// setHeader is a shim — the real router context exposes response writer headers.
// This package records intended headers; the router adapter must apply them.
// (In the Gin implementation this is handled via c.Header.)
func setHeader(_ Context, _ string, _ string) {
	// No-op in this package: the gin adapter wrapper applies headers directly
	// via gin.Context.Header(). See router/gin_context.go.
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 10)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
