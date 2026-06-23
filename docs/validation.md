# Validation Reference

gofastapi uses `go-playground/validator/v10` under the hood.

## Struct Tags

Add a `validate` tag to any field:

```go
type User struct {
    Name     string `json:"name"     validate:"required,min=2,max=100"`
    Email    string `json:"email"    validate:"required,email"`
    Age      int    `json:"age"      validate:"min=0,max=150"`
    Role     string `json:"role"     validate:"oneof=admin member guest"`
    Website  string `json:"website"  validate:"omitempty,url"`
    Password string `json:"password" validate:"required,strong_pass"`
    Slug     string `json:"slug"     validate:"required,slug"`
    Phone    string `json:"phone"    validate:"omitempty,phone"`
    Color    string `json:"color"    validate:"omitempty,hex_color"`
    Version  string `json:"version"  validate:"omitempty,semver"`
}
```

## Standard Rules

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field must be present and non-zero | — |
| `omitempty` | Skip validation if field is zero | — |
| `min=N` | Min value (number) or min length (string) | `min=2` |
| `max=N` | Max value (number) or max length (string) | `max=100` |
| `len=N` | Exact length | `len=36` |
| `gt=N` | Greater than | `gt=0` |
| `gte=N` | Greater than or equal | `gte=18` |
| `lt=N` | Less than | `lt=1000` |
| `lte=N` | Less than or equal | `lte=5` |
| `email` | Valid email address | — |
| `url` | Valid HTTP/HTTPS URL | — |
| `uri` | Valid URI (any scheme) | — |
| `uuid` | Valid UUID (any version) | — |
| `uuid4` | Valid UUIDv4 | — |
| `numeric` | Only digits | — |
| `alpha` | Only letters | — |
| `alphanum` | Letters and digits only | — |
| `oneof=a b c` | Value must be one of the listed options | `oneof=red green blue` |
| `eqfield=Other` | Must equal another field | `eqfield=Password` |

## Custom Rules (Built-in)

| Tag | Valid example | Description |
|-----|--------------|-------------|
| `slug` | `my-blog-post` | Lowercase letters, numbers, hyphens |
| `no_whitespace` | `helloworld` | No spaces or whitespace characters |
| `strong_pass` | `Str0ng!Pass` | 8+ chars, upper, lower, digit, special |
| `phone` | `+1 (555) 123-4567` | International phone number |
| `hex_color` | `#3498db` | CSS hex colour (#RGB or #RRGGBB) |
| `semver` | `2.14.0` | Semantic version (X.Y.Z) |

## Adding Custom Rules

```go
v := validation.New()

v.RegisterRule("positive_even", func(fl validator.FieldLevel) bool {
    n := fl.Field().Int()
    return n > 0 && n%2 == 0
}, "must be a positive even number")
```

Then use it:

```go
type Config struct {
    Workers int `json:"workers" validate:"required,positive_even"`
}
```

## Validation Error Response

When validation fails, gofastapi returns HTTP 422 with structured details:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "one or more fields failed validation",
    "details": {
      "email":    "must be a valid email address",
      "password": "must be at least 8 characters and contain uppercase, lowercase, a digit, and a special character",
      "slug":     "must be a valid URL slug (lowercase letters, numbers, and hyphens only)"
    }
  }
}
```

Field names in error responses use **JSON tag names** (not Go struct field names).
