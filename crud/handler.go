// Package crud — generic HTTP handler.
//
// Handler[T] converts HTTP requests into CRUD repository calls, applying
// validation and hooks at each step. It implements five methods that map
// directly to the five standard REST verbs.
package crud

import (
	"net/http"
	"strconv"

	"github.com/Brah-Timo/gofastapi/db"
	"github.com/Brah-Timo/gofastapi/hooks"
	"github.com/Brah-Timo/gofastapi/response"
	"github.com/Brah-Timo/gofastapi/validation"
)

// ─────────────────────────────────────────────────────────────────────────────
// Context — the thin HTTP context interface Handler depends on
// ─────────────────────────────────────────────────────────────────────────────

// Context is the minimal HTTP context interface required by a Handler.
// The router package (gin-backed) provides a concrete implementation.
type Context interface {
	// Request returns the underlying *http.Request.
	Request() *http.Request
	// BindJSON decodes the request body as JSON into v.
	BindJSON(v any) error
	// JSON sends a JSON response with the given status code.
	JSON(status int, v any)
	// Param returns the URL path parameter with name.
	Param(name string) string
	// Query returns a query string parameter.
	Query(name string) string
	// QueryDefault returns a query string parameter or defaultValue if absent.
	QueryDefault(name, defaultValue string) string
	// Set stores a key-value pair in the context for the duration of the request.
	Set(key string, value any)
	// Get retrieves a value previously stored with Set.
	Get(key string) (any, bool)
	// Abort stops the middleware chain and prevents further handlers from running.
	Abort()
	// Next calls the next handler in the middleware chain.
	Next()
	// ClientIP returns the client IP address.
	ClientIP() string
}

// MiddlewareFunc is a function that wraps a Context.
type MiddlewareFunc func(ctx Context)

// ─────────────────────────────────────────────────────────────────────────────
// HandlerConfig — all tunable knobs
// ─────────────────────────────────────────────────────────────────────────────

// HandlerConfig holds the configuration for a Handler instance.
// All fields have sane defaults and can be overridden with Option functions.
type HandlerConfig struct {
	// DefaultPageSize is the page size used when the client does not specify one.
	DefaultPageSize int
	// MaxPageSize caps the page size a client may request.
	MaxPageSize int
	// EnableSoftDelete uses GORM soft-delete semantics (sets deleted_at).
	EnableSoftDelete bool
	// SearchFields lists the column names the free-text search applies to.
	// When empty, free-text search is disabled.
	SearchFields []string
	// AllowedOrderFields whitelists columns usable in ORDER BY.
	// When empty, any column name without special characters is allowed.
	AllowedOrderFields []string
	// SelectFields restricts which columns are returned in List / Show.
	SelectFields []string
	// Preloads lists associations to eagerly load.
	Preloads []string
}

func defaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		DefaultPageSize:  20,
		MaxPageSize:      100,
		EnableSoftDelete: false,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Handler[T] — the generic HTTP handler
// ─────────────────────────────────────────────────────────────────────────────

// Handler[T] holds the state needed to serve all five CRUD endpoints for T.
type Handler[T any] struct {
	repo        Repository[T]
	validator   *validation.Validator
	hooksReg    *hooks.Registry[T]
	cfg         HandlerConfig
	middlewares []MiddlewareFunc
}

// NewHandler creates a Handler[T] with the default GORM repository.
// Pass Option[T] functions to customise behaviour.
func NewHandler[T any](database db.Database, opts ...Option[T]) *Handler[T] {
	cfg := defaultHandlerConfig()
	repo := NewGORMRepository[T](database)
	repo.searchFields = cfg.SearchFields
	repo.allowedOrder = cfg.AllowedOrderFields

	h := &Handler[T]{
		repo:      repo,
		validator: validation.New(),
		hooksReg:  hooks.NewRegistry[T](),
		cfg:       cfg,
	}
	for _, o := range opts {
		o(h)
	}
	// Propagate search/order config to repository.
	if gr, ok := h.repo.(*GORMRepository[T]); ok {
		gr.searchFields = h.cfg.SearchFields
		gr.allowedOrder = h.cfg.AllowedOrderFields
	}
	return h
}

// ─────────────────────────────────────────────────────────────────────────────
// LIST — GET /prefix
// ─────────────────────────────────────────────────────────────────────────────

// List handles GET /prefix
// Query params: page, page_size, search, order_by, order_dir, + any filters.
func (h *Handler[T]) List(ctx Context) {
	p := h.buildListParams(ctx)

	items, total, err := h.repo.List(ctx.Request().Context(), p)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err)
		return
	}

	response.Paginated(ctx, items, total, p.Page, p.PageSize)
}

// ─────────────────────────────────────────────────────────────────────────────
// SHOW — GET /prefix/:id
// ─────────────────────────────────────────────────────────────────────────────

// Show handles GET /prefix/:id
func (h *Handler[T]) Show(ctx Context) {
	id, ok := parseID(ctx.Param("id"))
	if !ok {
		response.Error(ctx, http.StatusBadRequest, ErrInvalidID)
		return
	}

	item, err := h.repo.FindByID(ctx.Request().Context(), id)
	if err != nil {
		response.Error(ctx, HTTPStatus(err), err)
		return
	}

	// AfterFind hook.
	_ = h.hooksReg.Run(hooks.AfterFind, &item, ctx)

	response.Success(ctx, http.StatusOK, item)
}

// ─────────────────────────────────────────────────────────────────────────────
// CREATE — POST /prefix
// ─────────────────────────────────────────────────────────────────────────────

// Create handles POST /prefix
func (h *Handler[T]) Create(ctx Context) {
	var item T

	if err := ctx.BindJSON(&item); err != nil {
		response.Error(ctx, http.StatusBadRequest, ErrInvalidBody)
		return
	}

	if errs := h.validator.Validate(item); len(errs) > 0 {
		response.ValidationErrors(ctx, errs)
		return
	}

	if err := h.hooksReg.Run(hooks.BeforeCreate, &item, ctx); err != nil {
		response.Error(ctx, HTTPStatus(err), err)
		return
	}

	if err := h.repo.Create(ctx.Request().Context(), &item); err != nil {
		response.Error(ctx, HTTPStatus(err), err)
		return
	}

	_ = h.hooksReg.Run(hooks.AfterCreate, &item, ctx)

	response.Success(ctx, http.StatusCreated, item)
}

// ─────────────────────────────────────────────────────────────────────────────
// UPDATE — PUT /prefix/:id
// ─────────────────────────────────────────────────────────────────────────────

// Update handles PUT /prefix/:id
func (h *Handler[T]) Update(ctx Context) {
	id, ok := parseID(ctx.Param("id"))
	if !ok {
		response.Error(ctx, http.StatusBadRequest, ErrInvalidID)
		return
	}

	existing, err := h.repo.FindByID(ctx.Request().Context(), id)
	if err != nil {
		response.Error(ctx, HTTPStatus(err), err)
		return
	}

	// Merge request body onto the existing record.
	// This preserves fields that the client did not include.
	if err := ctx.BindJSON(&existing); err != nil {
		response.Error(ctx, http.StatusBadRequest, ErrInvalidBody)
		return
	}

	if errs := h.validator.Validate(existing); len(errs) > 0 {
		response.ValidationErrors(ctx, errs)
		return
	}

	if err := h.hooksReg.Run(hooks.BeforeUpdate, &existing, ctx); err != nil {
		response.Error(ctx, HTTPStatus(err), err)
		return
	}

	if err := h.repo.Update(ctx.Request().Context(), &existing); err != nil {
		response.Error(ctx, HTTPStatus(err), err)
		return
	}

	_ = h.hooksReg.Run(hooks.AfterUpdate, &existing, ctx)

	response.Success(ctx, http.StatusOK, existing)
}

// ─────────────────────────────────────────────────────────────────────────────
// DELETE — DELETE /prefix/:id
// ─────────────────────────────────────────────────────────────────────────────

// Delete handles DELETE /prefix/:id
func (h *Handler[T]) Delete(ctx Context) {
	id, ok := parseID(ctx.Param("id"))
	if !ok {
		response.Error(ctx, http.StatusBadRequest, ErrInvalidID)
		return
	}

	item, err := h.repo.FindByID(ctx.Request().Context(), id)
	if err != nil {
		response.Error(ctx, HTTPStatus(err), err)
		return
	}

	if err := h.hooksReg.Run(hooks.BeforeDelete, &item, ctx); err != nil {
		response.Error(ctx, HTTPStatus(err), err)
		return
	}

	var delErr error
	if h.cfg.EnableSoftDelete {
		delErr = h.repo.SoftDelete(ctx.Request().Context(), id)
	} else {
		delErr = h.repo.Delete(ctx.Request().Context(), id)
	}
	if delErr != nil {
		response.Error(ctx, http.StatusInternalServerError, delErr)
		return
	}

	_ = h.hooksReg.Run(hooks.AfterDelete, &item, ctx)

	response.Success(ctx, http.StatusOK, map[string]string{
		"message": "deleted successfully",
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// buildListParams extracts and validates pagination / filter parameters from
// the request context.
func (h *Handler[T]) buildListParams(ctx Context) ListParams {
	page := queryInt(ctx, "page", 1)
	pageSize := queryInt(ctx, "page_size", h.cfg.DefaultPageSize)

	if pageSize > h.cfg.MaxPageSize {
		pageSize = h.cfg.MaxPageSize
	}
	if pageSize < 1 {
		pageSize = h.cfg.DefaultPageSize
	}
	if page < 1 {
		page = 1
	}

	return ListParams{
		Page:         page,
		PageSize:     pageSize,
		Search:       ctx.Query("search"),
		OrderBy:      ctx.Query("order_by"),
		OrderDir:     ctx.QueryDefault("order_dir", "asc"),
		SelectFields: h.cfg.SelectFields,
		Preloads:     h.cfg.Preloads,
	}
}

// parseID converts the :id URL parameter to a typed value.
// Accepts uint, int, and string UUIDs.
func parseID(raw string) (any, bool) {
	if raw == "" {
		return nil, false
	}
	// Try integer first.
	if n, err := strconv.ParseUint(raw, 10, 64); err == nil {
		return n, true
	}
	// Fall back to string (UUID, slug, …).
	if len(raw) > 0 {
		return raw, true
	}
	return nil, false
}

// queryInt returns the integer value of a query parameter or def.
func queryInt(ctx Context, name string, def int) int {
	raw := ctx.Query(name)
	if raw == "" {
		return def
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return n
	}
	return def
}
