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
	mergeNames         []string
	splitNames         []string
	flattenNames       []string
	convertNames       []string
	readMetadataNames  []string
	writeMetadataNames []string
	engines            []gotenberg.PdfEngine
	disableRoutes      bool
}

// Descriptor returns a PdfEngines' module descriptor.
func (mod *PdfEngines) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "pdfengines",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("pdfengines", flag.ExitOnError)
			fs.StringSlice("pdfengines-merge-engines", []string{"qpdf", "pdfcpu", "pdftk"}, "Set the PDF engines and their order for the merge feature - empty means all")
			fs.StringSlice("pdfengines-split-engines", []string{"pdfcpu", "qpdf", "pdftk"}, "Set the PDF engines and their order for the split feature - empty means all")
			fs.StringSlice("pdfengines-flatten-engines", []string{"qpdf"}, "Set the PDF engines and their order for the flatten feature - empty means all")
			fs.StringSlice("pdfengines-convert-engines", []string{"libreoffice-pdfengine"}, "Set the PDF engines and their order for the convert feature - empty means all")
			fs.StringSlice("pdfengines-read-metadata-engines", []string{"exiftool"}, "Set the PDF engines and their order for the read metadata feature - empty means all")
			fs.StringSlice("pdfengines-write-metadata-engines", []string{"exiftool"}, "Set the PDF engines and their order for the write metadata feature - empty means all")
			fs.Bool("pdfengines-disable-routes", false, "Disable the routes")

			// Deprecated flags.
			fs.StringSlice("pdfengines-engines", make([]string, 0), "Set the default PDF engines and their default order - all by default")
			err := fs.MarkDeprecated("pdfengines-engines", "use other flags for a more granular selection of PDF engines per method")
			if err != nil {
				panic(err)
			}

			return fs
		}(),
		New: func() gotenberg.Module { return new(PdfEngines) },
	}
}

// Provision gets either all [gotenberg.PdfEngine] modules or the modules
// selected by the user thanks to the "engines" flag.
func (mod *PdfEngines) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mergeNames := flags.MustStringSlice("pdfengines-merge-engines")
	splitNames := flags.MustStringSlice("pdfengines-split-engines")
	flattenNames := flags.MustStringSlice("pdfengines-flatten-engines")
	convertNames := flags.MustStringSlice("pdfengines-convert-engines")
	readMetadataNames := flags.MustStringSlice("pdfengines-read-metadata-engines")
	writeMetadataNames := flags.MustStringSlice("pdfengines-write-metadata-engines")
	mod.disableRoutes = flags.MustBool("pdfengines-disable-routes")

	engines, err := ctx.Modules(new(gotenberg.PdfEngine))
	if err != nil {
		return fmt.Errorf("get PDF engines: %w", err)
	}

	mod.engines = make([]gotenberg.PdfEngine, len(engines))

	for i, engine := range engines {
		mod.engines[i] = engine.(gotenberg.PdfEngine)
	}

	defaultNames := make([]string, len(mod.engines))
	for i, engine := range mod.engines {
		defaultNames[i] = engine.(gotenberg.Module).Descriptor().ID
	}

	// Example in the case of deprecated module name.
	//for i, name := range defaultNames {
	//	if name == "unoconv-pdfengine" || name == "uno-pdfengine" {
	//		logger.Warn(fmt.Sprintf("%s is deprecated; prefer libreoffice-pdfengine instead", name))
	//		mod.defaultNames[i] = "libreoffice-pdfengine"
	//	}
	//}

	mod.mergeNames = defaultNames
	if len(mergeNames) > 0 {
		mod.mergeNames = mergeNames
	}

	mod.splitNames = defaultNames
	if len(splitNames) > 0 {
		mod.splitNames = splitNames
	}

	mod.flattenNames = defaultNames
	if len(flattenNames) > 0 {
		mod.flattenNames = flattenNames
	}

	mod.convertNames = defaultNames
	if len(convertNames) > 0 {
		mod.convertNames = convertNames
	}

	mod.readMetadataNames = defaultNames
	if len(readMetadataNames) > 0 {
		mod.readMetadataNames = readMetadataNames
	}

	mod.writeMetadataNames = defaultNames
	if len(writeMetadataNames) > 0 {
		mod.writeMetadataNames = writeMetadataNames
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
	findNonExistingEngines := func(names []string) {
		for _, name := range names {
			engineExists := false

			for _, engine := range mod.engines {
				if name == engine.(gotenberg.Module).Descriptor().ID {
					engineExists = true
					break
				}
			}

			if engineExists {
				continue
			}

			alreadyInSlice := false
			for _, engine := range nonExistingEngines {
				if engine == name {
					alreadyInSlice = true
					break
				}
			}

			if !alreadyInSlice {
				nonExistingEngines = append(nonExistingEngines, name)
			}
		}
	}

	findNonExistingEngines(mod.mergeNames)
	findNonExistingEngines(mod.splitNames)
	findNonExistingEngines(mod.flattenNames)
	findNonExistingEngines(mod.convertNames)
	findNonExistingEngines(mod.readMetadataNames)
	findNonExistingEngines(mod.writeMetadataNames)

	if len(nonExistingEngines) == 0 {
		return nil
	}

	return fmt.Errorf("non-existing PDF engine(s): %s - available PDF engine(s): %s", nonExistingEngines, availableEngines)
}

// SystemMessages returns one message with the selected [gotenberg.PdfEngine]
// modules.
func (mod *PdfEngines) SystemMessages() []string {
	return []string{
		fmt.Sprintf("merge engines - %s", strings.Join(mod.mergeNames[:], " ")),
		fmt.Sprintf("split engines - %s", strings.Join(mod.splitNames[:], " ")),
		fmt.Sprintf("flatten engines - %s", strings.Join(mod.flattenNames[:], " ")),
		fmt.Sprintf("convert engines - %s", strings.Join(mod.convertNames[:], " ")),
		fmt.Sprintf("read metadata engines - %s", strings.Join(mod.readMetadataNames[:], " ")),
		fmt.Sprintf("write metadata engines - %s", strings.Join(mod.writeMetadataNames[:], " ")),
	}
}

// PdfEngine returns a [gotenberg.PdfEngine].
func (mod *PdfEngines) PdfEngine() (gotenberg.PdfEngine, error) {
	engines := func(names []string) []gotenberg.PdfEngine {
		list := make([]gotenberg.PdfEngine, len(names))
		for i, name := range names {
			for _, engine := range mod.engines {
				if name == engine.(gotenberg.Module).Descriptor().ID {
					list[i] = engine
					break
				}
			}
		}

		return list
	}

	return newMultiPdfEngines(
		engines(mod.mergeNames),
		engines(mod.splitNames),
		engines(mod.flattenNames),
		engines(mod.convertNames),
		engines(mod.readMetadataNames),
		engines(mod.writeMetadataNames),
	), nil
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
		splitRoute(engine),
		flattenRoute(engine),
		convertRoute(engine),
		readMetadataRoute(engine),
		writeMetadataRoute(engine),
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
