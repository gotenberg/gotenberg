package pdfengines

import (
	"errors"
	"fmt"
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
		scenario                        string
		ctx                             *gotenberg.Context
		expectedMergePdfEngines         []string
		expectedSplitPdfEngines         []string
		expectedFlattenPdfEngines       []string
		expectedConvertPdfEngines       []string
		expectedReadMetadataPdfEngines  []string
		expectedWriteMetadataPdfEngines []string
		expectError                     bool
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
			expectedMergePdfEngines:         []string{"qpdf", "pdfcpu", "pdftk"},
			expectedSplitPdfEngines:         []string{"pdfcpu", "qpdf", "pdftk"},
			expectedFlattenPdfEngines:       []string{"qpdf"},
			expectedConvertPdfEngines:       []string{"libreoffice-pdfengine"},
			expectedReadMetadataPdfEngines:  []string{"exiftool"},
			expectedWriteMetadataPdfEngines: []string{"exiftool"},
			expectError:                     false,
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
				err := fs.Parse([]string{"--pdfengines-merge-engines=b", "--pdfengines-split-engines=a", "--pdfengines-flatten-engines=c", "--pdfengines-convert-engines=b", "--pdfengines-read-metadata-engines=a", "--pdfengines-write-metadata-engines=a"})
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

			expectedMergePdfEngines:         []string{"b"},
			expectedSplitPdfEngines:         []string{"a"},
			expectedFlattenPdfEngines:       []string{"c"},
			expectedConvertPdfEngines:       []string{"b"},
			expectedReadMetadataPdfEngines:  []string{"a"},
			expectedWriteMetadataPdfEngines: []string{"a"},
			expectError:                     false,
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

			if len(tc.expectedMergePdfEngines) != len(mod.mergeNames) {
				t.Fatalf("expected %d merge names but got %d", len(tc.expectedMergePdfEngines), len(mod.mergeNames))
			}

			if len(tc.expectedFlattenPdfEngines) != len(mod.flattenNames) {
				t.Fatalf("expected %d flatten names but got %d", len(tc.expectedFlattenPdfEngines), len(mod.flattenNames))
			}

			if len(tc.expectedConvertPdfEngines) != len(mod.convertNames) {
				t.Fatalf("expected %d convert names but got %d", len(tc.expectedConvertPdfEngines), len(mod.convertNames))
			}

			if len(tc.expectedReadMetadataPdfEngines) != len(mod.readMetadataNames) {
				t.Fatalf("expected %d read metadata names but got %d", len(tc.expectedReadMetadataPdfEngines), len(mod.readMetadataNames))
			}

			if len(tc.expectedWriteMetadataPdfEngines) != len(mod.writeMetadataNames) {
				t.Fatalf("expected %d write metadata names but got %d", len(tc.expectedWriteMetadataPdfEngines), len(mod.writeMetadataNames))
			}

			for index, name := range mod.mergeNames {
				if name != tc.expectedMergePdfEngines[index] {
					t.Fatalf("expected merge name at index %d to be %s, but got: %s", index, name, tc.expectedMergePdfEngines[index])
				}
			}

			for index, name := range mod.splitNames {
				if name != tc.expectedSplitPdfEngines[index] {
					t.Fatalf("expected split name at index %d to be %s, but got: %s", index, name, tc.expectedSplitPdfEngines[index])
				}
			}

			for index, name := range mod.convertNames {
				if name != tc.expectedConvertPdfEngines[index] {
					t.Fatalf("expected convert name at index %d to be %s, but got: %s", index, name, tc.expectedConvertPdfEngines[index])
				}
			}

			for index, name := range mod.readMetadataNames {
				if name != tc.expectedReadMetadataPdfEngines[index] {
					t.Fatalf("expected read metadata name at index %d to be %s, but got: %s", index, name, tc.expectedReadMetadataPdfEngines[index])
				}
			}

			for index, name := range mod.writeMetadataNames {
				if name != tc.expectedWriteMetadataPdfEngines[index] {
					t.Fatalf("expected write metadat name at index %d to be %s, but got: %s", index, name, tc.expectedWriteMetadataPdfEngines[index])
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
				mergeNames:         tc.names,
				convertNames:       tc.names,
				readMetadataNames:  tc.names,
				writeMetadataNames: tc.names,
				engines:            tc.engines,
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
	mod.mergeNames = []string{"foo", "bar"}
	mod.splitNames = []string{"foo", "bar"}
	mod.convertNames = []string{"foo", "bar"}
	mod.readMetadataNames = []string{"foo", "bar"}
	mod.writeMetadataNames = []string{"foo", "bar"}

	expectedMessages := 6
	messages := mod.SystemMessages()
	if len(messages) != expectedMessages {
		t.Errorf("expected %d message(s), but got %d", expectedMessages, len(messages))
	}

	expect := []string{
		fmt.Sprintf("merge engines - %s", strings.Join(mod.mergeNames[:], " ")),
		fmt.Sprintf("split engines - %s", strings.Join(mod.splitNames[:], " ")),
		fmt.Sprintf("flatten engines - %s", strings.Join(mod.flattenNames[:], " ")),
		fmt.Sprintf("convert engines - %s", strings.Join(mod.convertNames[:], " ")),
		fmt.Sprintf("read metadata engines - %s", strings.Join(mod.readMetadataNames[:], " ")),
		fmt.Sprintf("write metadata engines - %s", strings.Join(mod.writeMetadataNames[:], " ")),
	}

	for i, message := range messages {
		if message != expect[i] {
			t.Errorf("expected message at index %d to be %s, but got %s", i, message, expect[i])
		}
	}
}

func TestPdfEngines_PdfEngine(t *testing.T) {
	mod := PdfEngines{
		mergeNames:         []string{"foo", "bar"},
		splitNames:         []string{"foo", "bar"},
		convertNames:       []string{"foo", "bar"},
		readMetadataNames:  []string{"foo", "bar"},
		writeMetadataNames: []string{"foo", "bar"},
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
			expectRoutes:  6,
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
