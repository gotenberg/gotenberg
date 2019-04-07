package resource

import (
	"os"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/hijackable"
)

type Resource interface {
	hijackable.Hijackable
	Parse(c echo.Context) error
}

type baseResource struct {
	values  []string
	dirPath string
}

func (r *baseResource) Parse(c echo.Context) error {
	return nil
}

func (r *baseResource) Hijack() error {
	return os.RemoveAll(r.dirPath)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Resource(new(baseResource))
	_ = hijackable.Hijackable(new(baseResource))
)
