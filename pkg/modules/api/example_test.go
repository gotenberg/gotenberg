package api_test

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// Stamp is a module adding the multipart/form-data route /forms/stamp. A
// custom build registers it in an init function, then imports its package,
// next to github.com/gotenberg/gotenberg/v9/pkg/standard, in a main calling
// github.com/gotenberg/gotenberg/v9/cmd.Run.
type Stamp struct{}

// Descriptor returns the module's descriptor.
func (Stamp) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "stamp",
		New: func() gotenberg.Module { return new(Stamp) },
	}
}

// Routes returns the module's routes.
func (Stamp) Routes() ([]api.Route, error) {
	return []api.Route{
		{
			Method:      http.MethodPost,
			Path:        "/forms/stamp",
			IsMultipart: true,
			Handler: func(c *echo.Context) error {
				// The API builds the context of a multipart route: the
				// uploaded files, the form fields, and the working directory.
				ctx := c.Get("context").(*api.Context)

				var inputPaths []string
				var text string
				err := ctx.FormData().
					MandatoryPaths([]string{".pdf"}, &inputPaths).
					String("text", &text).
					Validate()
				if err != nil {
					return fmt.Errorf("validate form data: %w", err)
				}

				// Write the results under ctx.GeneratePath(".pdf") and register
				// them: the API sends them back, zipped if many, or to the
				// webhook.
				return ctx.AddOutputPaths(inputPaths...)
			},
		},
	}, nil
}

// This example writes a module adding a route to the API. The interface
// guards make the compiler check that the module is what the application
// looks for.
func Example_router() {
	var (
		_ gotenberg.Module = (*Stamp)(nil)
		_ api.Router       = (*Stamp)(nil)
	)

	routes, _ := Stamp{}.Routes()
	for _, route := range routes {
		fmt.Println(route.Method, route.Path)
	}
	// Output: POST /forms/stamp
}
