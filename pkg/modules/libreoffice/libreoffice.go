package libreoffice

import (
	"fmt"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/uno"
	flag "github.com/spf13/pflag"
)

func init() {
	gotenberg.MustRegisterModule(LibreOffice{})
}

// LibreOffice is a module which provides a route for converting documents to
// PDF with LibreOffice.
type LibreOffice struct {
	unoAPI        uno.API
	engine        gotenberg.PDFEngine
	disableRoutes bool
}

// Descriptor returns a LibreOffice's module descriptor.
func (LibreOffice) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "libreoffice",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("libreoffice", flag.ExitOnError)
			fs.Bool("libreoffice-disable-routes", false, "Disable the routes")

			return fs
		}(),
		New: func() gotenberg.Module { return new(LibreOffice) },
	}
}

// Provision sets the module properties.
func (mod *LibreOffice) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mod.disableRoutes = flags.MustBool("libreoffice-disable-routes")

	provider, err := ctx.Module(new(uno.Provider))
	if err != nil {
		return fmt.Errorf("get unoAPI provider: %w", err)
	}

	unoAPI, err := provider.(uno.Provider).UNO()
	if err != nil {
		return fmt.Errorf("get unoAPI API: %w", err)
	}

	mod.unoAPI = unoAPI

	provider, err = ctx.Module(new(gotenberg.PDFEngineProvider))
	if err != nil {
		return fmt.Errorf("get PDF engine provider: %w", err)
	}

	engine, err := provider.(gotenberg.PDFEngineProvider).PDFEngine()
	if err != nil {
		return fmt.Errorf("get PDF engine: %w", err)
	}

	mod.engine = engine

	return nil
}

// Routes returns the HTTP routes.
func (mod LibreOffice) Routes() ([]api.Route, error) {
	if mod.disableRoutes {
		return nil, nil
	}

	return []api.Route{
		convertRoute(mod.unoAPI, mod.engine),
	}, nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*LibreOffice)(nil)
	_ gotenberg.Provisioner = (*LibreOffice)(nil)
	_ api.Router            = (*LibreOffice)(nil)
)
