package libreoffice

import (
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
	libreofficeapi "github.com/gotenberg/gotenberg/v9/pkg/modules/libreoffice/api"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/pdf"
)

func init() {
	gotenberg.MustRegisterModule(new(LibreOffice))
}

// LibreOffice is a module that provides a route for converting documents to
// PDF with LibreOffice.
type LibreOffice struct {
	api           libreofficeapi.Converter
	processor     *pdf.Processor
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

	provider, err := ctx.Module(new(libreofficeapi.Provider))
	if err != nil {
		return fmt.Errorf("get LibreOffice converter provider: %w", err)
	}

	libreOfficeApi, err := provider.(libreofficeapi.Provider).LibreOffice()
	if err != nil {
		return fmt.Errorf("get LibreOffice converter: %w", err)
	}

	mod.api = libreOfficeApi

	provider, err = ctx.Module(new(pdf.Provider))
	if err != nil {
		return fmt.Errorf("get PDF provider: %w", err)
	}

	mod.processor = provider.(pdf.Provider).Processor()

	return nil
}

// Routes returns the HTTP routes.
func (mod *LibreOffice) Routes() ([]api.Route, error) {
	if mod.disableRoutes {
		return nil, nil
	}

	return []api.Route{
		convertRoute(mod.api, mod.processor),
	}, nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*LibreOffice)(nil)
	_ gotenberg.Provisioner = (*LibreOffice)(nil)
	_ api.Router            = (*LibreOffice)(nil)
)
