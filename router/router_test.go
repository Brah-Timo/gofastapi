package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Brah-Timo/gofastapi/router"
)

func TestRouter_GET(t *testing.T) {
	r := router.New()
	r.GET("/ping", func(ctx *router.Context) {
		ctx.JSON(200, map[string]string{"pong": "ok"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRouter_Group(t *testing.T) {
	r := router.New()
	v1 := r.Group("/api/v1")
	v1.GET("/health", func(ctx *router.Context) {
		ctx.JSON(200, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRouter_Middleware(t *testing.T) {
	r := router.New()
	called := false
	r.Use(func(ctx *router.Context) {
		called = true
		ctx.Next()
	})
	r.GET("/test", func(ctx *router.Context) {
		ctx.JSON(200, nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	if !called {
		t.Error("middleware was not called")
	}
}

func TestRouter_NotFound(t *testing.T) {
	r := router.New()
	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRouter_Params(t *testing.T) {
	r := router.New()
	var gotID string
	r.GET("/users/:id", func(ctx *router.Context) {
		gotID = ctx.Param("id")
		ctx.JSON(200, nil)
	})

	req := httptest.NewRequest("GET", "/users/42", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	if gotID != "42" {
		t.Errorf("expected param id=42, got %q", gotID)
	}
}

func TestRouter_NestedGroup(t *testing.T) {
	r := router.New()
	v1 := r.Group("/api/v1")
	admin := v1.Group("/admin")
	admin.GET("/dashboard", func(ctx *router.Context) {
		ctx.JSON(200, nil)
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/dashboard", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
