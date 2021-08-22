package gotenberg

import (
	"reflect"
	"testing"
)

type ProtoModule struct {
	descriptor func() ModuleDescriptor
}

func (mod ProtoModule) Descriptor() ModuleDescriptor {
	return mod.descriptor()
}

type ProtoProvisioner struct {
	ProtoModule
	provision func(ctx *Context) error
}

func (mod ProtoProvisioner) Provision(ctx *Context) error {
	return mod.provision(ctx)
}

type ProtoValidator struct {
	ProtoModule
	validate func() error
}

func (mod ProtoValidator) Validate() error {
	return mod.validate()
}

func TestMustRegisterModule(t *testing.T) {
	descriptorsMu.RLock()
	descriptors = map[string]ModuleDescriptor{
		"a": {ID: "a"},
	}
	descriptorsMu.RUnlock()

	for i, tc := range []struct {
		ID          string
		New         func() Module
		expectPanic bool
	}{
		{
			ID:          "",
			New:         func() Module { return new(ProtoModule) },
			expectPanic: true,
		},
		{
			ID:          "b",
			New:         nil,
			expectPanic: true,
		},
		{
			ID:          "b",
			New:         func() Module { return nil },
			expectPanic: true,
		},
		{
			ID:          "a",
			New:         func() Module { return new(ProtoModule) },
			expectPanic: true,
		},
		{
			ID:  "b",
			New: func() Module { return new(ProtoModule) },
		},
	} {
		func() {
			mod := struct{ ProtoModule }{}
			mod.descriptor = func() ModuleDescriptor { return ModuleDescriptor{ID: tc.ID, New: tc.New} }

			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			MustRegisterModule(mod)
		}()
	}

	descriptorsMu.RLock()
	descriptors = make(map[string]ModuleDescriptor)
	descriptorsMu.RUnlock()
}

func TestGetModuleDescriptors(t *testing.T) {
	descriptorsMu.RLock()
	descriptors = map[string]ModuleDescriptor{
		"d": {ID: "d"},
		"c": {ID: "c"},
		"b": {ID: "b"},
		"a": {ID: "a"},
	}
	descriptorsMu.RUnlock()

	expect := []ModuleDescriptor{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
		{ID: "d"},
	}

	actual := GetModuleDescriptors()

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %v but got %v", expect, actual)
	}

	descriptorsMu.RLock()
	descriptors = make(map[string]ModuleDescriptor)
	descriptorsMu.RUnlock()
}

// Interface guards.
var (
	_ Module      = (*ProtoModule)(nil)
	_ Provisioner = (*ProtoProvisioner)(nil)
	_ Module      = (*ProtoProvisioner)(nil)
	_ Validator   = (*ProtoValidator)(nil)
	_ Module      = (*ProtoValidator)(nil)
)
