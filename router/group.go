// Package router — Gin context adapter + OpenAPI spec builder.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Context — wraps *gin.Context to implement crud.Context
// ─────────────────────────────────────────────────────────────────────────────

// Context wraps *gin.Context and satisfies the crud.Context interface.
// It is the concrete context type used everywhere in gofastapi.
type Context struct {
	gc *gin.Context
}

func newGinContext(gc *gin.Context) *Context {
	return &Context{gc: gc}
}

func (c *Context) Request() *http.Request { return c.gc.Request }

func (c *Context) BindJSON(v any) error {
	return c.gc.ShouldBindJSON(v)
}

func (c *Context) JSON(status int, v any) {
	if v == nil {
		c.gc.Status(status)
		return
	}
	c.gc.JSON(status, v)
}

func (c *Context) Param(name string) string {
	return c.gc.Param(name)
}

func (c *Context) Query(name string) string {
	return c.gc.Query(name)
}

func (c *Context) QueryDefault(name, defaultValue string) string {
	return c.gc.DefaultQuery(name, defaultValue)
}

func (c *Context) Set(key string, value any) {
	c.gc.Set(key, value)
}

func (c *Context) Get(key string) (any, bool) {
	return c.gc.Get(key)
}

func (c *Context) Abort() {
	c.gc.Abort()
}

func (c *Context) AbortWithStatus(status int) {
	c.gc.AbortWithStatus(status)
}

func (c *Context) Next() {
	c.gc.Next()
}

func (c *Context) ClientIP() string {
	return c.gc.ClientIP()
}

// Header sets a response header.
func (c *Context) Header(key, value string) {
	c.gc.Header(key, value)
}

// GetHeader returns a request header value.
func (c *Context) GetHeader(key string) string {
	return c.gc.GetHeader(key)
}

// Status sets the response status code without a body.
func (c *Context) Status(code int) {
	c.gc.Status(code)
}

// Gin returns the underlying *gin.Context for direct Gin API access.
// Use this only when you need functionality not exposed by the Context interface.
func (c *Context) Gin() *gin.Context {
	return c.gc
}

// ─────────────────────────────────────────────────────────────────────────────
// OpenAPISpec — collects route info for Swagger generation
// ─────────────────────────────────────────────────────────────────────────────

// OpenAPISpec holds a minimal OpenAPI 3.0 specification built at route
// registration time. It is consumed by the swagger package to render the UI.
type OpenAPISpec struct {
	Title       string
	Version     string
	Description string
	Routes      []RouteSpec
}

// RouteSpec describes a single API endpoint.
type RouteSpec struct {
	Method  string
	Path    string
	Tag     string
	Summary string
}

func newOpenAPISpec() *OpenAPISpec {
	return &OpenAPISpec{
		Title:   "gofastapi",
		Version: "1.0.0",
	}
}

// AddRoute registers a route in the spec.
func (s *OpenAPISpec) AddRoute(method, path, tag, summary string) {
	s.Routes = append(s.Routes, RouteSpec{
		Method:  method,
		Path:    path,
		Tag:     tag,
		Summary: summary,
	})
}
