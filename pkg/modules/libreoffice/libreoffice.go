package libreoffice

import (
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
	libeofficeapi "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
)

func init() {
	gotenberg.MustRegisterModule(new(LibreOffice))
}

// LibreOffice is a module which provides a route for converting documents to
// PDF with LibreOffice.
type LibreOffice struct {
	api           libeofficeapi.Uno
	engine        gotenberg.PdfEngine
	disableRoutes bool
}

// Descriptor returns a [LibreOffice]'s module descriptor.
func (mod *LibreOffice) Descriptor() gotenberg.ModuleDescriptor {
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

	provider, err := ctx.Module(new(libeofficeapi.Provider))
	if err != nil {
		return fmt.Errorf("get LibreOffice Uno provider: %w", err)
	}

	libreOfficeApi, err := provider.(libeofficeapi.Provider).LibreOffice()
	if err != nil {
		return fmt.Errorf("get LibreOffice Uno: %w", err)
	}

	mod.api = libreOfficeApi

	provider, err = ctx.Module(new(gotenberg.PdfEngineProvider))
	if err != nil {
		return fmt.Errorf("get PDF engine provider: %w", err)
	}

	engine, err := provider.(gotenberg.PdfEngineProvider).PdfEngine()
	if err != nil {
		return fmt.Errorf("get PDF engine: %w", err)
	}

	mod.engine = engine

	return nil
}

// Routes returns the HTTP routes.
func (mod *LibreOffice) Routes() ([]api.Route, error) {
	if mod.disableRoutes {
		return nil, nil
	}

	return []api.Route{
		convertRoute(mod.api, mod.engine),
	}, nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*LibreOffice)(nil)
	_ gotenberg.Provisioner = (*LibreOffice)(nil)
	_ api.Router            = (*LibreOffice)(nil)
)
