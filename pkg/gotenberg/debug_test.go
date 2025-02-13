package gotenberg

import (
	"reflect"
	"runtime"
	"testing"

	flag "github.com/spf13/pflag"
)

func TestBuildDebug(t *testing.T) {
	if !reflect.DeepEqual(Debug(), DebugInfo{}) {
		t.Errorf("Debug() should return empty debug data")
	}

	fs := flag.NewFlagSet("gotenberg", flag.ExitOnError)
	fs.String("foo", "bar", "Set foo")
	ctx := NewContext(ParsedFlags{
		FlagSet: fs,
	}, func() []ModuleDescriptor {
		mod1 := &struct {
			ModuleMock
		}{}
		mod1.DescriptorMock = func() ModuleDescriptor {
			return ModuleDescriptor{ID: "foo", New: func() Module { return mod1 }}
		}
		mod2 := &struct {
			ModuleMock
			DebuggableMock
		}{}
		mod2.DescriptorMock = func() ModuleDescriptor {
			return ModuleDescriptor{ID: "bar", New: func() Module { return mod2 }}
		}
		mod2.DebugMock = func() map[string]interface{} {
			return map[string]interface{}{
				"foo": "bar",
			}
		}

		return []ModuleDescriptor{mod1.Descriptor(), mod2.Descriptor()}
	}())

	// Load modules.
	_, err := ctx.Modules(new(Module))
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	// Build debug data.
	BuildDebug(ctx)

	expect := DebugInfo{
		Version:      Version,
		Architecture: runtime.GOARCH,
		Modules: []string{
			"bar",
			"foo",
		},
		ModulesAdditionalData: map[string]map[string]interface{}{
			"bar": {
				"foo": "bar",
			},
		},
		Flags: map[string]interface{}{
			"foo": "bar",
		},
	}

	if !reflect.DeepEqual(expect, Debug()) {
		t.Errorf("expected '%+v', bug got '%+v'", expect, Debug())
	}
}
