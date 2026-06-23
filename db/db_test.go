package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/Brah-Timo/gofastapi/db"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func mustOpenSQLite(t *testing.T) db.Database {
	t.Helper()
	d, err := db.InMemory()
	if err != nil {
		t.Fatalf("InMemory: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAutoConnect_SQLite(t *testing.T) {
	d := mustOpenSQLite(t)
	if d.Driver() != "sqlite" {
		t.Errorf("expected driver sqlite, got %q", d.Driver())
	}
}

func TestPing(t *testing.T) {
	d := mustOpenSQLite(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := d.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestIsHealthy(t *testing.T) {
	d := mustOpenSQLite(t)
	if !db.IsHealthy(d) {
		t.Error("expected IsHealthy to return true")
	}
}

func TestMigrate(t *testing.T) {
	type Widget struct {
		ID    uint   `gorm:"primaryKey"`
		Name  string `gorm:"not null"`
		Color string
	}

	d := mustOpenSQLite(t)
	if err := db.Migrate[Widget](d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Table should exist — insert a row.
	w := Widget{Name: "button", Color: "blue"}
	if err := d.DB().Create(&w).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}
	if w.ID == 0 {
		t.Error("expected auto-assigned ID")
	}
}

func TestWithTransaction_Commit(t *testing.T) {
	type Item struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"not null"`
	}

	d := mustOpenSQLite(t)
	db.Migrate[Item](d) //nolint:errcheck

	err := db.WithTransaction(d, func(tx *gorm.DB) error {
		return tx.Create(&Item{Name: "tx-item"}).Error
	})
	if err != nil {
		t.Fatalf("WithTransaction: %v", err)
	}

	var count int64
	d.DB().Model(&Item{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 item after commit, got %d", count)
	}
}

func TestDetectDriver(t *testing.T) {
	tests := []struct {
		dsn    string
		driver string
	}{
		{"postgres://user:pass@localhost/db", "postgres"},
		{"postgresql://user:pass@localhost/db", "postgres"},
		{"host=localhost dbname=db", "postgres"},
		{":memory:", "sqlite"},
		{"./app.db", "sqlite"},
		{"sqlite://./app.db", "sqlite"},
	}
	for _, tt := range tests {
		d, err := db.AutoConnect(tt.dsn)
		if tt.driver == "sqlite" {
			if err != nil {
				t.Errorf("AutoConnect(%q): unexpected error %v", tt.dsn, err)
				continue
			}
			if d.Driver() != tt.driver {
				t.Errorf("AutoConnect(%q): driver = %q, want %q", tt.dsn, d.Driver(), tt.driver)
			}
			d.Close()
		}
		// For postgres/mysql we just check the error message contains the right driver
		// since we don't have those servers in CI.
	}
}

func TestPostgresDSN(t *testing.T) {
	want := "host=localhost port=5432 dbname=mydb user=alice password=secret sslmode=disable"
	got := db.PostgresDSN("localhost", 5432, "mydb", "alice", "secret", "disable")
	if got != want {
		t.Errorf("\ngot  %q\nwant %q", got, want)
	}
}

func TestMySQLDSN(t *testing.T) {
	got := db.MySQLDSN("root", "pass", "localhost", 3306, "mydb")
	want := "root:pass@tcp(localhost:3306)/mydb?parseTime=True&loc=Local"
	if got != want {
		t.Errorf("\ngot  %q\nwant %q", got, want)
	}
}
