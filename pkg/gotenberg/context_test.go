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
	for i, tc := range []struct {
		mods      []ModuleDescriptor
		kind      interface{}
		expectErr bool
	}{
		{
			mods: func() []ModuleDescriptor {
				mod := struct{ ProtoProvisioner }{}
				mod.descriptor = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.provision = func(ctx *Context) error { return errors.New("foo") }

				return []ModuleDescriptor{mod.Descriptor()}
			}(),
			kind:      new(Provisioner),
			expectErr: true,
		},
		{
			mods: func() []ModuleDescriptor {
				mod := struct{ ProtoProvisioner }{}
				mod.descriptor = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.provision = func(ctx *Context) error { return nil }

				return []ModuleDescriptor{mod.Descriptor(), mod.Descriptor()}
			}(),
			kind:      new(Provisioner),
			expectErr: true,
		},
		{
			mods: func() []ModuleDescriptor {
				mod := struct{ ProtoProvisioner }{}
				mod.descriptor = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.provision = func(ctx *Context) error { return nil }

				return []ModuleDescriptor{mod.Descriptor()}
			}(),
			kind: new(Provisioner),
		},
	} {

		ctx := NewContext(ParsedFlags{}, tc.mods)
		_, err := ctx.Module(tc.kind)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestContext_Modules(t *testing.T) {
	for i, tc := range []struct {
		mods      []ModuleDescriptor
		kind      interface{}
		expectErr bool
	}{
		{
			mods: func() []ModuleDescriptor {
				mod := struct{ ProtoProvisioner }{}
				mod.descriptor = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.provision = func(ctx *Context) error { return errors.New("foo") }

				return []ModuleDescriptor{mod.Descriptor()}
			}(),
			kind:      new(Provisioner),
			expectErr: true,
		},
		{
			mods: func() []ModuleDescriptor {
				mod := struct{ ProtoProvisioner }{}
				mod.descriptor = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.provision = func(ctx *Context) error { return nil }

				return []ModuleDescriptor{mod.Descriptor(), mod.Descriptor()}
			}(),
			kind: new(Provisioner),
		},
		{
			mods: func() []ModuleDescriptor {
				mod := struct{ ProtoProvisioner }{}
				mod.descriptor = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.provision = func(ctx *Context) error { return nil }

				return []ModuleDescriptor{mod.Descriptor()}
			}(),
			kind: new(Provisioner),
		},
	} {

		ctx := NewContext(ParsedFlags{}, tc.mods)
		_, err := ctx.Modules(tc.kind)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestContext_loadModule(t *testing.T) {
	for i, tc := range []struct {
		instance  interface{}
		expectErr bool
	}{
		{
			instance: func() interface{} {
				mod := struct{ ProtoProvisioner }{}
				mod.descriptor = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.provision = func(ctx *Context) error { return errors.New("foo") }

				return mod
			}(),
			expectErr: true,
		},
		{
			instance: func() interface{} {
				mod := struct{ ProtoValidator }{}
				mod.descriptor = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.validate = func() error { return errors.New("foo") }

				return mod
			}(),
			expectErr: true,
		},
		{
			instance: func() interface{} {
				mod := struct{ ProtoValidator }{}
				mod.descriptor = func() ModuleDescriptor {
					return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
				}
				mod.validate = func() error { return nil }

				return mod
			}(),
		},
	} {

		ctx := NewContext(ParsedFlags{}, nil)
		err := ctx.loadModule("foo", tc.instance)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}
