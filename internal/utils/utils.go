// Package utils provides miscellaneous internal helpers for gofastapi.
// Nothing in this package is part of the public API.
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ─────────────────────────────────────────────────────────────────────────────
// Environment helpers
// ─────────────────────────────────────────────────────────────────────────────

// Env returns the value of the environment variable named by key.
// If the variable is not set or empty it returns fallback.
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// EnvInt returns the integer value of an environment variable.
// Returns fallback if the variable is absent or cannot be parsed.
func EnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// EnvBool returns the boolean value of an environment variable.
// Truthy values: "1", "true", "yes", "on" (case-insensitive).
func EnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}

// ─────────────────────────────────────────────────────────────────────────────
// String helpers
// ─────────────────────────────────────────────────────────────────────────────

// Truncate returns s truncated to maxLen runes. If s is longer than maxLen
// a unicode-safe ellipsis ("…") replaces the last character.
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 0 {
		return ""
	}
	return string(runes[:maxLen-1]) + "…"
}

// Slugify converts a human-readable string to a URL-safe slug.
// "Hello, World!" → "hello-world"
func Slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		} else if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// Coalesce returns the first non-empty string from the provided list.
func Coalesce(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Numeric / pagination helpers
// ─────────────────────────────────────────────────────────────────────────────

// Clamp returns v clamped to [min, max].
func Clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// PageOffset computes the SQL OFFSET for a given 1-based page and page size.
func PageOffset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * pageSize
}

// TotalPages computes the number of pages needed to show total items
// at pageSize per page.
func TotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(pageSize)))
}

// ─────────────────────────────────────────────────────────────────────────────
// Crypto / ID helpers
// ─────────────────────────────────────────────────────────────────────────────

// RandomHex returns a cryptographically random hex string of length n bytes
// (resulting string has 2n characters).
func RandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("RandomHex: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// MustRandomHex is like RandomHex but panics on error.
func MustRandomHex(n int) string {
	s, err := RandomHex(n)
	if err != nil {
		panic(err)
	}
	return s
}

// ─────────────────────────────────────────────────────────────────────────────
// Time helpers
// ─────────────────────────────────────────────────────────────────────────────

// NowUTC returns the current UTC time with nanosecond precision.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// FormatRFC3339 returns t formatted as RFC 3339 (ISO 8601 with timezone).
func FormatRFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ParseRFC3339 parses a RFC 3339 formatted string.
func ParseRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// ─────────────────────────────────────────────────────────────────────────────
// Map / slice helpers
// ─────────────────────────────────────────────────────────────────────────────

// ContainsString reports whether slice contains s (case-sensitive).
func ContainsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// UniqueStrings returns a deduplicated copy of slice, preserving order.
func UniqueStrings(slice []string) []string {
	seen := make(map[string]struct{}, len(slice))
	out := make([]string, 0, len(slice))
	for _, s := range slice {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// MapKeys returns the keys of a string-keyed map in an unspecified order.
func MapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Ptr returns a pointer to v. Useful for optional struct fields.
func Ptr[T any](v T) *T { return &v }

// Deref dereferences p; returns zero value of T if p is nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
