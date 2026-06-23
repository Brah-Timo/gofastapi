package crud_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Brah-Timo/gofastapi/crud"
	"github.com/Brah-Timo/gofastapi/db"
	"github.com/Brah-Timo/gofastapi/hooks"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test model
// ─────────────────────────────────────────────────────────────────────────────

type Article struct {
	ID    uint   `json:"id"    gorm:"primaryKey"`
	Title string `json:"title" validate:"required,min=3"`
	Body  string `json:"body"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Fake Context for unit tests
// ─────────────────────────────────────────────────────────────────────────────

type fakeCtx struct {
	req     *http.Request
	body    []byte
	params  map[string]string
	queries map[string]string
	store   map[string]any
	status  int
	resp    any
	aborted bool
}

func newFakeCtx(method, body string, params, queries map[string]string) *fakeCtx {
	req := httptest.NewRequest(method, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return &fakeCtx{
		req:     req,
		body:    []byte(body),
		params:  params,
		queries: queries,
		store:   make(map[string]any),
	}
}

func (f *fakeCtx) Request() *http.Request { return f.req }
func (f *fakeCtx) BindJSON(v any) error   { return json.Unmarshal(f.body, v) }
func (f *fakeCtx) JSON(status int, v any) { f.status = status; f.resp = v }
func (f *fakeCtx) Param(name string) string {
	if f.params != nil {
		return f.params[name]
	}
	return ""
}
func (f *fakeCtx) Query(name string) string {
	if f.queries != nil {
		return f.queries[name]
	}
	return ""
}
func (f *fakeCtx) QueryDefault(name, def string) string {
	if v := f.Query(name); v != "" {
		return v
	}
	return def
}
func (f *fakeCtx) Set(k string, v any)      { f.store[k] = v }
func (f *fakeCtx) Get(k string) (any, bool) { v, ok := f.store[k]; return v, ok }
func (f *fakeCtx) Abort()                   { f.aborted = true }
func (f *fakeCtx) Next()                    {}
func (f *fakeCtx) ClientIP() string         { return "127.0.0.1" }

// ─────────────────────────────────────────────────────────────────────────────
// Fake Repository for unit tests
// ─────────────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	items  map[uint]*Article
	nextID uint
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: make(map[uint]*Article), nextID: 1}
}

func (r *fakeRepo) List(_ context.Context, p crud.ListParams) ([]Article, int64, error) {
	out := make([]Article, 0, len(r.items))
	for _, a := range r.items {
		out = append(out, *a)
	}
	return out, int64(len(out)), nil
}
func (r *fakeRepo) FindByID(_ context.Context, id any) (Article, error) {
	uid := toUint(id)
	if a, ok := r.items[uid]; ok {
		return *a, nil
	}
	return Article{}, crud.ErrNotFound
}
func (r *fakeRepo) Create(_ context.Context, a *Article) error {
	a.ID = r.nextID
	r.nextID++
	r.items[a.ID] = a
	return nil
}
func (r *fakeRepo) Update(_ context.Context, a *Article) error {
	r.items[a.ID] = a
	return nil
}
func (r *fakeRepo) Delete(_ context.Context, id any) error {
	uid := toUint(id)
	delete(r.items, uid)
	return nil
}
func (r *fakeRepo) SoftDelete(_ context.Context, id any) error { return r.Delete(nil, id) }
func (r *fakeRepo) Count(_ context.Context, _ crud.ListParams) (int64, error) {
	return int64(len(r.items)), nil
}

func toUint(id any) uint {
	switch v := id.(type) {
	case uint64:
		return uint(v)
	case uint:
		return v
	}
	return 0
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

func newTestHandler() *crud.Handler[Article] {
	// Use a null DB — we override the repo.
	d := db.MustInMemory()
	h := crud.NewHandler[Article](d,
		crud.WithRepository[Article](newFakeRepo()),
	)
	return h
}

func TestHandler_Create_Success(t *testing.T) {
	h := newTestHandler()
	ctx := newFakeCtx("POST", `{"title":"Hello World","body":"Some body"}`, nil, nil)
	h.Create(ctx)
	if ctx.status != http.StatusCreated {
		t.Errorf("expected 201, got %d", ctx.status)
	}
}

func TestHandler_Create_ValidationError(t *testing.T) {
	h := newTestHandler()
	ctx := newFakeCtx("POST", `{"title":"ab"}`, nil, nil) // min=3 → "ab" is 2 chars
	h.Create(ctx)
	if ctx.status != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", ctx.status)
	}
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	h := newTestHandler()
	ctx := newFakeCtx("POST", `not-json`, nil, nil)
	h.Create(ctx)
	if ctx.status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", ctx.status)
	}
}

func TestHandler_Show_NotFound(t *testing.T) {
	h := newTestHandler()
	ctx := newFakeCtx("GET", "", map[string]string{"id": "999"}, nil)
	h.Show(ctx)
	if ctx.status != http.StatusNotFound {
		t.Errorf("expected 404, got %d", ctx.status)
	}
}

func TestHandler_Show_Found(t *testing.T) {
	repo := newFakeRepo()
	a := &Article{Title: "Test", Body: "Body"}
	repo.Create(context.Background(), a)

	d := db.MustInMemory()
	h := crud.NewHandler[Article](d, crud.WithRepository[Article](repo))
	ctx := newFakeCtx("GET", "", map[string]string{"id": "1"}, nil)
	h.Show(ctx)
	if ctx.status != http.StatusOK {
		t.Errorf("expected 200, got %d", ctx.status)
	}
}

func TestHandler_Update_Success(t *testing.T) {
	repo := newFakeRepo()
	a := &Article{Title: "Old Title", Body: "Old Body"}
	repo.Create(context.Background(), a)

	d := db.MustInMemory()
	h := crud.NewHandler[Article](d, crud.WithRepository[Article](repo))
	ctx := newFakeCtx("PUT", `{"title":"New Title","body":"New Body"}`, map[string]string{"id": "1"}, nil)
	h.Update(ctx)
	if ctx.status != http.StatusOK {
		t.Errorf("expected 200, got %d", ctx.status)
	}
}

func TestHandler_Delete_Success(t *testing.T) {
	repo := newFakeRepo()
	a := &Article{Title: "To Delete", Body: "body"}
	repo.Create(context.Background(), a)

	d := db.MustInMemory()
	h := crud.NewHandler[Article](d, crud.WithRepository[Article](repo))
	ctx := newFakeCtx("DELETE", "", map[string]string{"id": "1"}, nil)
	h.Delete(ctx)
	if ctx.status != http.StatusOK {
		t.Errorf("expected 200, got %d", ctx.status)
	}
}

func TestHandler_Delete_NotFound(t *testing.T) {
	h := newTestHandler()
	ctx := newFakeCtx("DELETE", "", map[string]string{"id": "999"}, nil)
	h.Delete(ctx)
	if ctx.status != http.StatusNotFound {
		t.Errorf("expected 404, got %d", ctx.status)
	}
}

func TestHandler_BeforeCreate_Hook(t *testing.T) {
	called := false
	repo := newFakeRepo()
	d := db.MustInMemory()
	h := crud.NewHandler[Article](d,
		crud.WithRepository[Article](repo),
		crud.WithBeforeCreate[Article](func(a *Article, ctx hooks.Context) error {
			called = true
			a.Body = "hook-modified"
			return nil
		}),
	)
	ctx := newFakeCtx("POST", `{"title":"Hook Test","body":"original"}`, nil, nil)
	h.Create(ctx)
	if !called {
		t.Error("BeforeCreate hook was not called")
	}
	if ctx.status != http.StatusCreated {
		t.Errorf("expected 201, got %d", ctx.status)
	}
}

func TestHandler_AfterCreate_Hook(t *testing.T) {
	calledWith := ""
	repo := newFakeRepo()
	d := db.MustInMemory()
	h := crud.NewHandler[Article](d,
		crud.WithRepository[Article](repo),
		crud.WithAfterCreate[Article](func(a *Article, ctx hooks.Context) error {
			calledWith = a.Title
			return nil
		}),
	)
	ctx := newFakeCtx("POST", `{"title":"After Hook","body":"b"}`, nil, nil)
	h.Create(ctx)
	if calledWith != "After Hook" {
		t.Errorf("AfterCreate hook received wrong title: %q", calledWith)
	}
}

func TestHandler_List(t *testing.T) {
	repo := newFakeRepo()
	for i := 0; i < 5; i++ {
		a := &Article{Title: "Article", Body: "body"}
		repo.Create(context.Background(), a)
	}
	d := db.MustInMemory()
	h := crud.NewHandler[Article](d, crud.WithRepository[Article](repo))
	ctx := newFakeCtx("GET", "", nil, map[string]string{"page": "1", "page_size": "10"})
	h.List(ctx)
	if ctx.status != http.StatusOK {
		t.Errorf("expected 200, got %d", ctx.status)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Hook type aliases exposed by crud package
// ─────────────────────────────────────────────────────────────────────────────

func TestHookConstants(t *testing.T) {
	if crud.HookBeforeCreate != hooks.BeforeCreate {
		t.Error("HookBeforeCreate mismatch")
	}
	if crud.HookAfterCreate != hooks.AfterCreate {
		t.Error("HookAfterCreate mismatch")
	}
}
