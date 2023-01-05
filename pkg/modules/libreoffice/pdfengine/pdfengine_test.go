package pdfengine

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/uno"
	"go.uber.org/zap"
)

func TestUNO_Descriptor(t *testing.T) {
	descriptor := UNO{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(UNO))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestUNO_Provider(t *testing.T) {
	tests := []struct {
		name               string
		ctx                *gotenberg.Context
		expectProvisionErr bool
	}{
		{
			name: "nominal behavior",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					uno.ProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.UNOMock = func() (uno.API, error) {
					return uno.APIMock{}, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(UNO).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
		},
		{
			name: "no UNO API provider",
			ctx: gotenberg.NewContext(
				gotenberg.ParsedFlags{
					FlagSet: new(UNO).Descriptor().FlagSet,
				},
				[]gotenberg.ModuleDescriptor{},
			),
			expectProvisionErr: true,
		},
		{
			name: "no API from UNO API provider",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					uno.ProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.UNOMock = func() (uno.API, error) {
					return uno.APIMock{}, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(UNO).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
			expectProvisionErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mod := new(UNO)
			err := mod.Provision(tc.ctx)

			if tc.expectProvisionErr && err == nil {
				t.Error("expected mod.Provision() error, but got none")
			}

			if !tc.expectProvisionErr && err != nil {
				t.Errorf("expected no error from mod.Provision(), but got: %v", err)
			}
		})
	}
}

func TestUNO_Merge(t *testing.T) {
	mod := new(UNO)
	err := mod.Merge(context.Background(), zap.NewNop(), nil, "")

	if !errors.Is(err, gotenberg.ErrPDFEngineMethodNotAvailable) {
		t.Errorf("expected error %v from mod.Merge(), but got: %v", gotenberg.ErrPDFEngineMethodNotAvailable, err)
	}
}

func TestUNO_Convert(t *testing.T) {
	tests := []struct {
		name             string
		mod              UNO
		expectConvertErr bool
	}{
		{
			name: "nominal behavior",
			mod: UNO{
				unoAPI: uno.APIMock{
					ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
						return nil
					},
				},
			},
		},
		{
			name: "invalid PDF format",
			mod: UNO{
				unoAPI: uno.APIMock{
					ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
						return uno.ErrInvalidPDFformat
					},
				},
			},
			expectConvertErr: true,
		},
		{
			name: "convert fail",
			mod: UNO{
				unoAPI: uno.APIMock{
					ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
						return errors.New("foo")
					},
				},
			},
			expectConvertErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mod.Convert(context.Background(), zap.NewNop(), "", "", "")

			if tc.expectConvertErr && err == nil {
				t.Errorf("expected mod.Convert() error, but got none")
			}

			if !tc.expectConvertErr && err != nil {
				t.Fatalf("expected no error from mod.Convert(), but got: %v", err)
			}
		})
	}
}
