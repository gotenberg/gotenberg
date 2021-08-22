package libreoffice

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/unoconv"
	"go.uber.org/zap"
)

type ProtoModule struct {
	descriptor func() gotenberg.ModuleDescriptor
}

func (mod ProtoModule) Descriptor() gotenberg.ModuleDescriptor {
	return mod.descriptor()
}

type ProtoUnoconvProvider struct {
	ProtoModule
	unoconv func() (unoconv.API, error)
}

func (mod ProtoUnoconvProvider) Unoconv() (unoconv.API, error) {
	return mod.unoconv()
}

type ProtoUnoconvAPI struct {
	pdf        func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error
	extensions func() []string
}

func (mod ProtoUnoconvAPI) PDF(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options unoconv.Options) error {
	return mod.pdf(ctx, logger, inputPath, outputPath, options)
}

func (mod ProtoUnoconvAPI) Extensions() []string {
	return mod.extensions()
}

type ProtoPDFEngineProvider struct {
	ProtoModule
	pdfEngine func() (gotenberg.PDFEngine, error)
}

func (mod ProtoPDFEngineProvider) PDFEngine() (gotenberg.PDFEngine, error) {
	return mod.pdfEngine()
}

type ProtoPDFEngine struct {
	merge   func(_ context.Context, _ *zap.Logger, _ []string, _ string) error
	convert func(_ context.Context, _ *zap.Logger, _, _, _ string) error
}

func (mod ProtoPDFEngine) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	return mod.merge(ctx, logger, inputPaths, outputPath)
}

func (mod ProtoPDFEngine) Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
	return mod.convert(ctx, logger, format, inputPath, outputPath)
}

func TestLibreOffice_Descriptor(t *testing.T) {
	descriptor := LibreOffice{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(LibreOffice))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestLibreOffice_Provision(t *testing.T) {
	for i, tc := range []struct {
		ctx       *gotenberg.Context
		expectErr bool
	}{
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoModule }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoUnoconvProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.unoconv = func() (unoconv.API, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoUnoconvProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.unoconv = func() (unoconv.API, error) {
					return struct{ ProtoUnoconvAPI }{}, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod1 := struct{ ProtoUnoconvProvider }{}
				mod1.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod1 }}
				}
				mod1.unoconv = func() (unoconv.API, error) {
					return struct{ ProtoUnoconvAPI }{}, nil
				}

				mod2 := struct{ ProtoPDFEngineProvider }{}
				mod2.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod2 }}
				}
				mod2.pdfEngine = func() (gotenberg.PDFEngine, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod1.Descriptor(),
						mod2.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod1 := struct{ ProtoUnoconvProvider }{}
				mod1.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod1 }}
				}
				mod1.unoconv = func() (unoconv.API, error) {
					return struct{ ProtoUnoconvAPI }{}, nil
				}

				mod2 := struct{ ProtoPDFEngineProvider }{}
				mod2.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod2 }}
				}
				mod2.pdfEngine = func() (gotenberg.PDFEngine, error) {
					return struct{ ProtoPDFEngine }{}, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod1.Descriptor(),
						mod2.Descriptor(),
					},
				)
			}(),
		},
	} {
		mod := new(LibreOffice)
		err := mod.Provision(tc.ctx)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestLibreOffice_Routes(t *testing.T) {
	for i, tc := range []struct {
		expectRoutes  int
		disableRoutes bool
	}{
		{
			expectRoutes: 1,
		},
		{
			disableRoutes: true,
		},
	} {
		mod := new(LibreOffice)
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
	_ gotenberg.Module            = (*ProtoModule)(nil)
	_ unoconv.Provider            = (*ProtoUnoconvProvider)(nil)
	_ gotenberg.Module            = (*ProtoUnoconvProvider)(nil)
	_ unoconv.API                 = (*ProtoUnoconvAPI)(nil)
	_ gotenberg.PDFEngineProvider = (*ProtoPDFEngineProvider)(nil)
	_ gotenberg.Module            = (*ProtoPDFEngineProvider)(nil)
	_ gotenberg.PDFEngine         = (*ProtoPDFEngine)(nil)
)
