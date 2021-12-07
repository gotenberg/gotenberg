package pdfengines

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

type ProtoModule struct {
	descriptor func() gotenberg.ModuleDescriptor
}

func (mod ProtoModule) Descriptor() gotenberg.ModuleDescriptor {
	return mod.descriptor()
}

type ProtoValidator struct {
	ProtoModule
	validate func() error
}

func (mod ProtoValidator) Validate() error {
	return mod.validate()
}

type ProtoPDFEngine struct {
	ProtoValidator
	merge   func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error
	convert func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error
}

func (mod ProtoPDFEngine) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	return mod.merge(ctx, logger, inputPaths, outputPath)
}

func (mod ProtoPDFEngine) Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
	return mod.convert(ctx, logger, format, inputPath, outputPath)
}

func TestPDFEngine_Descriptor(t *testing.T) {
	descriptor := PDFEngines{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(PDFEngines))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestPDFEngine_Provision(t *testing.T) {
	for i, tc := range []struct {
		ctx                *gotenberg.Context
		expectNames        []string
		expectEnginesCount int
		expectErr          bool
	}{
		{
			ctx: func() *gotenberg.Context {
				engine := struct{ ProtoPDFEngine }{}
				engine.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine }}
				}
				engine.validate = func() error { return errors.New("foo") }

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(PDFEngines).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						engine.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				engine := struct{ ProtoPDFEngine }{}
				engine.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine }}
				}
				engine.validate = func() error { return nil }

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(PDFEngines).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						engine.Descriptor(),
					},
				)
			}(),
			expectNames:        []string{"foo"},
			expectEnginesCount: 1,
		},
		{
			ctx: func() *gotenberg.Context {
				engine1 := struct{ ProtoPDFEngine }{}
				engine1.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "a", New: func() gotenberg.Module { return engine1 }}
				}
				engine1.validate = func() error { return nil }

				engine2 := struct{ ProtoPDFEngine }{}
				engine2.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "b", New: func() gotenberg.Module { return engine2 }}
				}
				engine2.validate = func() error { return nil }

				fs := new(PDFEngines).Descriptor().FlagSet
				err := fs.Parse([]string{"--pdfengines-engines=b", "--pdfengines-engines=a"})

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					[]gotenberg.ModuleDescriptor{
						engine1.Descriptor(),
						engine2.Descriptor(),
					},
				)
			}(),
			expectNames:        []string{"b", "a"},
			expectEnginesCount: 2,
		},
		{
			ctx: func() *gotenberg.Context {
				engine1 := struct{ ProtoPDFEngine }{}
				engine1.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "a", New: func() gotenberg.Module { return engine1 }}
				}
				engine1.validate = func() error { return nil }

				engine2 := struct{ ProtoPDFEngine }{}
				engine2.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "b", New: func() gotenberg.Module { return engine2 }}
				}
				engine2.validate = func() error { return nil }

				fs := new(PDFEngines).Descriptor().FlagSet
				err := fs.Parse([]string{"--pdfengines-engines=b"})

				if err != nil {
					t.Fatalf("expected error but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					[]gotenberg.ModuleDescriptor{
						engine1.Descriptor(),
						engine2.Descriptor(),
					},
				)
			}(),
			expectNames:        []string{"b"},
			expectEnginesCount: 2,
		},
	} {
		mod := new(PDFEngines)
		err := mod.Provision(tc.ctx)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		if len(tc.expectNames) != len(mod.names) {
			t.Errorf("test %d: expected %d names but got %d", i, len(tc.expectNames), len(mod.names))
		}

		if tc.expectEnginesCount != len(mod.engines) {
			t.Errorf("test %d: expected %d engines but got %d", i, tc.expectEnginesCount, len(mod.engines))
		}

		for index, name := range mod.names {
			if name != tc.expectNames[index] {
				t.Errorf("test %d: expected name at index %d to be %s, but got: %s", i, index, name, tc.expectNames[index])
			}
		}
	}
}

func TestPDFEngine_Validate(t *testing.T) {
	for i, tc := range []struct {
		names     []string
		engines   []gotenberg.PDFEngine
		expectErr bool
	}{
		{
			expectErr: true,
		},
		{
			names: []string{"foo"},
			engines: func() []gotenberg.PDFEngine {
				engine := struct{ ProtoPDFEngine }{}
				engine.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine }}
				}

				return []gotenberg.PDFEngine{
					engine,
				}
			}(),
		},
		{
			names: []string{"foo", "bar", "baz"},
			engines: func() []gotenberg.PDFEngine {
				engine1 := struct{ ProtoPDFEngine }{}
				engine1.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine1 }}
				}

				engine2 := struct{ ProtoPDFEngine }{}
				engine2.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "baz", New: func() gotenberg.Module { return engine2 }}
				}

				return []gotenberg.PDFEngine{
					engine1,
					engine2,
				}
			}(),
			expectErr: true,
		},
	} {
		mod := new(PDFEngines)
		mod.names = tc.names
		mod.engines = tc.engines
		err := mod.Validate()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestPDFEngines_SystemMessages(t *testing.T) {
	mod := new(PDFEngines)
	mod.names = []string{"foo", "bar"}

	messages := mod.SystemMessages()
	if len(messages) != 1 {
		t.Errorf("expected one and only one message but got %d", len(messages))
	}

	expect := strings.Join(mod.names[:], " ")
	if messages[0] != expect {
		t.Errorf("expected message '%s' but got '%s'", expect, messages[0])
	}
}

func TestPDFEngine_PDFEngine(t *testing.T) {
	mod := new(PDFEngines)
	mod.names = []string{"foo", "bar"}
	mod.engines = func() []gotenberg.PDFEngine {
		engine1 := struct{ ProtoPDFEngine }{}
		engine1.descriptor = func() gotenberg.ModuleDescriptor {
			return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine1 }}
		}

		engine2 := struct{ ProtoPDFEngine }{}
		engine2.descriptor = func() gotenberg.ModuleDescriptor {
			return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return engine2 }}
		}

		return []gotenberg.PDFEngine{
			engine1,
			engine2,
		}
	}()

	_, err := mod.PDFEngine()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestPDFEngine_Routes(t *testing.T) {
	for i, tc := range []struct {
		expectRoutes  int
		disableRoutes bool
	}{
		{
			expectRoutes: 2,
		},
		{
			disableRoutes: true,
		},
	} {
		mod := new(PDFEngines)
		mod.engines = []gotenberg.PDFEngine{
			struct{ ProtoPDFEngine }{},
		}
		mod.disableRoutes = tc.disableRoutes

		routes, err := mod.Routes()
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		if tc.expectRoutes != len(routes) {
			t.Errorf("test %d: expected %d routes but got %d", i, tc.expectRoutes, len(routes))
		}
	}
}

// Interface guards.
var (
	_ gotenberg.Module    = (*ProtoModule)(nil)
	_ gotenberg.Validator = (*ProtoValidator)(nil)
	_ gotenberg.Module    = (*ProtoValidator)(nil)
	_ gotenberg.PDFEngine = (*ProtoPDFEngine)(nil)
	_ gotenberg.Module    = (*ProtoPDFEngine)(nil)
	_ gotenberg.Validator = (*ProtoPDFEngine)(nil)
)
