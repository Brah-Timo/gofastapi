// Package validation wraps go-playground/validator/v10 with a gofastapi-
// friendly API that returns structured field errors instead of raw strings.
package validation

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// ─────────────────────────────────────────────────────────────────────────────
// Validator — thread-safe singleton wrapper
// ─────────────────────────────────────────────────────────────────────────────

// Validator wraps go-playground/validator with gofastapi-specific helpers.
type Validator struct {
	v *validator.Validate
}

var (
	once     sync.Once
	instance *Validator
)

// New returns a configured Validator instance.
// The instance is created once and reused (the underlying validator is
// goroutine-safe after registration).
func New() *Validator {
	once.Do(func() {
		v := validator.New()

		// Use JSON tag names in error messages instead of Go field names.
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return fld.Name
			}
			if name == "" {
				return fld.Name
			}
			return name
		})

		// Register custom rules.
		registerCustomRules(v)

		instance = &Validator{v: v}
	})
	return instance
}

// ─────────────────────────────────────────────────────────────────────────────
// Validate — the primary method
// ─────────────────────────────────────────────────────────────────────────────

// Validate validates the struct s using its `validate` struct tags.
// Returns a map of field name → error message, or nil if valid.
//
//	type User struct {
//	    Name  string `json:"name"  validate:"required,min=2,max=50"`
//	    Email string `json:"email" validate:"required,email"`
//	    Age   int    `json:"age"   validate:"min=0,max=150"`
//	}
//
//	errs := v.Validate(user)
//	// → {"name": "min must be 2", "email": "must be a valid email address"}
func (vl *Validator) Validate(s any) map[string]string {
	err := vl.v.Struct(s)
	if err == nil {
		return nil
	}

	var ve validator.ValidationErrors
	if !isValidationErrors(err, &ve) {
		// Non-validation error (e.g. nil pointer, not a struct) → generic.
		return map[string]string{"_": err.Error()}
	}

	errs := make(map[string]string, len(ve))
	for _, fe := range ve {
		errs[fe.Field()] = humaniseTag(fe)
	}
	return errs
}

// ValidateVar validates a single variable against the provided tag.
// Returns an error message or empty string if valid.
//
//	msg := v.ValidateVar("not-an-email", "email")
//	// → "must be a valid email address"
func (vl *Validator) ValidateVar(value any, tag string) string {
	err := vl.v.Var(value, tag)
	if err == nil {
		return ""
	}
	var ve validator.ValidationErrors
	if isValidationErrors(err, &ve) && len(ve) > 0 {
		return humaniseTag(ve[0])
	}
	return err.Error()
}

// RegisterRule adds a custom validation rule.
//
//	v.RegisterRule("is_slug", func(fl validator.FieldLevel) bool {
//	    return regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(fl.Field().String())
//	}, "must be a valid URL slug")
func (vl *Validator) RegisterRule(tag string, fn validator.Func, msg string) error {
	customMessages[tag] = msg
	return vl.v.RegisterValidation(tag, fn)
}

// ─────────────────────────────────────────────────────────────────────────────
// Custom rules (rules.go integration)
// ─────────────────────────────────────────────────────────────────────────────

func registerCustomRules(v *validator.Validate) {
	for tag, rule := range customRules {
		v.RegisterValidation(tag, rule) //nolint:errcheck
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Error message humanisation
// ─────────────────────────────────────────────────────────────────────────────

// customMessages maps validation tags to human-readable messages.
// Values registered via RegisterRule are added here.
var customMessages = map[string]string{
	// Custom rule messages (populated by rules.go).
}

// humaniseTag converts a validator.FieldError to a human-readable string.
func humaniseTag(fe validator.FieldError) string {
	// Check custom messages first.
	if msg, ok := customMessages[fe.Tag()]; ok {
		return msg
	}

	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		if fe.Type().Kind() == reflect.String {
			return fmt.Sprintf("must be at least %s characters long", fe.Param())
		}
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		if fe.Type().Kind() == reflect.String {
			return fmt.Sprintf("must be at most %s characters long", fe.Param())
		}
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters long", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", strings.ReplaceAll(fe.Param(), " ", ", "))
	case "url":
		return "must be a valid URL"
	case "uri":
		return "must be a valid URI"
	case "uuid":
		return "must be a valid UUID"
	case "uuid4":
		return "must be a valid UUIDv4"
	case "numeric":
		return "must contain only numeric characters"
	case "alpha":
		return "must contain only alphabetic characters"
	case "alphanum":
		return "must contain only alphanumeric characters"
	case "unique":
		return "must contain unique values"
	case "eqfield":
		return fmt.Sprintf("must equal field %s", fe.Param())
	case "nefield":
		return fmt.Sprintf("must not equal field %s", fe.Param())
	}

	return fmt.Sprintf("failed on rule '%s'", fe.Tag())
}

// isValidationErrors is a type assertion helper.
func isValidationErrors(err error, ve *validator.ValidationErrors) bool {
	e, ok := err.(validator.ValidationErrors)
	if ok {
		*ve = e
	}
	return ok
}
