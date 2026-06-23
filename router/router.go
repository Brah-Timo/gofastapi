// Package router provides the HTTP routing layer for gofastapi.
//
// It wraps Gin with a thin adapter that maps Gin's context to the
// crud.Context interface, keeping the rest of the framework independent
// of any specific HTTP library.
//
// The Router is created once per App and supports:
//   - Route groups with shared prefixes and middleware
//   - Global middleware (applied to every request)
//   - Standard HTTP methods: GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD
//   - OpenAPI spec generation (used by the swagger package)
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

// HandlerFunc is a gin-compatible handler that receives our Context wrapper.
type HandlerFunc func(ctx *Context)

// MiddlewareFunc is an alias for HandlerFunc (middleware and handlers share
// the same signature in gofastapi).
type MiddlewareFunc = HandlerFunc

// ─────────────────────────────────────────────────────────────────────────────
// Router
// ─────────────────────────────────────────────────────────────────────────────

// Router wraps a *gin.Engine and exposes a framework-agnostic API.
type Router struct {
	engine *gin.Engine
	spec   *OpenAPISpec
}

// New creates a Router backed by a Gin engine.
// In production (GIN_MODE=release) verbose debug logging is suppressed.
func New() *Router {
	// Suppress Gin startup banner.
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	return &Router{
		engine: engine,
		spec:   newOpenAPISpec(),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Middleware
// ─────────────────────────────────────────────────────────────────────────────

// Use adds global middleware to the router.
func (r *Router) Use(mw ...MiddlewareFunc) {
	for _, m := range mw {
		m := m // capture
		r.engine.Use(func(gc *gin.Context) {
			m(newGinContext(gc))
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Route registration helpers
// ─────────────────────────────────────────────────────────────────────────────

func (r *Router) GET(path string, handler HandlerFunc) {
	r.engine.GET(path, wrap(handler))
}

func (r *Router) POST(path string, handler HandlerFunc) {
	r.engine.POST(path, wrap(handler))
}

func (r *Router) PUT(path string, handler HandlerFunc) {
	r.engine.PUT(path, wrap(handler))
}

func (r *Router) PATCH(path string, handler HandlerFunc) {
	r.engine.PATCH(path, wrap(handler))
}

func (r *Router) DELETE(path string, handler HandlerFunc) {
	r.engine.DELETE(path, wrap(handler))
}

func (r *Router) OPTIONS(path string, handler HandlerFunc) {
	r.engine.OPTIONS(path, wrap(handler))
}

func (r *Router) HEAD(path string, handler HandlerFunc) {
	r.engine.HEAD(path, wrap(handler))
}

// ─────────────────────────────────────────────────────────────────────────────
// Groups
// ─────────────────────────────────────────────────────────────────────────────

// Group creates a route group with a shared URL prefix.
// Optional middleware applies only to routes in this group.
func (r *Router) Group(prefix string, mw ...MiddlewareFunc) *RouterGroup {
	gg := r.engine.Group(prefix)
	for _, m := range mw {
		m := m
		gg.Use(func(gc *gin.Context) { m(newGinContext(gc)) })
	}
	return &RouterGroup{group: gg}
}

// ─────────────────────────────────────────────────────────────────────────────
// Server
// ─────────────────────────────────────────────────────────────────────────────

// Handler returns the underlying http.Handler.
func (r *Router) Handler() http.Handler {
	return r.engine
}

// Run starts the Gin server on addr (blocks until stopped).
func (r *Router) Run(addr string) error {
	gin.SetMode(gin.ReleaseMode)
	return r.engine.Run(addr)
}

// Spec returns the accumulated OpenAPI spec for swagger generation.
func (r *Router) Spec() *OpenAPISpec {
	return r.spec
}

// ─────────────────────────────────────────────────────────────────────────────
// RouterGroup
// ─────────────────────────────────────────────────────────────────────────────

// RouterGroup wraps a *gin.RouterGroup.
type RouterGroup struct {
	group *gin.RouterGroup
}

func (g *RouterGroup) GET(path string, handler HandlerFunc) {
	g.group.GET(path, wrap(handler))
}
func (g *RouterGroup) POST(path string, handler HandlerFunc) {
	g.group.POST(path, wrap(handler))
}
func (g *RouterGroup) PUT(path string, handler HandlerFunc) {
	g.group.PUT(path, wrap(handler))
}
func (g *RouterGroup) PATCH(path string, handler HandlerFunc) {
	g.group.PATCH(path, wrap(handler))
}
func (g *RouterGroup) DELETE(path string, handler HandlerFunc) {
	g.group.DELETE(path, wrap(handler))
}
func (g *RouterGroup) OPTIONS(path string, handler HandlerFunc) {
	g.group.OPTIONS(path, wrap(handler))
}

// Use adds middleware scoped to this group.
func (g *RouterGroup) Use(mw ...MiddlewareFunc) {
	for _, m := range mw {
		m := m
		g.group.Use(func(gc *gin.Context) { m(newGinContext(gc)) })
	}
}

// Group creates a nested sub-group.
func (g *RouterGroup) Group(prefix string, mw ...MiddlewareFunc) *RouterGroup {
	gg := g.group.Group(prefix)
	for _, m := range mw {
		m := m
		gg.Use(func(gc *gin.Context) { m(newGinContext(gc)) })
	}
	return &RouterGroup{group: gg}
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal — gin adapter
// ─────────────────────────────────────────────────────────────────────────────

// wrap converts a HandlerFunc into a gin.HandlerFunc.
func wrap(h HandlerFunc) gin.HandlerFunc {
	return func(gc *gin.Context) {
		h(newGinContext(gc))
	}
}
