// Package reflect provides internal reflection helpers for gofastapi.
// It wraps the standard library's reflect package with higher-level utilities
// specifically tailored for generic struct introspection.
//
// All functions in this package are internal — they are not part of the public
// API and may change without notice.
package reflect

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// TypeInfo — cached struct metadata
// ─────────────────────────────────────────────────────────────────────────────

// FieldInfo holds metadata about a single struct field.
type FieldInfo struct {
	// Name is the Go field name (e.g. "UserEmail").
	Name string
	// JSONTag is the value of the `json` struct tag (empty string if absent).
	JSONTag string
	// DBTag is the value of the `gorm` or `db` struct tag.
	DBTag string
	// ValidateTag is the value of the `validate` struct tag.
	ValidateTag string
	// Type is the reflect.Type of the field.
	Type reflect.Type
	// Index is the field index within the struct.
	Index int
	// IsExported reports whether the field is exported.
	IsExported bool
	// IsPrimaryKey reports whether the field is tagged as a primary key.
	IsPrimaryKey bool
	// IsRequired reports whether the validate tag contains "required".
	IsRequired bool
}

// TypeInfo holds all reflected metadata for a struct type.
// Instances are cached after the first computation.
type TypeInfo struct {
	// Type is the reflect.Type of the struct (dereferenced if pointer).
	Type reflect.Type
	// Name is the unqualified type name (e.g. "User").
	Name string
	// PkgPath is the import path of the package defining this type.
	PkgPath string
	// Fields is the ordered list of struct fields.
	Fields []FieldInfo
	// PrimaryKey is the first field tagged as primary key (or named "ID").
	PrimaryKey *FieldInfo
	// JSONFields maps json tag → FieldInfo for fast lookup.
	JSONFields map[string]*FieldInfo
	// DBFields maps db column name → FieldInfo for fast lookup.
	DBFields map[string]*FieldInfo
}

// cache stores TypeInfo instances keyed by reflect.Type to avoid recomputing.
var (
	cacheMu sync.RWMutex
	cache   = make(map[reflect.Type]*TypeInfo)
)

// Of returns the TypeInfo for the concrete type T.
// Results are cached after the first call for each type.
func Of[T any]() *TypeInfo {
	var zero T
	t := reflect.TypeOf(zero)
	// Dereference pointer types.
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return ofType(t)
}

// OfValue returns the TypeInfo for the dynamic type of v.
func OfValue(v any) *TypeInfo {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return ofType(t)
}

func ofType(t reflect.Type) *TypeInfo {
	cacheMu.RLock()
	if ti, ok := cache[t]; ok {
		cacheMu.RUnlock()
		return ti
	}
	cacheMu.RUnlock()

	// Build TypeInfo (may duplicate work under rare concurrent first calls,
	// but that's acceptable — result is deterministic).
	ti := buildTypeInfo(t)

	cacheMu.Lock()
	cache[t] = ti
	cacheMu.Unlock()

	return ti
}

func buildTypeInfo(t reflect.Type) *TypeInfo {
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gofastapi/internal/reflect: Of called on non-struct type %s", t))
	}

	ti := &TypeInfo{
		Type:       t,
		Name:       t.Name(),
		PkgPath:    t.PkgPath(),
		JSONFields: make(map[string]*FieldInfo),
		DBFields:   make(map[string]*FieldInfo),
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		fi := FieldInfo{
			Name:       sf.Name,
			Type:       sf.Type,
			Index:      i,
			IsExported: true,
		}

		// Parse struct tags.
		fi.JSONTag = parseFirstTagPart(sf.Tag.Get("json"))
		fi.DBTag = parseGORMColumn(sf.Tag.Get("gorm"), sf.Name)
		fi.ValidateTag = sf.Tag.Get("validate")
		fi.IsPrimaryKey = strings.Contains(sf.Tag.Get("gorm"), "primaryKey") ||
			strings.EqualFold(sf.Name, "id")
		fi.IsRequired = strings.Contains(fi.ValidateTag, "required")

		ti.Fields = append(ti.Fields, fi)
		ref := &ti.Fields[len(ti.Fields)-1]

		if fi.JSONTag != "" && fi.JSONTag != "-" {
			ti.JSONFields[fi.JSONTag] = ref
		}
		if fi.DBTag != "" {
			ti.DBFields[fi.DBTag] = ref
		}
		if fi.IsPrimaryKey && ti.PrimaryKey == nil {
			ti.PrimaryKey = ref
		}
	}

	return ti
}

// ─────────────────────────────────────────────────────────────────────────────
// Value helpers
// ─────────────────────────────────────────────────────────────────────────────

// GetPrimaryKeyValue extracts the primary key value from a struct instance.
// Returns the zero value and false if no primary key field is found.
func GetPrimaryKeyValue(v any) (any, bool) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	ti := ofType(rv.Type())
	if ti.PrimaryKey == nil {
		return nil, false
	}
	val := rv.Field(ti.PrimaryKey.Index).Interface()
	return val, true
}

// SetField sets the value of a named field (by JSON tag or Go name) on v.
// v must be a pointer to a struct.
func SetField(v any, fieldName string, value any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("SetField: v must be a pointer")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("SetField: v must point to a struct")
	}

	ti := ofType(rv.Type())
	var fi *FieldInfo

	// Try JSON tag first, then Go field name.
	if f, ok := ti.JSONFields[fieldName]; ok {
		fi = f
	} else {
		for i := range ti.Fields {
			if ti.Fields[i].Name == fieldName {
				fi = &ti.Fields[i]
				break
			}
		}
	}
	if fi == nil {
		return fmt.Errorf("SetField: field %q not found on %s", fieldName, ti.Name)
	}

	fv := rv.Field(fi.Index)
	if !fv.CanSet() {
		return fmt.Errorf("SetField: field %q is not settable", fieldName)
	}
	fv.Set(reflect.ValueOf(value).Convert(fi.Type))
	return nil
}

// IsZero reports whether v is the zero value for its type.
func IsZero(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.IsZero()
}

// ZeroNew returns a pointer to a new zero value of type T.
func ZeroNew[T any]() *T {
	return new(T)
}

// SliceOf returns a pointer to a new empty slice of type T.
func SliceOf[T any]() *[]T {
	s := make([]T, 0)
	return &s
}

// ─────────────────────────────────────────────────────────────────────────────
// Tag parsing helpers (internal)
// ─────────────────────────────────────────────────────────────────────────────

// parseFirstTagPart returns the first comma-separated part of a struct tag
// value (the "name" segment). E.g. `json:"email,omitempty"` → "email".
func parseFirstTagPart(tag string) string {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx]
	}
	return tag
}

// parseGORMColumn extracts the column name from a gorm tag.
// Falls back to snake_case of the Go field name when no column tag is present.
func parseGORMColumn(gormTag, fieldName string) string {
	for _, part := range strings.Split(gormTag, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "column:") {
			return strings.TrimPrefix(strings.ToLower(part), "column:")
		}
	}
	return toSnakeCase(fieldName)
}

// toSnakeCase converts "UserID" → "user_id", "CreatedAt" → "created_at".
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r | 32) // toLower ASCII fast path
	}
	return b.String()
}

// TypeName returns the package-qualified name of T (e.g. "main.User").
func TypeName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.PkgPath() + "." + t.Name()
}

// UnqualifiedName returns just the struct name of T (e.g. "User").
func UnqualifiedName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}
