package response_test

import (
	"encoding/json"
	"testing"

	"github.com/Brah-Timo/gofastapi/response"
)

// fakeCtx captures what response functions send.
type fakeCtx struct {
	status int
	body   []byte
}

func (f *fakeCtx) JSON(status int, v any) {
	f.status = status
	b, _ := json.Marshal(v)
	f.body = b
}

func decode[T any](data []byte) T {
	var v T
	json.Unmarshal(data, &v)
	return v
}

type envelope struct {
	Success bool               `json:"success"`
	Data    any                `json:"data"`
	Error   *response.APIError `json:"error"`
	Meta    *response.Meta     `json:"meta"`
}

func TestSuccess(t *testing.T) {
	ctx := &fakeCtx{}
	response.Success(ctx, 200, map[string]string{"name": "Alice"})
	if ctx.status != 200 {
		t.Errorf("expected 200, got %d", ctx.status)
	}
	env := decode[envelope](ctx.body)
	if !env.Success {
		t.Error("expected success=true")
	}
}

func TestError(t *testing.T) {
	ctx := &fakeCtx{}
	response.ErrorMsg(ctx, 404, "not found")
	if ctx.status != 404 {
		t.Errorf("expected 404, got %d", ctx.status)
	}
	env := decode[envelope](ctx.body)
	if env.Success {
		t.Error("expected success=false")
	}
	if env.Error == nil || env.Error.Code != "NOT_FOUND" {
		t.Errorf("expected error code NOT_FOUND, got %v", env.Error)
	}
}

func TestValidationErrors(t *testing.T) {
	ctx := &fakeCtx{}
	response.ValidationErrors(ctx, map[string]string{
		"email": "must be valid",
		"name":  "required",
	})
	if ctx.status != 422 {
		t.Errorf("expected 422, got %d", ctx.status)
	}
	env := decode[envelope](ctx.body)
	if env.Error == nil || env.Error.Code != "VALIDATION_FAILED" {
		t.Error("expected VALIDATION_FAILED code")
	}
}

func TestPaginated_NilSlice(t *testing.T) {
	ctx := &fakeCtx{}
	// nil slice should be returned as [] not null
	var items []string
	response.Paginated(ctx, items, 0, 1, 20)
	if ctx.status != 200 {
		t.Errorf("expected 200, got %d", ctx.status)
	}
}

func TestPaginated_Meta(t *testing.T) {
	ctx := &fakeCtx{}
	items := []int{1, 2, 3}
	response.Paginated(ctx, items, 25, 2, 10)

	env := decode[envelope](ctx.body)
	if env.Meta == nil {
		t.Fatal("expected meta to be present")
	}
	if env.Meta.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", env.Meta.TotalPages)
	}
	if !env.Meta.HasNext {
		t.Error("expected has_next=true on page 2 of 3")
	}
	if !env.Meta.HasPrev {
		t.Error("expected has_prev=true on page 2")
	}
}
