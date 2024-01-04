package pdfengines

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestPdfEngines_Descriptor(t *testing.T) {
	descriptor := new(PdfEngines).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(PdfEngines))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestPdfEngines_Provision(t *testing.T) {
	for _, tc := range []struct {
		scenario           string
		ctx                *gotenberg.Context
		expectedPdfEngines []string
		expectError        bool
	}{
		{
			scenario: "no selection from user",
			ctx: func() *gotenberg.Context {
				provider := &struct {
					gotenberg.ModuleMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}

				engine := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.PdfEngineMock
				}{}
				engine.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return engine }}
				}
				engine.ValidateMock = func() error {
					return nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(PdfEngines).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
						engine.Descriptor(),
					},
				)
			}(),
			expectedPdfEngines: []string{"bar"},
			expectError:        false,
		},
		{
			scenario: "selection from user",
			ctx: func() *gotenberg.Context {
				provider := &struct {
					gotenberg.ModuleMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				engine1 := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.PdfEngineMock
				}{}
				engine1.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "a", New: func() gotenberg.Module { return engine1 }}
				}
				engine1.ValidateMock = func() error {
					return nil
				}

				engine2 := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.PdfEngineMock
				}{}
				engine2.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "b", New: func() gotenberg.Module { return engine2 }}
				}
				engine2.ValidateMock = func() error {
					return nil
				}

				fs := new(PdfEngines).Descriptor().FlagSet
				err := fs.Parse([]string{"--pdfengines-engines=b", "--pdfengines-engines=a"})
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
						engine1.Descriptor(),
						engine2.Descriptor(),
					},
				)
			}(),
			expectedPdfEngines: []string{"b", "a"},
			expectError:        false,
		},
		{
			scenario: "no valid PDF engine",
			ctx: func() *gotenberg.Context {
				provider := &struct {
					gotenberg.ModuleMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				engine := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.PdfEngineMock
				}{}
				engine.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return engine }}
				}
				engine.ValidateMock = func() error {
					return errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(PdfEngines).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
						engine.Descriptor(),
					},
				)
			}(),
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(PdfEngines)
			err := mod.Provision(tc.ctx)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if len(tc.expectedPdfEngines) != len(mod.names) {
				t.Fatalf("expected %d names but got %d", len(tc.expectedPdfEngines), len(mod.names))
			}

			for index, name := range mod.names {
				if name != tc.expectedPdfEngines[index] {
					t.Fatalf("expected scenario at index %d to be %s, but got: %s", index, name, tc.expectedPdfEngines[index])
				}
			}
		})
	}
}

func TestPdfEngines_Validate(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		names       []string
		engines     []gotenberg.PdfEngine
		expectError bool
	}{
		{
			scenario: "existing PDF engine",
			names:    []string{"foo"},
			engines: func() []gotenberg.PdfEngine {
				engine := &struct {
					gotenberg.ModuleMock
					gotenberg.PdfEngineMock
				}{}
				engine.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine }}
				}

				return []gotenberg.PdfEngine{
					engine,
				}
			}(),
			expectError: false,
		},
		{
			scenario: "non-existing bar PDF engine",
			names:    []string{"foo", "bar", "baz"},
			engines: func() []gotenberg.PdfEngine {
				engine1 := &struct {
					gotenberg.ModuleMock
					gotenberg.PdfEngineMock
				}{}
				engine1.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine1 }}
				}

				engine2 := &struct {
					gotenberg.ModuleMock
					gotenberg.PdfEngineMock
				}{}
				engine2.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "baz", New: func() gotenberg.Module { return engine2 }}
				}

				return []gotenberg.PdfEngine{
					engine1,
					engine2,
				}
			}(),
			expectError: true,
		},
		{
			scenario:    "no PDF engine",
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := PdfEngines{
				names:   tc.names,
				engines: tc.engines,
			}

			err := mod.Validate()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestPdfEngines_SystemMessages(t *testing.T) {
	mod := new(PdfEngines)
	mod.names = []string{"foo", "bar"}

	messages := mod.SystemMessages()
	if len(messages) != 1 {
		t.Errorf("expected one and only one message, but got %d", len(messages))
	}

	expect := strings.Join(mod.names[:], " ")
	if messages[0] != expect {
		t.Errorf("expected message '%s', but got '%s'", expect, messages[0])
	}
}

func TestPdfEngines_PdfEngine(t *testing.T) {
	mod := PdfEngines{
		names: []string{"foo", "bar"},
		engines: func() []gotenberg.PdfEngine {
			engine1 := &struct {
				gotenberg.ModuleMock
				gotenberg.PdfEngineMock
			}{}
			engine1.DescriptorMock = func() gotenberg.ModuleDescriptor {
				return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine1 }}
			}

			engine2 := &struct {
				gotenberg.ModuleMock
				gotenberg.PdfEngineMock
			}{}
			engine2.DescriptorMock = func() gotenberg.ModuleDescriptor {
				return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return engine2 }}
			}

			return []gotenberg.PdfEngine{
				engine1,
				engine2,
			}
		}(),
	}

	_, err := mod.PdfEngine()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestPdfEngines_Routes(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		expectRoutes  int
		disableRoutes bool
	}{
		{
			scenario:      "routes not disabled",
			expectRoutes:  2,
			disableRoutes: false,
		},
		{
			scenario:      "routes disabled",
			expectRoutes:  0,
			disableRoutes: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(PdfEngines)
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
