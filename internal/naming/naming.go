// Package naming provides automatic naming conventions for gofastapi.
// It converts Go struct names into URL route prefixes and database table names
// following Rails-style conventions: plural snake_case.
//
//	User      → /users      + users
//	BlogPost  → /blog-posts + blog_posts
//	Category  → /categories + categories
//	Person    → /people     + people     (irregular plural)
package naming

import (
	"strings"
	"unicode"
)

// ─────────────────────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────────────────────

// RoutePrefix converts a struct name to a URL route prefix.
// Uses plural kebab-case: "BlogPost" → "/blog-posts".
func RoutePrefix(structName string) string {
	return "/" + ToKebabCase(Pluralize(structName))
}

// TableName converts a struct name to a database table name.
// Uses plural snake_case: "BlogPost" → "blog_posts".
func TableName(structName string) string {
	return ToSnakeCase(Pluralize(structName))
}

// ─────────────────────────────────────────────────────────────────────────────
// Pluralize — English-aware pluralisation
// ─────────────────────────────────────────────────────────────────────────────

// irregulars maps singular → plural for irregular English words.
var irregulars = map[string]string{
	"person":     "people",
	"man":        "men",
	"woman":      "women",
	"child":      "children",
	"foot":       "feet",
	"tooth":      "teeth",
	"goose":      "geese",
	"mouse":      "mice",
	"ox":         "oxen",
	"leaf":       "leaves",
	"life":       "lives",
	"knife":      "knives",
	"wife":       "wives",
	"half":       "halves",
	"shelf":      "shelves",
	"loaf":       "loaves",
	"potato":     "potatoes",
	"tomato":     "tomatoes",
	"cactus":     "cacti",
	"focus":      "foci",
	"fungus":     "fungi",
	"nucleus":    "nuclei",
	"syllabus":   "syllabi",
	"analysis":   "analyses",
	"diagnosis":  "diagnoses",
	"oasis":      "oases",
	"thesis":     "theses",
	"crisis":     "crises",
	"phenomenon": "phenomena",
	"criterion":  "criteria",
	"datum":      "data",
	"medium":     "media",
	"series":     "series",
	"species":    "species",
	"sheep":      "sheep",
	"fish":       "fish",
	"deer":       "deer",
	"aircraft":   "aircraft",
	"news":       "news",
}

// Pluralize returns the English plural of word.
// word is expected to be a single lowercase word.
func Pluralize(word string) string {
	lower := strings.ToLower(word)

	// Check irregulars first.
	if plural, ok := irregulars[lower]; ok {
		return plural
	}

	// Already ends in common plural suffixes → return as-is.
	for _, suffix := range []string{"s", "x", "z", "ch", "sh"} {
		if strings.HasSuffix(lower, suffix) && !strings.HasSuffix(lower, "ss") {
			return lower + "es"
		}
	}

	// Consonant + "y" → drop "y", add "ies"
	if strings.HasSuffix(lower, "y") && len(lower) > 1 {
		penultimate := lower[len(lower)-2]
		if !isVowel(rune(penultimate)) {
			return lower[:len(lower)-1] + "ies"
		}
	}

	// "f" or "fe" → "ves"
	if strings.HasSuffix(lower, "fe") {
		return lower[:len(lower)-2] + "ves"
	}
	if strings.HasSuffix(lower, "f") && !strings.HasSuffix(lower, "ff") {
		return lower[:len(lower)-1] + "ves"
	}

	// Default: add "s"
	return lower + "s"
}

// isVowel reports whether r is an English vowel.
func isVowel(r rune) bool {
	return strings.ContainsRune("aeiou", unicode.ToLower(r))
}

// ─────────────────────────────────────────────────────────────────────────────
// Case converters
// ─────────────────────────────────────────────────────────────────────────────

// ToSnakeCase converts "BlogPost" → "blog_post", "UserID" → "user_id".
// Handles consecutive uppercase letters (acronyms): "HTMLParser" → "html_parser".
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		upper := unicode.IsUpper(r)
		if upper {
			// Insert underscore if:
			//   • not the first character, AND
			//   • previous char was lowercase, OR next char is lowercase (end of acronym)
			if i > 0 {
				prevLower := !unicode.IsUpper(runes[i-1])
				nextLower := i+1 < len(runes) && !unicode.IsUpper(runes[i+1]) && !unicode.IsDigit(runes[i+1])
				if prevLower || nextLower {
					b.WriteByte('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ToKebabCase converts "BlogPost" → "blog-post", "UserID" → "user-id".
func ToKebabCase(s string) string {
	return strings.ReplaceAll(ToSnakeCase(s), "_", "-")
}

// ToCamelCase converts "blog_post" or "blog-post" → "BlogPost".
func ToCamelCase(s string) string {
	var b strings.Builder
	nextUpper := true
	for _, r := range s {
		if r == '_' || r == '-' {
			nextUpper = true
			continue
		}
		if nextUpper {
			b.WriteRune(unicode.ToUpper(r))
			nextUpper = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ToLowerCamelCase converts "BlogPost" → "blogPost", "user_id" → "userId".
func ToLowerCamelCase(s string) string {
	cc := ToCamelCase(s)
	if cc == "" {
		return ""
	}
	runes := []rune(cc)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// ─────────────────────────────────────────────────────────────────────────────
// Struct-name conveniences
// ─────────────────────────────────────────────────────────────────────────────

// StructToRoute converts a Go struct name to its conventional REST route prefix.
// Handles multi-word struct names via camelCase splitting.
//
// Examples:
//
//	User         → "/users"
//	BlogPost     → "/blog-posts"
//	ProductOrder → "/product-orders"
func StructToRoute(structName string) string {
	return RoutePrefix(structName)
}

// StructToTable converts a Go struct name to a database table name.
//
// Examples:
//
//	User         → "users"
//	BlogPost     → "blog_posts"
//	ProductOrder → "product_orders"
func StructToTable(structName string) string {
	return TableName(structName)
}
