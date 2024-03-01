package pdfengines

import (
	"errors"
	"fmt"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(PdfEngines))
}

// PdfEngines acts as an aggregator and manager for multiple PDF engine
// modules. It enables the selection and ordering of PDF engines based on user
// preferences passed via command-line flags. The [PdfEngines] module also
// implements the [gotenberg.PdfEngine] interface, providing a unified approach
// to PDF processing across the various engines it manages.
//
// When processing PDFs, [PdfEngines] will attempt to use the engines in the
// order they were defined. If the primary engine encounters an error,
// [PdfEngines] can fall back to the next available engine. It also implements
// the [api.Router] interface to expose relevant PDF processing routes if
// enabled.
type PdfEngines struct {
	names         []string
	engines       []gotenberg.PdfEngine
	disableRoutes bool
}

// Descriptor returns a PdfEngines' module descriptor.
func (mod *PdfEngines) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "pdfengines",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("pdfengines", flag.ExitOnError)
			fs.StringSlice("pdfengines-engines", make([]string, 0), "Set the PDF engines and their order - all by default")
			fs.Bool("pdfengines-disable-routes", false, "Disable the routes")

			return fs
		}(),
		New: func() gotenberg.Module { return new(PdfEngines) },
	}
}

// Provision gets either all [gotenberg.PdfEngine] modules or the modules
// selected by the user thanks to the "engines" flag.
func (mod *PdfEngines) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	names := flags.MustStringSlice("pdfengines-engines")
	mod.disableRoutes = flags.MustBool("pdfengines-disable-routes")

	engines, err := ctx.Modules(new(gotenberg.PdfEngine))
	if err != nil {
		return fmt.Errorf("get PDF engines: %w", err)
	}

	mod.engines = make([]gotenberg.PdfEngine, len(engines))

	for i, engine := range engines {
		mod.engines[i] = engine.(gotenberg.PdfEngine)
	}

	if len(names) > 0 {
		// Selection from user.
		mod.names = names

		// Example in case of deprecated module name.
		//for i, name := range names {
		//	if name == "unoconv-pdfengine" || name == "uno-pdfengine" {
		//		logger.Warn(fmt.Sprintf("%s is deprecated; prefer libreoffice-pdfengine instead", name))
		//		mod.names[i] = "libreoffice-pdfengine"
		//	}
		//}

		return nil
	}

	// No selection from user, use all PDF engines available.
	mod.names = make([]string, len(mod.engines))

	for i, engine := range mod.engines {
		mod.names[i] = engine.(gotenberg.Module).Descriptor().ID
	}

	return nil
}

// Validate validates there is at least one [gotenberg.PdfEngine] module
// available. It also validates that selected [gotenberg.PdfEngine] modules
// actually exist.
func (mod *PdfEngines) Validate() error {
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

// SystemMessages returns one message with the selected [gotenberg.PdfEngine]
// modules.
func (mod *PdfEngines) SystemMessages() []string {
	return []string{
		strings.Join(mod.names[:], " "),
	}
}

// PdfEngine returns a [gotenberg.PdfEngine].
func (mod *PdfEngines) PdfEngine() (gotenberg.PdfEngine, error) {
	engines := make([]gotenberg.PdfEngine, len(mod.names))

	for i, name := range mod.names {
		for _, engine := range mod.engines {
			if name == engine.(gotenberg.Module).Descriptor().ID {
				engines[i] = engine
				break
			}
		}
	}

	return newMultiPdfEngines(engines...), nil
}

// Routes returns the HTTP routes.
func (mod *PdfEngines) Routes() ([]api.Route, error) {
	if mod.disableRoutes {
		return nil, nil
	}

	engine, err := mod.PdfEngine()
	if err != nil {
		// Should not happen, unless our provider implementation
		// changes in the future.
		return nil, fmt.Errorf("get pdf mod: %w", err)
	}

	return []api.Route{
		mergeRoute(engine),
		convertRoute(engine),
	}, nil
}

// Interface guards.
var (
	_ gotenberg.Module            = (*PdfEngines)(nil)
	_ gotenberg.Provisioner       = (*PdfEngines)(nil)
	_ gotenberg.Validator         = (*PdfEngines)(nil)
	_ gotenberg.SystemLogger      = (*PdfEngines)(nil)
	_ gotenberg.PdfEngineProvider = (*PdfEngines)(nil)
	_ api.Router                  = (*PdfEngines)(nil)
)
