// Package swagger provides automatic OpenAPI 3.0 documentation generation
// and a Swagger UI server for gofastapi applications.
//
// It builds the spec from the route information collected by the router
// during application startup, so no annotations or code generation are needed.
//
// Usage:
//
//	gofastapi.EnableSwagger("My API", "1.0.0", "My API description")
//
// Then visit http://localhost:8080/docs to see the interactive UI.
package swagger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Brah-Timo/gofastapi/router"
)

// ─────────────────────────────────────────────────────────────────────────────
// OpenAPI 3.0 structs
// ─────────────────────────────────────────────────────────────────────────────

// Spec is the root OpenAPI 3.0 document.
type Spec struct {
	OpenAPI string         `json:"openapi"`
	Info    Info           `json:"info"`
	Paths   map[string]any `json:"paths"`
	Tags    []Tag          `json:"tags,omitempty"`
}

// Info holds API metadata.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Tag groups related endpoints.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Handler
// ─────────────────────────────────────────────────────────────────────────────

// Handler serves the Swagger UI and the OpenAPI spec JSON.
type Handler struct {
	spec []byte
}

// NewHandler builds a Swagger handler from the router's accumulated route spec.
func NewHandler(title, version, description string, routerSpec *router.OpenAPISpec) *Handler {
	s := buildSpec(title, version, description, routerSpec)
	b, _ := json.MarshalIndent(s, "", "  ")
	return &Handler{spec: b}
}

// ServeHTTP serves the Swagger UI HTML page.
func (h *Handler) ServeHTTP(ctx interface {
	JSON(int, any)
	Request() *http.Request
}) {
	if strings.HasSuffix(ctx.Request().URL.Path, "/swagger.json") ||
		strings.HasSuffix(ctx.Request().URL.Path, "/openapi.json") {
		h.ServeSpec(ctx)
		return
	}
	// Serve Swagger UI HTML.
	ctx.JSON(http.StatusOK, map[string]any{
		"_raw_html": swaggerUIHTML(ctx.Request().Host),
	})
}

// ServeSpec serves the raw OpenAPI JSON spec.
func (h *Handler) ServeSpec(ctx interface{ JSON(int, any) }) {
	var spec any
	json.Unmarshal(h.spec, &spec)
	ctx.JSON(http.StatusOK, spec)
}

// ─────────────────────────────────────────────────────────────────────────────
// Spec builder
// ─────────────────────────────────────────────────────────────────────────────

func buildSpec(title, version, description string, routerSpec *router.OpenAPISpec) *Spec {
	s := &Spec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       title,
			Version:     version,
			Description: description,
		},
		Paths: make(map[string]any),
	}

	tagsSeen := make(map[string]bool)

	if routerSpec != nil {
		for _, route := range routerSpec.Routes {
			method := strings.ToLower(route.Method)
			path := ginPathToOpenAPI(route.Path)

			if _, ok := s.Paths[path]; !ok {
				s.Paths[path] = make(map[string]any)
			}

			pathItem := s.Paths[path].(map[string]any)
			op := map[string]any{
				"summary": route.Summary,
				"operationId": fmt.Sprintf("%s_%s",
					method,
					strings.ReplaceAll(strings.Trim(path, "/"), "/", "_"),
				),
			}
			if route.Tag != "" {
				op["tags"] = []string{route.Tag}
				if !tagsSeen[route.Tag] {
					s.Tags = append(s.Tags, Tag{Name: route.Tag})
					tagsSeen[route.Tag] = true
				}
			}

			// Add path parameters.
			params := extractPathParams(path)
			if len(params) > 0 {
				paramDefs := make([]map[string]any, len(params))
				for i, p := range params {
					paramDefs[i] = map[string]any{
						"name":     p,
						"in":       "path",
						"required": true,
						"schema":   map[string]string{"type": "string"},
					}
				}
				op["parameters"] = paramDefs
			}

			pathItem[method] = op
		}
	}

	return s
}

// ginPathToOpenAPI converts Gin's /:param syntax to OpenAPI's /{param} syntax.
func ginPathToOpenAPI(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "{" + p[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func extractPathParams(path string) []string {
	var params []string
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			params = append(params, segment[1:len(segment)-1])
		}
	}
	return params
}

// ─────────────────────────────────────────────────────────────────────────────
// Swagger UI HTML
// ─────────────────────────────────────────────────────────────────────────────

func swaggerUIHTML(host string) string {
	specURL := "/openapi.json"
	_ = host // could be used for absolute URL if needed
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>gofastapi — API Documentation</title>
  <link rel="stylesheet" type="text/css"
    href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #fafafa; }
    .swagger-ui .topbar { background: #1a1a2e; }
    .swagger-ui .topbar .topbar-wrapper .link { display: none; }
    .topbar-title { color: #fff; font-size: 1.2rem; padding: 0 1rem; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: "%s",
        dom_id: '#swagger-ui',
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
        layout: "StandaloneLayout",
        deepLinking: true,
        showExtensions: true,
        showCommonExtensions: true
      });
    };
  </script>
</body>
</html>`, specURL)
}
