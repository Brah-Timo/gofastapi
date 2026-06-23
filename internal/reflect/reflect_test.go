package reflect_test

import (
	"testing"

	ireflect "github.com/Brah-Timo/gofastapi/internal/reflect"
)

type SampleModel struct {
	ID        uint   `json:"id"    gorm:"primaryKey"`
	FirstName string `json:"first_name" validate:"required,min=2"`
	Email     string `json:"email"      validate:"required,email" gorm:"column:email_address"`
	private   string //nolint:unused // intentionally unexported
}

func TestOf_TypeName(t *testing.T) {
	ti := ireflect.Of[SampleModel]()
	if ti.Name != "SampleModel" {
		t.Errorf("expected SampleModel, got %q", ti.Name)
	}
}

func TestOf_Fields(t *testing.T) {
	ti := ireflect.Of[SampleModel]()
	// private field must be excluded
	if len(ti.Fields) != 3 {
		t.Errorf("expected 3 exported fields, got %d", len(ti.Fields))
	}
}

func TestOf_PrimaryKey(t *testing.T) {
	ti := ireflect.Of[SampleModel]()
	if ti.PrimaryKey == nil {
		t.Fatal("expected primary key to be detected")
	}
	if ti.PrimaryKey.Name != "ID" {
		t.Errorf("expected primary key field name ID, got %q", ti.PrimaryKey.Name)
	}
}

func TestOf_JSONFields(t *testing.T) {
	ti := ireflect.Of[SampleModel]()
	if _, ok := ti.JSONFields["email"]; !ok {
		t.Error("expected json field 'email' to be indexed")
	}
}

func TestOf_DBColumn(t *testing.T) {
	ti := ireflect.Of[SampleModel]()
	if fi, ok := ti.DBFields["email_address"]; !ok {
		t.Error("expected db column 'email_address' to be indexed")
	} else if fi.Name != "Email" {
		t.Errorf("expected Go field name Email, got %q", fi.Name)
	}
}

func TestOf_Cache(t *testing.T) {
	// Second call must return the same pointer (cached).
	ti1 := ireflect.Of[SampleModel]()
	ti2 := ireflect.Of[SampleModel]()
	if ti1 != ti2 {
		t.Error("expected cached TypeInfo to be the same pointer")
	}
}

func TestGetPrimaryKeyValue(t *testing.T) {
	m := SampleModel{ID: 42, FirstName: "Alice", Email: "a@example.com"}
	val, ok := ireflect.GetPrimaryKeyValue(m)
	if !ok {
		t.Fatal("expected primary key to be found")
	}
	if v, ok := val.(uint); !ok || v != 42 {
		t.Errorf("expected pk=42, got %v", val)
	}
}

func TestIsZero(t *testing.T) {
	if !ireflect.IsZero(nil) {
		t.Error("nil should be zero")
	}
	if !ireflect.IsZero(0) {
		t.Error("0 should be zero")
	}
	if ireflect.IsZero(1) {
		t.Error("1 should not be zero")
	}
}

func TestUnqualifiedName(t *testing.T) {
	name := ireflect.UnqualifiedName[SampleModel]()
	if name != "SampleModel" {
		t.Errorf("expected SampleModel, got %q", name)
	}
}
