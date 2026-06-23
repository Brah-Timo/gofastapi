// Package crud — error sentinel values and helpers.
package crud

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// Sentinel errors returned by Handlers
// ─────────────────────────────────────────────────────────────────────────────

var (
	// ErrInvalidID is returned when the URL :id parameter cannot be parsed.
	ErrInvalidID = errors.New("invalid or missing ID parameter")

	// ErrNotFound is returned when the requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrInvalidBody is returned when the request body cannot be bound.
	ErrInvalidBody = errors.New("request body is invalid or malformed JSON")

	// ErrValidationFailed is returned when struct validation fails.
	ErrValidationFailed = errors.New("validation failed")

	// ErrForbidden is returned by hooks or middleware to deny an action.
	ErrForbidden = errors.New("action is not allowed")

	// ErrConflict is returned when a uniqueness constraint is violated.
	ErrConflict = errors.New("resource already exists (conflict)")

	// ErrUnprocessable is a generic 422 error.
	ErrUnprocessable = errors.New("unprocessable entity")
)

// ─────────────────────────────────────────────────────────────────────────────
// GORM error classification
// ─────────────────────────────────────────────────────────────────────────────

// IsNotFound reports whether err is a GORM "record not found" error.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// IsDuplicateKey reports whether err is a database unique-constraint violation.
// Works across PostgreSQL, MySQL, and SQLite.
func IsDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique_violation") ||
		strings.Contains(msg, "1062") // MySQL error code
}

// ClassifyDBError maps a raw database error to a gofastapi sentinel.
// Returns the original error if it doesn't match a known pattern.
func ClassifyDBError(err error) error {
	if err == nil {
		return nil
	}
	if IsNotFound(err) {
		return ErrNotFound
	}
	if IsDuplicateKey(err) {
		return ErrConflict
	}
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTPStatus maps sentinel errors to HTTP status codes
// ─────────────────────────────────────────────────────────────────────────────

// HTTPStatus returns the appropriate HTTP status code for err.
func HTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return 404
	case errors.Is(err, ErrInvalidID),
		errors.Is(err, ErrInvalidBody):
		return 400
	case errors.Is(err, ErrValidationFailed),
		errors.Is(err, ErrUnprocessable):
		return 422
	case errors.Is(err, ErrForbidden):
		return 403
	case errors.Is(err, ErrConflict):
		return 409
	default:
		return 500
	}
}
