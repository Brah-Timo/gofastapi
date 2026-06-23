package naming_test

import (
	"testing"

	"github.com/Brah-Timo/gofastapi/internal/naming"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"User", "user"},
		{"BlogPost", "blog_post"},
		{"UserID", "user_id"},
		{"HTMLParser", "html_parser"},
		{"ProductOrder", "product_order"},
		{"createdAt", "created_at"},
	}
	for _, tt := range tests {
		got := naming.ToSnakeCase(tt.in)
		if got != tt.want {
			t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"User", "user"},
		{"BlogPost", "blog-post"},
		{"UserID", "user-id"},
		{"ProductOrder", "product-order"},
	}
	for _, tt := range tests {
		got := naming.ToKebabCase(tt.in)
		if got != tt.want {
			t.Errorf("ToKebabCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"user", "users"},
		{"category", "categories"},
		{"person", "people"},
		{"leaf", "leaves"},
		{"analysis", "analyses"},
		{"series", "series"},
		{"status", "statuses"},
	}
	for _, tt := range tests {
		got := naming.Pluralize(tt.in)
		if got != tt.want {
			t.Errorf("Pluralize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRoutePrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"User", "/users"},
		{"BlogPost", "/blog-posts"},
		{"ProductOrder", "/product-orders"},
		{"Category", "/categories"},
	}
	for _, tt := range tests {
		got := naming.RoutePrefix(tt.in)
		if got != tt.want {
			t.Errorf("RoutePrefix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTableName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"User", "users"},
		{"BlogPost", "blog_posts"},
		{"ProductOrder", "product_orders"},
		{"Category", "categories"},
	}
	for _, tt := range tests {
		got := naming.TableName(tt.in)
		if got != tt.want {
			t.Errorf("TableName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
