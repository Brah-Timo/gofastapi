// Package response provides the standard JSON response shapes for gofastapi.
//
// Every API endpoint returns one of three shapes:
//
//  1. Success   → { "success": true, "data": <T> }
//  2. Paginated → { "success": true, "data": [<T>], "meta": { pagination info } }
//  3. Error     → { "success": false, "error": { "code": "…", "message": "…" } }
//
// This guarantees a consistent, predictable contract for API consumers.
package response

import (
	"math"
	"net/http"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// Context — minimal interface required to send a response
// ─────────────────────────────────────────────────────────────────────────────

// Context is the minimal interface response functions need from the router.
// It matches crud.Context so the two packages can stay loosely coupled.
type Context interface {
	JSON(status int, v any)
}

// ─────────────────────────────────────────────────────────────────────────────
// Response shapes
// ─────────────────────────────────────────────────────────────────────────────

// APIResponse is the envelope wrapping every API response.
// T is the data payload type.
type APIResponse[T any] struct {
	// Success indicates whether the operation succeeded.
	Success bool `json:"success"`
	// Data holds the response payload (absent on error).
	Data T `json:"data,omitempty"`
	// Error holds error details (absent on success).
	Error *APIError `json:"error,omitempty"`
	// Meta holds pagination metadata (absent on non-list responses).
	Meta *Meta `json:"meta,omitempty"`
	// RequestID is an optional correlation ID propagated from the request.
	RequestID string `json:"request_id,omitempty"`
}

// APIError carries structured error information.
type APIError struct {
	// Code is a machine-readable error code (e.g. "NOT_FOUND").
	Code string `json:"code"`
	// Message is a human-readable description of the error.
	Message string `json:"message"`
	// Details holds per-field validation errors.
	Details map[string]string `json:"details,omitempty"`
}

// Meta holds pagination information for list responses.
type Meta struct {
	// Page is the current 1-based page number.
	Page int `json:"page"`
	// PageSize is the number of items per page.
	PageSize int `json:"page_size"`
	// Total is the total number of records matching the query.
	Total int64 `json:"total"`
	// TotalPages is the total number of pages.
	TotalPages int `json:"total_pages"`
	// HasNext indicates whether a next page exists.
	HasNext bool `json:"has_next"`
	// HasPrev indicates whether a previous page exists.
	HasPrev bool `json:"has_prev"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Response helpers
// ─────────────────────────────────────────────────────────────────────────────

// Success sends a successful JSON response with the given HTTP status and data.
//
//	response.Success(ctx, http.StatusCreated, newUser)
func Success[T any](ctx Context, status int, data T) {
	ctx.JSON(status, APIResponse[T]{
		Success: true,
		Data:    data,
	})
}

// OK is a convenience wrapper for Success with status 200.
func OK[T any](ctx Context, data T) {
	Success(ctx, http.StatusOK, data)
}

// Created is a convenience wrapper for Success with status 201.
func Created[T any](ctx Context, data T) {
	Success(ctx, http.StatusCreated, data)
}

// Paginated sends a paginated list response.
//
//	response.Paginated(ctx, users, 250, 3, 20)
func Paginated[T any](ctx Context, items []T, total int64, page, pageSize int) {
	if items == nil {
		items = []T{} // Never return null for a list.
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pageSize <= 0 {
		totalPages = 0
	}

	ctx.JSON(http.StatusOK, APIResponse[[]T]{
		Success: true,
		Data:    items,
		Meta: &Meta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	})
}

// Error sends an error JSON response.
//
//	response.Error(ctx, http.StatusNotFound, err)
func Error(ctx Context, status int, err error) {
	msg := "internal server error"
	if err != nil {
		msg = err.Error()
	}
	ctx.JSON(status, APIResponse[any]{
		Success: false,
		Error: &APIError{
			Code:    statusToCode(status),
			Message: msg,
		},
	})
}

// ErrorMsg sends an error response with a plain string message.
func ErrorMsg(ctx Context, status int, msg string) {
	ctx.JSON(status, APIResponse[any]{
		Success: false,
		Error: &APIError{
			Code:    statusToCode(status),
			Message: msg,
		},
	})
}

// ValidationErrors sends a 422 response with per-field error details.
//
//	response.ValidationErrors(ctx, map[string]string{
//	    "email": "must be a valid email address",
//	    "name":  "is required",
//	})
func ValidationErrors(ctx Context, details map[string]string) {
	ctx.JSON(http.StatusUnprocessableEntity, APIResponse[any]{
		Success: false,
		Error: &APIError{
			Code:    "VALIDATION_FAILED",
			Message: "one or more fields failed validation",
			Details: details,
		},
	})
}

// NotFound sends a 404 response.
func NotFound(ctx Context, resource string) {
	ErrorMsg(ctx, http.StatusNotFound, resource+" not found")
}

// Forbidden sends a 403 response.
func Forbidden(ctx Context, reason string) {
	if reason == "" {
		reason = "you do not have permission to perform this action"
	}
	ErrorMsg(ctx, http.StatusForbidden, reason)
}

// Unauthorized sends a 401 response.
func Unauthorized(ctx Context, reason string) {
	if reason == "" {
		reason = "authentication required"
	}
	ErrorMsg(ctx, http.StatusUnauthorized, reason)
}

// Conflict sends a 409 response.
func Conflict(ctx Context, msg string) {
	if msg == "" {
		msg = "resource already exists"
	}
	ErrorMsg(ctx, http.StatusConflict, msg)
}

// NoContent sends a 204 No Content response (no body).
// Use for DELETE operations that don't return data.
type noContentContext interface {
	Status(int)
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// statusToCode maps HTTP status codes to machine-readable error code strings.
func statusToCode(status int) string {
	codes := map[int]string{
		400: "BAD_REQUEST",
		401: "UNAUTHORIZED",
		403: "FORBIDDEN",
		404: "NOT_FOUND",
		405: "METHOD_NOT_ALLOWED",
		409: "CONFLICT",
		410: "GONE",
		422: "VALIDATION_FAILED",
		429: "TOO_MANY_REQUESTS",
		500: "INTERNAL_SERVER_ERROR",
		502: "BAD_GATEWAY",
		503: "SERVICE_UNAVAILABLE",
		504: "GATEWAY_TIMEOUT",
	}
	if code, ok := codes[status]; ok {
		return code
	}
	return strings.ToUpper(http.StatusText(status))
}
