package libreoffice

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/uno"
)

func TestLibreOffice_Descriptor(t *testing.T) {
	descriptor := LibreOffice{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(LibreOffice))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestLibreOffice_Provision(t *testing.T) {
	tests := []struct {
		name               string
		ctx                *gotenberg.Context
		expectProvisionErr bool
	}{
		{
			name: "nominal behavior",
			ctx: func() *gotenberg.Context {
				provider1 := struct {
					gotenberg.ModuleMock
					uno.ProviderMock
				}{}
				provider1.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider1
					}}
				}
				provider1.UNOMock = func() (uno.API, error) {
					return uno.APIMock{}, nil
				}

				provider2 := struct {
					gotenberg.ModuleMock
					gotenberg.PDFEngineProviderMock
				}{}
				provider2.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module {
						return provider2
					}}
				}
				provider2.PDFEngineMock = func() (gotenberg.PDFEngine, error) {
					return gotenberg.PDFEngineMock{}, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider1.Descriptor(),
						provider2.Descriptor(),
					},
				)
			}(),
		},
		{
			name: "no UNO API provider",
			ctx: gotenberg.NewContext(
				gotenberg.ParsedFlags{
					FlagSet: new(LibreOffice).Descriptor().FlagSet,
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
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
			expectProvisionErr: true,
		},
		{
			name: "no PDF engine provider",
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
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
			expectProvisionErr: true,
		},
		{
			name: "no PDF engine from PDF engine provider",
			ctx: func() *gotenberg.Context {
				provider1 := struct {
					gotenberg.ModuleMock
					uno.ProviderMock
				}{}
				provider1.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider1
					}}
				}
				provider1.UNOMock = func() (uno.API, error) {
					return uno.APIMock{}, nil
				}

				provider2 := struct {
					gotenberg.ModuleMock
					gotenberg.PDFEngineProviderMock
				}{}
				provider2.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module {
						return provider2
					}}
				}
				provider2.PDFEngineMock = func() (gotenberg.PDFEngine, error) {
					return gotenberg.PDFEngineMock{}, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider1.Descriptor(),
						provider2.Descriptor(),
					},
				)
			}(),
			expectProvisionErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mod := new(LibreOffice)
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

func TestLibreOffice_Routes(t *testing.T) {
	tests := []struct {
		name              string
		mod               LibreOffice
		expectRoutesCount int
	}{
		{
			name:              "route not disabled",
			mod:               LibreOffice{},
			expectRoutesCount: 1,
		},
		{
			name: "route disabled",
			mod: LibreOffice{
				disableRoutes: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			routes, err := tc.mod.Routes()
			if err != nil {
				t.Fatalf("expected no error from mod.Routes(), but got: %v", err)
			}

			if tc.expectRoutesCount != len(routes) {
				t.Errorf("expected %d routes from mod.Routes(), but got %d", tc.expectRoutesCount, len(routes))
			}
		})
	}
}
