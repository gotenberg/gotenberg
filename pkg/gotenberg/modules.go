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
	Stop(ctx context.Context) error
}

// Ingress is an [App] that takes requests and hands their work to the other
// apps, such as the API. [StopApps] stops every Ingress before the other apps,
// so the work in flight completes while the apps it uses still run.
type Ingress interface {
	App
	Ingress()
}

// Reporter is a module interface for modules which describe their state in
// the startup summary, once every [App] has started.
type Reporter interface {
	Report() Report
}

// Report is a module's line in the startup summary.
type Report struct {
	// Name labels the line, such as "Chromium".
	Name string

	// Fields describe the module's state, in display order.
	Fields []ReportField
}

// ReportField is a fact of a [Report]. Key and Value form the attribute of the
// startup log record. Text, when set, replaces Value in the terminal summary,
// where it reads after the [Report] name.
type ReportField struct {
	Key   string
	Value any
	Text  string
}

// Debuggable is a module interface for modules which want to provide
// additional debug data.
type Debuggable interface {
	Debug() map[string]any
}

// MustRegisterModule registers a module.
//
// To register a module, create an init() method in the module main go file:
//
//	func init() {
//		gotenberg.MustRegisterModule(YourModule{})
//	}
//
// Then, in the main command (github.com/gotenberg/gotenberg/v9/cmd/gotenberg),
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
