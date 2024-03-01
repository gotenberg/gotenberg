package libreoffice

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	libreofficeapi "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
)

func TestLibreOffice_Descriptor(t *testing.T) {
	descriptor := new(LibreOffice).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(LibreOffice))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestLibreOffice_Provision(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         *gotenberg.Context
		expectError bool
	}{
		{
			scenario: "no LibreOffice API provider",
			ctx: func() *gotenberg.Context {
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOffice).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "no LibreOffice API from LibreOffice API provider",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					libreofficeapi.ProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.LibreOfficeMock = func() (libreofficeapi.Uno, error) {
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
			expectError: true,
		},
		{
			scenario: "no PDF engine provider",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					libreofficeapi.ProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.LibreOfficeMock = func() (libreofficeapi.Uno, error) {
					return new(libreofficeapi.ApiMock), nil
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
			expectError: true,
		},
		{
			scenario: "no PDF engine from PDF engine provider",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					libreofficeapi.ProviderMock
					gotenberg.PdfEngineProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.LibreOfficeMock = func() (libreofficeapi.Uno, error) {
					return new(libreofficeapi.ApiMock), nil
				}
				mod.PdfEngineMock = func() (gotenberg.PdfEngine, error) {
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
			expectError: true,
		},
		{
			scenario: "provision success",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					libreofficeapi.ProviderMock
					gotenberg.PdfEngineProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.LibreOfficeMock = func() (libreofficeapi.Uno, error) {
					return new(libreofficeapi.ApiMock), nil
				}
				mod.PdfEngineMock = func() (gotenberg.PdfEngine, error) {
					return new(gotenberg.PdfEngineMock), nil
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
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(LibreOffice)
			err := mod.Provision(tc.ctx)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestLibreOffice_Routes(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		expectRoutes  int
		disableRoutes bool
	}{
		{
			scenario:      "routes not disabled",
			expectRoutes:  1,
			disableRoutes: false,
		},
		{
			scenario:      "routes disabled",
			expectRoutes:  0,
			disableRoutes: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(LibreOffice)
			mod.disableRoutes = tc.disableRoutes

			routes, err := mod.Routes()
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectRoutes != len(routes) {
				t.Errorf("expected %d routes but got %d", tc.expectRoutes, len(routes))
			}
		})
	}
}
