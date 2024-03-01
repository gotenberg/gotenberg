package gotenberg

import (
	"fmt"
	"reflect"
)

// Context is a struct which helps to initialize modules. When provisioning, a
// module may use the context to get other modules that it needs internally.
type Context struct {
	flags           ParsedFlags
	descriptors     []ModuleDescriptor
	moduleInstances map[string]interface{}
}

// NewContext creates a [Context].
// In a module, prefer the [Provisioner] interface to get a [Context].
func NewContext(
	flags ParsedFlags,
	descriptors []ModuleDescriptor,
) *Context {
	return &Context{
		flags:           flags,
		descriptors:     descriptors,
		moduleInstances: make(map[string]interface{}),
	}
}

// ParsedFlags returns the parsed flags.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		flags := ctx.ParsedFlags()
//		m.foo = flags.RequiredString("foo")
//	}
func (ctx *Context) ParsedFlags() ParsedFlags {
	return ctx.flags
}

// Module returns a module which satisfies the requested interface.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		mod, _ := ctx.Module(new(ModuleInterface))
//		real := mod.(ModuleInterface)
//	}
//
// If the module has not yet been initialized, this method
// initializes it. Otherwise, returns the already initialized instance.
func (ctx *Context) Module(kind interface{}) (interface{}, error) {
	mods, err := ctx.Modules(kind)
	if err != nil {
		return nil, fmt.Errorf("get module: %w", err)
	}

	if len(mods) != 1 {
		return nil, fmt.Errorf("expected to have one and only one %s module", kind)
	}

	return mods[0], nil
}

// Modules returns the list of modules which satisfies the requested interface.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		mods, _ := ctx.Modules(new(ModuleInterface))
//		for _, mod := range mods {
//			real := mod.(ModuleInterface)
//			// ...
//		}
//	}
//
// If one or more modules have not yet been initialized, this method
// initializes them. Otherwise, returns the already initialized instances.
func (ctx *Context) Modules(kind interface{}) ([]interface{}, error) {
	realKind := reflect.TypeOf(kind).Elem()

	var mods []interface{}

	for _, desc := range ctx.descriptors {
		newInstance := desc.New()

		if ok := reflect.TypeOf(newInstance).Implements(realKind); ok {
			// The module implements the requested interface.
			// We check if it has already been initialized.
			instance, ok := ctx.moduleInstances[desc.ID]

			if ok {
				mods = append(mods, instance)
			} else {
				err := ctx.loadModule(desc.ID, newInstance)
				if err != nil {
					return nil, err
				}

				mods = append(mods, newInstance)
			}
		}
	}

	return mods, nil
}

// loadModule calls the Provision and/or Validate methods of the requested
// module if it satisfies the [Provisioner] and/or [Validator] interfaces.
func (ctx *Context) loadModule(id string, instance interface{}) error {
	if prov, ok := instance.(Provisioner); ok {
		// The instance can be provisioned.
		err := prov.Provision(ctx)
		if err != nil {
			return fmt.Errorf("provision module %s: %w", id, err)
		}
	}

	if validator, ok := instance.(Validator); ok {
		// The instance can be validated.
		err := validator.Validate()
		if err != nil {
			return fmt.Errorf("validate module %s: %w", id, err)
		}
	}

	ctx.moduleInstances[id] = instance

	return nil
}
