// Example: Lifecycle hooks — BeforeCreate, AfterCreate, BeforeDelete.
//
// Demonstrates how to use hooks for:
//   - Auto-populating fields (slug generation)
//   - Sending notifications after record creation
//   - Preventing deletion of protected records
//
// Run:
//
//	go run ./examples/with-hooks
package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/Brah-Timo/gofastapi"
	"github.com/Brah-Timo/gofastapi/crud"
	"github.com/Brah-Timo/gofastapi/middleware"
)

// BlogPost represents a published article.
type BlogPost struct {
	ID        uint   `json:"id"         gorm:"primaryKey"`
	Title     string `json:"title"      validate:"required,min=5,max=200"`
	Slug      string `json:"slug"       gorm:"uniqueIndex"`
	Body      string `json:"body"       validate:"required,min=10"`
	AuthorID  uint   `json:"author_id"`
	Published bool   `json:"published"  gorm:"default:false"`
	Protected bool   `json:"protected"  gorm:"default:false"` // cannot be deleted
}

func main() {
	db := gofastapi.ConnectDB("sqlite://./hooks-example.db")
	gofastapi.MustAutoMigrate[BlogPost](db)

	gofastapi.Use(
		middleware.Recovery(),
		middleware.Logger(),
	)

	gofastapi.CRUD[BlogPost]("/posts", db,
		crud.WithPageSize[BlogPost](10),
		crud.WithSearchFields[BlogPost]("title", "body"),
		crud.WithSoftDelete[BlogPost](),

		// ── BeforeCreate ──────────────────────────────────────────────────
		// Auto-generate a URL-safe slug from the title.
		crud.WithBeforeCreate[BlogPost](func(post *BlogPost, ctx crud.Context) error {
			if post.Slug == "" {
				post.Slug = slugify(post.Title)
			}
			log.Printf("📝 Creating post: %q (slug: %s)", post.Title, post.Slug)
			return nil
		}),

		// ── AfterCreate ───────────────────────────────────────────────────
		// "Send" a notification (simulated with a log line).
		crud.WithAfterCreate[BlogPost](func(post *BlogPost, ctx crud.Context) error {
			go func() {
				log.Printf("📧 [notification] New post published: %q (id=%d)", post.Title, post.ID)
				// In production: call email/slack/webhook service here.
			}()
			return nil
		}),

		// ── BeforeUpdate ──────────────────────────────────────────────────
		// Re-generate slug if title changed and slug was auto-generated.
		crud.WithBeforeUpdate[BlogPost](func(post *BlogPost, ctx crud.Context) error {
			if post.Slug == "" {
				post.Slug = slugify(post.Title)
			}
			return nil
		}),

		// ── BeforeDelete ──────────────────────────────────────────────────
		// Prevent deletion of posts marked as protected.
		crud.WithBeforeDelete[BlogPost](func(post *BlogPost, ctx crud.Context) error {
			if post.Protected {
				return errors.New("this post is protected and cannot be deleted")
			}
			return nil
		}),

		// ── AfterDelete ───────────────────────────────────────────────────
		// Log deletion for audit trail.
		crud.WithAfterDelete[BlogPost](func(post *BlogPost, ctx crud.Context) error {
			log.Printf("🗑️  Post deleted: id=%d title=%q by IP=%s",
				post.ID, post.Title, ctx.ClientIP())
			return nil
		}),
	)

	fmt.Println("\n📋 Available endpoints:")
	fmt.Println("  GET    /posts          — list posts")
	fmt.Println("  GET    /posts/:id      — get post")
	fmt.Println("  POST   /posts          — create (slug auto-generated)")
	fmt.Println("  PUT    /posts/:id      — update")
	fmt.Println("  DELETE /posts/:id      — delete (blocked if protected=true)")
	fmt.Println()

	gofastapi.Run(":8080")
}

// slugify converts a title to a URL-safe slug.
// "Hello, World!" → "hello-world"
func slugify(s string) string {
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
