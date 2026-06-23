package validation_test

import (
	"testing"

	"github.com/Brah-Timo/gofastapi/validation"
)

type TestUser struct {
	Name  string `json:"name"  validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"min=0,max=150"`
}

type SlugModel struct {
	Slug string `json:"slug" validate:"required,slug"`
}

type PasswordModel struct {
	Password string `json:"password" validate:"required,strong_pass"`
}

func TestValidate_Valid(t *testing.T) {
	v := validation.New()
	errs := v.Validate(TestUser{Name: "Alice", Email: "alice@example.com", Age: 30})
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_RequiredMissing(t *testing.T) {
	v := validation.New()
	errs := v.Validate(TestUser{})
	if _, ok := errs["name"]; !ok {
		t.Error("expected 'name' field error for required")
	}
	if _, ok := errs["email"]; !ok {
		t.Error("expected 'email' field error for required")
	}
}

func TestValidate_InvalidEmail(t *testing.T) {
	v := validation.New()
	errs := v.Validate(TestUser{Name: "Alice", Email: "not-an-email", Age: 0})
	if _, ok := errs["email"]; !ok {
		t.Error("expected 'email' field error")
	}
}

func TestValidate_MinLength(t *testing.T) {
	v := validation.New()
	errs := v.Validate(TestUser{Name: "A", Email: "a@b.com", Age: 0}) // min=2
	if _, ok := errs["name"]; !ok {
		t.Error("expected 'name' min length error")
	}
}

func TestValidate_AgeOutOfRange(t *testing.T) {
	v := validation.New()
	errs := v.Validate(TestUser{Name: "Bob", Email: "bob@example.com", Age: 999})
	if _, ok := errs["age"]; !ok {
		t.Error("expected 'age' max error")
	}
}

func TestValidate_JSONFieldNames(t *testing.T) {
	v := validation.New()
	errs := v.Validate(TestUser{})
	// Errors should use JSON names, not Go field names.
	if _, ok := errs["Name"]; ok {
		t.Error("error should use json name 'name', not Go name 'Name'")
	}
}

func TestCustomRule_Slug_Valid(t *testing.T) {
	v := validation.New()
	errs := v.Validate(SlugModel{Slug: "my-blog-post"})
	if len(errs) != 0 {
		t.Errorf("expected valid slug, got errors: %v", errs)
	}
}

func TestCustomRule_Slug_Invalid(t *testing.T) {
	v := validation.New()
	errs := v.Validate(SlugModel{Slug: "My Blog Post!"})
	if _, ok := errs["slug"]; !ok {
		t.Error("expected slug validation error")
	}
}

func TestCustomRule_StrongPassword_Valid(t *testing.T) {
	v := validation.New()
	errs := v.Validate(PasswordModel{Password: "Str0ng!Pass"})
	if len(errs) != 0 {
		t.Errorf("expected valid password, got: %v", errs)
	}
}

func TestCustomRule_StrongPassword_Weak(t *testing.T) {
	v := validation.New()
	errs := v.Validate(PasswordModel{Password: "weakpassword"})
	if _, ok := errs["password"]; !ok {
		t.Error("expected strong_pass validation error for weak password")
	}
}

func TestValidateVar(t *testing.T) {
	v := validation.New()
	if msg := v.ValidateVar("not-an-email", "email"); msg == "" {
		t.Error("expected error message for invalid email")
	}
	if msg := v.ValidateVar("user@example.com", "email"); msg != "" {
		t.Errorf("expected no error for valid email, got: %q", msg)
	}
}
