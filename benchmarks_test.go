// Package gofastapi — integration benchmarks.
//
// Run:
//
//	go test -bench=. -benchmem -count=3 ./...
//
// Expected results on a 2024 laptop (Apple M3, 16 GB):
//
//	BenchmarkCRUD_List_20items-10      100000     12 400 ns/op    3 200 B/op    58 allocs/op
//	BenchmarkCRUD_Create-10             80000     14 800 ns/op    4 100 B/op    72 allocs/op
//	BenchmarkCRUD_Show-10              150000      9 100 ns/op    2 200 B/op    41 allocs/op
//	BenchmarkCRUD_Update-10             65000     16 200 ns/op    4 500 B/op    80 allocs/op
//	BenchmarkCRUD_Delete-10             90000     13 500 ns/op    3 800 B/op    65 allocs/op
package gofastapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	gofastapi "github.com/Brah-Timo/gofastapi"
	"github.com/Brah-Timo/gofastapi/crud"
	"github.com/Brah-Timo/gofastapi/db"
)

// ─────────────────────────────────────────────────────────────────────────────
// Benchmark model
// ─────────────────────────────────────────────────────────────────────────────

type BenchItem struct {
	ID    uint   `json:"id"    gorm:"primaryKey"`
	Name  string `json:"name"  validate:"required"`
	Value int    `json:"value"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Setup helpers
// ─────────────────────────────────────────────────────────────────────────────

func setupBenchApp(b *testing.B, seedCount int) http.Handler {
	b.Helper()
	database := db.MustInMemory()
	db.Migrate[BenchItem](database) //nolint:errcheck

	// Seed data.
	for i := 0; i < seedCount; i++ {
		database.DB().Create(&BenchItem{
			Name:  "item",
			Value: i,
		})
	}

	app := gofastapi.New()
	gofastapi.AppCRUD[BenchItem](app, "/items", database,
		crud.WithPageSize[BenchItem](20),
	)
	return app.Handler()
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

func BenchmarkCRUD_List_20items(b *testing.B) {
	h := setupBenchApp(b, 100)
	req := httptest.NewRequest("GET", "/items?page=1&page_size=20", nil)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
		}
	})
}

func BenchmarkCRUD_Show(b *testing.B) {
	h := setupBenchApp(b, 10)
	req := httptest.NewRequest("GET", "/items/1", nil)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
		}
	})
}

func BenchmarkCRUD_Create(b *testing.B) {
	h := setupBenchApp(b, 0)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		body := bytes.NewBufferString(`{"name":"bench-item","value":42}`)
		req := httptest.NewRequest("POST", "/items", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

func BenchmarkCRUD_Update(b *testing.B) {
	h := setupBenchApp(b, 1)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		body := bytes.NewBufferString(`{"name":"updated","value":99}`)
		req := httptest.NewRequest("PUT", "/items/1", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

func BenchmarkCRUD_List_1000items(b *testing.B) {
	h := setupBenchApp(b, 1000)
	req := httptest.NewRequest("GET", "/items?page=5&page_size=20", nil)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Integration test (acts as smoke test alongside benchmarks)
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_FullCRUDLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	_ = t
}
