package pdfengines

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

func TestPDFEngines_Descriptor(t *testing.T) {
	descriptor := PDFEngines{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(PDFEngines))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestPDFEngines_Provision(t *testing.T) {
	tests := []struct {
		name                 string
		ctx                  *gotenberg.Context
		expectPDFEngineNames []string
		expectProvisionErr   bool
	}{
		{
			name: "no selection from user",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}

				engine := struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.PDFEngineMock
				}{}
				engine.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return engine }}
				}
				engine.ValidateMock = func() error {
					return nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(PDFEngines).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
						engine.Descriptor(),
					},
				)
			}(),
			expectPDFEngineNames: []string{"bar"},
		},
		{
			name: "selection from user",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}

				engine1 := struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.PDFEngineMock
				}{}
				engine1.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "a", New: func() gotenberg.Module { return engine1 }}
				}
				engine1.ValidateMock = func() error {
					return nil
				}

				engine2 := struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.PDFEngineMock
				}{}
				engine2.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "b", New: func() gotenberg.Module { return engine2 }}
				}
				engine2.ValidateMock = func() error {
					return nil
				}

				fs := new(PDFEngines).Descriptor().FlagSet
				err := fs.Parse([]string{"--pdfengines-engines=b", "--pdfengines-engines=a"})

				if err != nil {
					t.Fatalf("expected no error from fs.Parse(), but got: %v", err)
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
			expectPDFEngineNames: []string{"b", "a"},
		},
		{
			name: "user select deprecated unoconv-pdfengine",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}

				engine := struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.PDFEngineMock
				}{}
				engine.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "uno-pdfengine", New: func() gotenberg.Module { return engine }}
				}
				engine.ValidateMock = func() error {
					return nil
				}

				fs := new(PDFEngines).Descriptor().FlagSet
				err := fs.Parse([]string{"--pdfengines-engines=unoconv-pdfengine"})

				if err != nil {
					t.Fatalf("expected no error from fs.Parse(), but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
						engine.Descriptor(),
					},
				)
			}(),
			expectPDFEngineNames: []string{"uno-pdfengine"},
		},
		{
			name: "no logger provider",
			ctx: func() *gotenberg.Context {
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(PDFEngines).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{},
				)
			}(),
			expectProvisionErr: true,
		},
		{
			name: "no logger from logger provider",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(PDFEngines).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
			expectProvisionErr: true,
		},
		{
			name: "no valid PDF engines",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}

				engine := struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.PDFEngineMock
				}{}
				engine.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return engine }}
				}
				engine.ValidateMock = func() error {
					return errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(PDFEngines).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
						engine.Descriptor(),
					},
				)
			}(),
			expectProvisionErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mod := new(PDFEngines)
			err := mod.Provision(tc.ctx)

			if tc.expectProvisionErr && err == nil {
				t.Fatal("expected mod.Provision() error, but got none")
			}

			if !tc.expectProvisionErr && err != nil {
				t.Fatalf("expected no error from mod.Provision(), but got: %v", err)
			}

			if len(tc.expectPDFEngineNames) != len(mod.names) {
				t.Errorf("expected %d names but got %d", len(tc.expectPDFEngineNames), len(mod.names))
			}

			for index, name := range mod.names {
				if name != tc.expectPDFEngineNames[index] {
					t.Errorf("expected name at index %d to be %s, but got: %s", index, name, tc.expectPDFEngineNames[index])
				}
			}
		})
	}
}

func TestPDFEngines_Validate(t *testing.T) {
	tests := []struct {
		name              string
		names             []string
		engines           []gotenberg.PDFEngine
		expectValidateErr bool
	}{
		{
			name:  "existing PDF engine",
			names: []string{"foo"},
			engines: func() []gotenberg.PDFEngine {
				engine := struct {
					gotenberg.ModuleMock
					gotenberg.PDFEngineMock
				}{}
				engine.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine }}
				}

				return []gotenberg.PDFEngine{
					engine,
				}
			}(),
		},
		{
			name:  "non-existing bar PDF engine",
			names: []string{"foo", "bar", "baz"},
			engines: func() []gotenberg.PDFEngine {
				engine1 := struct {
					gotenberg.ModuleMock
					gotenberg.PDFEngineMock
				}{}
				engine1.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine1 }}
				}

				engine2 := struct {
					gotenberg.ModuleMock
					gotenberg.PDFEngineMock
				}{}
				engine2.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "baz", New: func() gotenberg.Module { return engine2 }}
				}

				return []gotenberg.PDFEngine{
					engine1,
					engine2,
				}
			}(),
			expectValidateErr: true,
		},
		{
			name:              "no PDF engine",
			expectValidateErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mod := PDFEngines{
				names:   tc.names,
				engines: tc.engines,
			}

			err := mod.Validate()

			if tc.expectValidateErr && err == nil {
				t.Errorf("expected mod.Validate() error, but got none")
			}

			if !tc.expectValidateErr && err != nil {
				t.Errorf("expected no error from mod.Validate(), but got: %v", err)
			}
		})
	}
}

func TestPDFEngines_SystemMessages(t *testing.T) {
	mod := new(PDFEngines)
	mod.names = []string{"foo", "bar"}

	messages := mod.SystemMessages()
	if len(messages) != 1 {
		t.Errorf("expected one and only one message from mod.SystemMessages(), but got %d", len(messages))
	}

	expect := strings.Join(mod.names[:], " ")
	if messages[0] != expect {
		t.Errorf("expected message '%s' from mod.SystemMessages(), but got '%s'", expect, messages[0])
	}
}

func TestPDFEngines_PDFEngine(t *testing.T) {
	mod := PDFEngines{
		names: []string{"foo", "bar"},
		engines: func() []gotenberg.PDFEngine {
			engine1 := struct {
				gotenberg.ModuleMock
				gotenberg.PDFEngineMock
			}{}
			engine1.DescriptorMock = func() gotenberg.ModuleDescriptor {
				return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return engine1 }}
			}

			engine2 := struct {
				gotenberg.ModuleMock
				gotenberg.PDFEngineMock
			}{}
			engine2.DescriptorMock = func() gotenberg.ModuleDescriptor {
				return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return engine2 }}
			}

			return []gotenberg.PDFEngine{
				engine1,
				engine2,
			}
		}(),
	}

	_, err := mod.PDFEngine()
	if err != nil {
		t.Errorf("expected no error from mod.PDFEngine, but got: %v", err)
	}
}

func TestPDFEngines_Routes(t *testing.T) {
	tests := []struct {
		name              string
		mod               PDFEngines
		expectRoutesCount int
	}{
		{
			name: "route not disabled",
			mod: PDFEngines{
				engines: []gotenberg.PDFEngine{
					gotenberg.PDFEngineMock{},
				},
			},
			expectRoutesCount: 2,
		},
		{
			name: "route disabled",
			mod: PDFEngines{
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
