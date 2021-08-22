package pdfengine

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/unoconv"
	flag "github.com/spf13/pflag"
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
	pdf func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options unoconv.Options) error
}

func (mod ProtoUnoconvAPI) PDF(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options unoconv.Options) error {
	return mod.pdf(ctx, logger, inputPath, outputPath, options)
}

func (mod ProtoUnoconvAPI) Extensions() []string {
	return nil
}

func TestUnoconvPDFEngine_Descriptor(t *testing.T) {
	descriptor := UnoconvPDFEngine{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(UnoconvPDFEngine))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestUnoconvPDFEngine_Provision(t *testing.T) {
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
						FlagSet: flag.NewFlagSet("foo", flag.ExitOnError),
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
						FlagSet: flag.NewFlagSet("foo", flag.ExitOnError),
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
						FlagSet: flag.NewFlagSet("foo", flag.ExitOnError),
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
		},
	} {
		mod := new(UnoconvPDFEngine)
		err := mod.Provision(tc.ctx)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestUnoconvPDFEngine_Merge(t *testing.T) {
	mod := new(UnoconvPDFEngine)
	err := mod.Merge(context.TODO(), zap.NewNop(), nil, "")

	if !errors.Is(err, gotenberg.ErrPDFEngineMethodNotAvailable) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPDFEngineMethodNotAvailable, err)
	}
}

func TestUnoconvPDFEngine_Convert(t *testing.T) {
	for i, tc := range []struct {
		api       unoconv.API
		format    string
		expectErr bool
	}{
		{
			format:    "",
			expectErr: true,
		},
		{
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, __ string, _ unoconv.Options) error {
					return errors.New("foo")
				}

				return unoconvAPI
			}(),
			format:    gotenberg.FormatPDFA1a,
			expectErr: true,
		},
		{
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, __ string, _ unoconv.Options) error {
					return nil
				}

				return unoconvAPI
			}(),
			format: gotenberg.FormatPDFA1a,
		},
	} {
		mod := new(UnoconvPDFEngine)
		mod.unoconv = tc.api

		err := mod.Convert(context.TODO(), zap.NewNop(), tc.format, "", "")

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

// Interface guards.
var (
	_ gotenberg.Module = (*ProtoModule)(nil)
	_ unoconv.Provider = (*ProtoUnoconvProvider)(nil)
	_ gotenberg.Module = (*ProtoUnoconvProvider)(nil)
	_ unoconv.API      = (*ProtoUnoconvAPI)(nil)
)
