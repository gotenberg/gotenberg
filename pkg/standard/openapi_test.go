package standard

import (
	"os"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// builtinRoutes are the routes the api module serves itself, outside any
// [api.Router]: the root, the health check, the version and the debug route.
var builtinRoutes = []string{"GET /", "GET /health", "HEAD /health", "GET /version", "GET /debug"}

// TestOpenAPIListsEveryRoute checks that openapi.yaml, which gotenberg.dev
// generates from its reference data (npm run sync:openapi), describes every
// route the standard modules register, and no other. A route that answers a
// Gotenberg 8 path with where it went, such as /forms/pdfengines/*, is left
// out: it describes no operation.
func TestOpenAPIListsEveryRoute(t *testing.T) {
	data, err := os.ReadFile("../../openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}

	var spec struct {
		OpenAPI string                    `yaml:"openapi"`
		Paths   map[string]map[string]any `yaml:"paths"`
	}
	err = yaml.Unmarshal(data, &spec)
	if err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	if spec.OpenAPI != "3.1.0" {
		t.Errorf("openapi = %q, want 3.1.0", spec.OpenAPI)
	}

	var described []string
	for path, operations := range spec.Paths {
		for method := range operations {
			described = append(described, strings.ToUpper(method)+" "+path)
		}
	}

	registered := slices.Clone(builtinRoutes)
	for _, descriptor := range gotenberg.GetModuleDescriptors() {
		router, ok := descriptor.New().(api.Router)
		if !ok {
			continue
		}

		routes, err := router.Routes()
		if err != nil {
			t.Fatalf("routes of module %q: %v", descriptor.ID, err)
		}

		for _, route := range routes {
			if strings.HasSuffix(route.Path, "/*") {
				continue
			}
			registered = append(registered, route.Method+" "+route.Path)
		}
	}

	slices.Sort(described)
	slices.Sort(registered)

	for _, route := range registered {
		if !slices.Contains(described, route) {
			t.Errorf("%s is missing from openapi.yaml: add it to gotenberg.dev's reference data, then run npm run sync:openapi", route)
		}
	}
	for _, route := range described {
		if !slices.Contains(registered, route) {
			t.Errorf("%s is in openapi.yaml, but no module registers it", route)
		}
	}
}
