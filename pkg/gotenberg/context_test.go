package gotenberg

import (
	"errors"
	"testing"
)

func TestNewContext(t *testing.T) {
	if NewContext(ParsedFlags{}, nil) == nil {
		t.Error("expected a non-nil value")
	}
}

func TestContext_ParsedFlags(t *testing.T) {
	ctx := NewContext(ParsedFlags{}, nil)

	actual := ctx.ParsedFlags()
	expect := ParsedFlags{}

	if actual != expect {
		t.Errorf("expected %v but got %v", expect, actual)
	}
}

func TestContext_Module(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		mods        []ModuleDescriptor
		kind        interface{}
		expectError bool
	}{
		{
			scenario: "module with error on provision",
			mods: func() []ModuleDescriptor {
				mod := &struct {
					ModuleMock
					ProvisionerMock
				}{}
				mod.DescriptorMock = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.ProvisionMock = func(ctx *Context) error { return errors.New("foo") }
				return []ModuleDescriptor{mod.Descriptor()}
			}(),
			kind:        new(Provisioner),
			expectError: true,
		},
		{
			scenario: "two modules instead of one",
			mods: func() []ModuleDescriptor {
				mod := &struct {
					ModuleMock
					ProvisionerMock
				}{}
				mod.DescriptorMock = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.ProvisionMock = func(ctx *Context) error { return nil }
				return []ModuleDescriptor{mod.Descriptor(), mod.Descriptor()}
			}(),
			kind:        new(Provisioner),
			expectError: true,
		},
		{
			scenario: "success",
			mods: func() []ModuleDescriptor {
				mod := &struct {
					ModuleMock
					ProvisionerMock
				}{}
				mod.DescriptorMock = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.ProvisionMock = func(ctx *Context) error { return nil }
				return []ModuleDescriptor{mod.Descriptor()}
			}(),
			kind:        new(Provisioner),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ctx := NewContext(ParsedFlags{}, tc.mods)
			_, err := ctx.Module(tc.kind)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestContext_Modules(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		mods        []ModuleDescriptor
		kind        interface{}
		expectError bool
	}{
		{
			scenario: "module with error on provision",
			mods: func() []ModuleDescriptor {
				mod := &struct {
					ModuleMock
					ProvisionerMock
				}{}
				mod.DescriptorMock = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.ProvisionMock = func(ctx *Context) error { return errors.New("foo") }
				return []ModuleDescriptor{mod.Descriptor()}
			}(),
			kind:        new(Provisioner),
			expectError: true,
		},
		{
			scenario: "success (module)",
			mods: func() []ModuleDescriptor {
				mod := &struct {
					ModuleMock
					ProvisionerMock
				}{}
				mod.DescriptorMock = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.ProvisionMock = func(ctx *Context) error { return nil }
				return []ModuleDescriptor{mod.Descriptor(), mod.Descriptor()}
			}(),
			kind:        new(Provisioner),
			expectError: false,
		},
		{
			scenario: "success (one module)",
			mods: func() []ModuleDescriptor {
				mod := &struct {
					ModuleMock
					ProvisionerMock
				}{}
				mod.DescriptorMock = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.ProvisionMock = func(ctx *Context) error { return nil }

				return []ModuleDescriptor{mod.Descriptor()}
			}(),
			kind:        new(Provisioner),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ctx := NewContext(ParsedFlags{}, tc.mods)
			_, err := ctx.Modules(tc.kind)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestContext_loadModule(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		instance    interface{}
		expectError bool
	}{
		{
			scenario: "module with error on provision",
			instance: func() interface{} {
				mod := &struct {
					ModuleMock
					ProvisionerMock
				}{}
				mod.DescriptorMock = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.ProvisionMock = func(ctx *Context) error { return errors.New("foo") }
				return mod
			}(),
			expectError: true,
		},
		{
			scenario: "module with error on validation",
			instance: func() interface{} {
				mod := &struct {
					ModuleMock
					ValidatorMock
				}{}
				mod.DescriptorMock = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.ValidateMock = func() error { return errors.New("foo") }
				return mod
			}(),
			expectError: true,
		},
		{
			scenario: "success",
			instance: func() interface{} {
				mod := &struct {
					ModuleMock
					ValidatorMock
				}{}
				mod.DescriptorMock = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.ValidateMock = func() error { return nil }

				return mod
			}(),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ctx := NewContext(ParsedFlags{}, nil)
			err := ctx.loadModule("foo", tc.instance)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}
