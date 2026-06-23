package utils_test

import (
	"testing"

	"github.com/Brah-Timo/gofastapi/internal/utils"
)

func TestTruncate(t *testing.T) {
	if got := utils.Truncate("Hello, World!", 5); got != "Hell…" {
		t.Errorf("got %q", got)
	}
	if got := utils.Truncate("Hi", 10); got != "Hi" {
		t.Errorf("got %q", got)
	}
	if got := utils.Truncate("Hi", 0); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Hello, World!", "hello-world"},
		{"  spaces  ", "spaces"},
		{"Über Café", "ber-caf"},
		{"already-slug", "already-slug"},
	}
	for _, tt := range tests {
		got := utils.Slugify(tt.in)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestClamp(t *testing.T) {
	if utils.Clamp(5, 1, 10) != 5 {
		t.Error("in range")
	}
	if utils.Clamp(0, 1, 10) != 1 {
		t.Error("below min")
	}
	if utils.Clamp(20, 1, 10) != 10 {
		t.Error("above max")
	}
}

func TestPageOffset(t *testing.T) {
	if utils.PageOffset(1, 20) != 0 {
		t.Error("page 1 offset should be 0")
	}
	if utils.PageOffset(3, 20) != 40 {
		t.Error("page 3 offset should be 40")
	}
	if utils.PageOffset(0, 20) != 0 {
		t.Error("page 0 should be treated as page 1")
	}
}

func TestTotalPages(t *testing.T) {
	if utils.TotalPages(100, 20) != 5 {
		t.Error("100 items / 20 per page = 5 pages")
	}
	if utils.TotalPages(101, 20) != 6 {
		t.Error("101 items / 20 per page = 6 pages (ceil)")
	}
	if utils.TotalPages(0, 20) != 0 {
		t.Error("0 items = 0 pages")
	}
}

func TestCoalesce(t *testing.T) {
	if utils.Coalesce("", "", "hello") != "hello" {
		t.Error("should return first non-empty")
	}
	if utils.Coalesce("", "") != "" {
		t.Error("all empty should return empty")
	}
}

func TestContainsString(t *testing.T) {
	sl := []string{"a", "b", "c"}
	if !utils.ContainsString(sl, "b") {
		t.Error("should contain b")
	}
	if utils.ContainsString(sl, "z") {
		t.Error("should not contain z")
	}
}

func TestUniqueStrings(t *testing.T) {
	got := utils.UniqueStrings([]string{"a", "b", "a", "c", "b"})
	if len(got) != 3 {
		t.Errorf("expected 3 unique strings, got %d: %v", len(got), got)
	}
}

func TestPtr(t *testing.T) {
	n := 42
	p := utils.Ptr(n)
	if *p != n {
		t.Error("Ptr should return pointer to value")
	}
}

func TestDeref(t *testing.T) {
	n := 42
	if utils.Deref(&n) != 42 {
		t.Error("Deref should return value")
	}
	var p *int
	if utils.Deref(p) != 0 {
		t.Error("Deref of nil should return zero value")
	}
}

func TestRandomHex(t *testing.T) {
	s, err := utils.RandomHex(16)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 32 {
		t.Errorf("expected 32 hex chars, got %d", len(s))
	}
	s2, _ := utils.RandomHex(16)
	if s == s2 {
		t.Error("two random hex strings should differ")
	}
}
