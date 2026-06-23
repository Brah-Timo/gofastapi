package crud_test

import (
	"context"
	"testing"

	"github.com/Brah-Timo/gofastapi/crud"
	"github.com/Brah-Timo/gofastapi/db"
)

type Post struct {
	ID    uint   `gorm:"primaryKey"`
	Title string `gorm:"not null"`
	Body  string
}

func setupPostDB(t *testing.T) db.Database {
	t.Helper()
	d := db.MustInMemory()
	if err := db.Migrate[Post](d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestGORMRepository_Create_FindByID(t *testing.T) {
	d := setupPostDB(t)
	repo := crud.NewRepository[Post](d)

	p := Post{Title: "Hello", Body: "World"}
	if err := repo.Create(context.Background(), &p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == 0 {
		t.Error("expected auto-assigned ID after Create")
	}

	found, err := repo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Title != "Hello" {
		t.Errorf("expected title Hello, got %q", found.Title)
	}
}

func TestGORMRepository_Update(t *testing.T) {
	d := setupPostDB(t)
	repo := crud.NewRepository[Post](d)

	p := Post{Title: "Old", Body: "body"}
	repo.Create(context.Background(), &p)

	p.Title = "New"
	if err := repo.Update(context.Background(), &p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, _ := repo.FindByID(context.Background(), p.ID)
	if found.Title != "New" {
		t.Errorf("expected updated title New, got %q", found.Title)
	}
}

func TestGORMRepository_Delete(t *testing.T) {
	d := setupPostDB(t)
	repo := crud.NewRepository[Post](d)

	p := Post{Title: "ToDelete", Body: "body"}
	repo.Create(context.Background(), &p)

	if err := repo.Delete(context.Background(), p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.FindByID(context.Background(), p.ID)
	if err != crud.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestGORMRepository_List_Pagination(t *testing.T) {
	d := setupPostDB(t)
	repo := crud.NewRepository[Post](d)

	for i := 0; i < 15; i++ {
		repo.Create(context.Background(), &Post{Title: "Post", Body: "body"})
	}

	items, total, err := repo.List(context.Background(), crud.ListParams{
		Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 15 {
		t.Errorf("expected total 15, got %d", total)
	}
	if len(items) != 10 {
		t.Errorf("expected 10 items on page 1, got %d", len(items))
	}

	page2, _, _ := repo.List(context.Background(), crud.ListParams{
		Page: 2, PageSize: 10,
	})
	if len(page2) != 5 {
		t.Errorf("expected 5 items on page 2, got %d", len(page2))
	}
}

func TestGORMRepository_FindByID_NotFound(t *testing.T) {
	d := setupPostDB(t)
	repo := crud.NewRepository[Post](d)

	_, err := repo.FindByID(context.Background(), uint64(9999))
	if err != crud.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
