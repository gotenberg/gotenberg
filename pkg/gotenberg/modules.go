package gotenberg

import (
	"context"
	"fmt"
	"sort"
	"sync"

	flag "github.com/spf13/pflag"
)

// Module is a sort of plugin which adds new functionalities to the application
// or other modules.
//
//	type YourModule struct {
//		property string
//	}
//
//	func (YourModule) Descriptor() gotenberg.ModuleDescriptor {
//		return gotenberg.ModuleDescriptor{
//			ID: "your_module",
//			FlagSet: func() *flag.FlagSet {
//				fs := flag.NewFlagSet("your_module", flag.ExitOnError)
//				fs.String("your_module-property", "default value", "flag description")
//
//				return fs
//			}(),
//			New: func() gotenberg.Module { return new(YourModule) },
//		}
//	}
type Module interface {
	Descriptor() ModuleDescriptor
}

// ModuleDescriptor describes your module for the application.
type ModuleDescriptor struct {
	// ID is the unique name (snake case) of the module.
	// Required.
	ID string

	// FlagSet is the definition of the flags of the module.
	// Optional.
	FlagSet *flag.FlagSet

	// New returns a new and empty instance of the module's type.
	// Required.
	New func() Module
}

// Provisioner is a module interface for modules which have to be initialized
// according to flags, environment variables, the context, etc.
type Provisioner interface {
	Provision(*Context) error
}

// Validator is a module interface for modules which have to be validated after
// provisioning.
type Validator interface {
	Validate() error
}

// App is a module interface for modules which can be started or stopped by the
// application.
type App interface {
	Start() error
	// StartupMessage returns a custom message to display on startup. If it
	// returns an empty string, a default startup message is used instead.
	StartupMessage() string
	Stop(ctx context.Context) error
}

// SystemLogger is a module interface for modules which want to display
// messages on startup.
type SystemLogger interface {
	SystemMessages() []string
}

// MustRegisterModule registers a module.
//
// To register a module, create an init() method in the module main go file:
//
//	func init() {
//		gotenberg.MustRegisterModule(YourModule{})
//	}
//
// Then, in the main command (github.com/gotenberg/gotenberg/v8/cmd/gotenberg),
// import the module:
//
//	imports (
//		// Gotenberg modules.
//		_ "your_module_path"
//	)
func MustRegisterModule(mod Module) {
	desc := mod.Descriptor()

	if desc.ID == "" {
		panic("module with an empty ID cannot be registered")
	}

	if desc.New == nil {
		panic("module New function cannot be nil")
	}

	if val := desc.New(); val == nil {
		panic("module New function cannot return a nil instance")
	}

	descriptorsMu.Lock()
	defer descriptorsMu.Unlock()

	if _, ok := descriptors[desc.ID]; ok {
		panic(fmt.Sprintf("module %s is already registered", desc.ID))
	}

	descriptors[desc.ID] = desc
}

// GetModuleDescriptors returns the descriptors of all registered modules.
func GetModuleDescriptors() []ModuleDescriptor {
	descriptorsMu.RLock()
	defer descriptorsMu.RUnlock()

	mods := make([]ModuleDescriptor, len(descriptors))
	i := 0

	for _, desc := range descriptors {
		mods[i] = desc
		i++
	}

	sort.Slice(mods, func(i, j int) bool {
		return mods[i].ID < mods[j].ID
	})

	return mods
}

var (
	descriptors   = make(map[string]ModuleDescriptor)
	descriptorsMu sync.RWMutex
)
