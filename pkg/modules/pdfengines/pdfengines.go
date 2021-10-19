package pdfengines

import (
	"errors"
	"fmt"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	flag "github.com/spf13/pflag"
)

func init() {
	gotenberg.MustRegisterModule(PDFEngines{})
}

// PDFEngines is a module which gathers available gotenberg.PDFEngine modules.
// The available gotenberg.PDFEngine modules can be either all
// gotenberg.PDFEngine modules or the modules selected by the user thanks to
// the "engines" flag.
//
// PDFEngines wraps the gotenberg.PDFEngine modules in an internal struct which
// also implements gotenberg.PDFEngine. This struct provides a sort of fallback
// mechanism: if an engine's method returns an error, it calls the same method
// from another engine.
//
// This module implements the gotenberg.PDFEngineProvider interface.
type PDFEngines struct {
	names         []string
	engines       []gotenberg.PDFEngine
	disableRoutes bool
}

// Descriptor returns a PDFEngines' module descriptor.
func (PDFEngines) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "pdfengines",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("pdfengines", flag.ExitOnError)
			fs.StringSlice("pdfengines-engines", make([]string, 0), "Set the PDF engines - all by default")
			fs.Bool("pdfengines-disable-routes", false, "Disable the routes")

			return fs
		}(),
		New: func() gotenberg.Module { return new(PDFEngines) },
	}
}

// Provision gets either all gotenberg.PDFEngine modules or the modules
// selected by the user thanks to the "engines" flag.
func (mod *PDFEngines) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	names := flags.MustStringSlice("pdfengines-engines")
	mod.disableRoutes = flags.MustBool("pdfengines-disable-routes")

	engines, err := ctx.Modules(new(gotenberg.PDFEngine))
	if err != nil {
		return fmt.Errorf("get PDF engines: %w", err)
	}

	mod.engines = make([]gotenberg.PDFEngine, len(engines))

	for i, engine := range engines {
		mod.engines[i] = engine.(gotenberg.PDFEngine)
	}

	if len(names) > 0 {
		// Selection from user.
		mod.names = names

		return nil
	}

	// No selection from user, use all PDF engines available.
	mod.names = make([]string, len(mod.engines))

	for i, engine := range mod.engines {
		mod.names[i] = engine.(gotenberg.Module).Descriptor().ID
	}

	return nil
}

// Validate validates there is at least one gotenberg.PDFEngine module
// available. It also validates that selected gotenberg.PDFEngine modules
// actually exist.
func (mod PDFEngines) Validate() error {
	if len(mod.engines) == 0 {
		return errors.New("no PDF engine")
	}

	availableEngines := make([]string, len(mod.engines))

	for i, engine := range mod.engines {
		availableEngines[i] = engine.(gotenberg.Module).Descriptor().ID
	}

	nonExistingEngines := make([]string, 0)

	for _, name := range mod.names {
		engineExists := false

		for _, engine := range mod.engines {
			if name == engine.(gotenberg.Module).Descriptor().ID {
				engineExists = true
				break
			}
		}

		if !engineExists {
			nonExistingEngines = append(nonExistingEngines, name)
		}
	}

	if len(nonExistingEngines) == 0 {
		return nil
	}

	return fmt.Errorf("non-existing PDF engine(s): %s - available PDF engine(s): %s", nonExistingEngines, availableEngines)
}

// PDFEngine returns a gotenberg.PDFEngine.
func (mod PDFEngines) PDFEngine() (gotenberg.PDFEngine, error) {
	engines := make([]gotenberg.PDFEngine, len(mod.engines))

	i := 0
	for _, engine := range mod.engines {
		engines[i] = engine
		i++
	}

	return newMultiPDFEngines(engines...), nil
}

// Routes returns the HTTP routes.
func (mod PDFEngines) Routes() ([]api.Route, error) {
	if mod.disableRoutes {
		return nil, nil
	}

	engine, err := mod.PDFEngine()
	if err != nil {
		// Should not happen, unless our provider implementation
		// changes in the future.
		return nil, fmt.Errorf("get pdf engine: %w", err)
	}

	return []api.Route{
		mergeRoute(engine),
		convertRoute(engine),
	}, nil
}

// Interface guards.
var (
	_ gotenberg.Module            = (*PDFEngines)(nil)
	_ gotenberg.Provisioner       = (*PDFEngines)(nil)
	_ gotenberg.Validator         = (*PDFEngines)(nil)
	_ gotenberg.PDFEngineProvider = (*PDFEngines)(nil)
	_ api.Router                  = (*PDFEngines)(nil)
)
