// Package validation — custom validation rules.
//
// Add new rules here. They are registered automatically in registerCustomRules.
package validation

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// customRules maps tag name → validation function.
// All entries are registered on the global validator at startup.
var customRules = map[string]validator.Func{
	"slug":          validateSlug,
	"no_whitespace": validateNoWhitespace,
	"strong_pass":   validateStrongPassword,
	"phone":         validatePhone,
	"hex_color":     validateHexColor,
	"semver":        validateSemVer,
}

var (
	slugRegex     = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	hexColorRegex = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	semverRegex   = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	phoneRegex    = regexp.MustCompile(`^\+?[\d\s\-().]{7,20}$`)
)

func init() {
	customMessages["slug"] = "must be a valid URL slug (lowercase letters, numbers, and hyphens only)"
	customMessages["no_whitespace"] = "must not contain whitespace"
	customMessages["strong_pass"] = "must be at least 8 characters and contain uppercase, lowercase, a digit, and a special character"
	customMessages["phone"] = "must be a valid phone number"
	customMessages["hex_color"] = "must be a valid hex colour (e.g. #fff or #ffffff)"
	customMessages["semver"] = "must be a valid semantic version (e.g. 1.2.3)"
}

// validateSlug passes for strings like "my-blog-post" or "api-v2".
func validateSlug(fl validator.FieldLevel) bool {
	return slugRegex.MatchString(fl.Field().String())
}

// validateNoWhitespace passes when the string contains no whitespace.
func validateNoWhitespace(fl validator.FieldLevel) bool {
	for _, r := range fl.Field().String() {
		if unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// validateStrongPassword requires at least:
//   - 8 characters total
//   - 1 uppercase letter
//   - 1 lowercase letter
//   - 1 digit
//   - 1 special character
func validateStrongPassword(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	if len(s) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case strings.ContainsRune(`!@#$%^&*()_+-=[]{}|;':",.<>?/~\`+"`", r):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

// validatePhone passes for common phone number formats.
func validatePhone(fl validator.FieldLevel) bool {
	return phoneRegex.MatchString(fl.Field().String())
}

// validateHexColor passes for CSS hex colours: #fff or #ffffff.
func validateHexColor(fl validator.FieldLevel) bool {
	return hexColorRegex.MatchString(fl.Field().String())
}

// validateSemVer passes for strings like "1.0.0" or "2.14.3".
func validateSemVer(fl validator.FieldLevel) bool {
	return semverRegex.MatchString(fl.Field().String())
}
