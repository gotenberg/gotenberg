package pdfengine

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
)

func TestLibreOfficePdfEngine_Descriptor(t *testing.T) {
	descriptor := new(LibreOfficePdfEngine).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(LibreOfficePdfEngine))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestLibreOfficePdfEngine_Provider(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         *gotenberg.Context
		expectError bool
	}{
		{
			scenario: "no LibreOffice API provider",
			ctx: gotenberg.NewContext(
				gotenberg.ParsedFlags{
					FlagSet: new(LibreOfficePdfEngine).Descriptor().FlagSet,
				},
				[]gotenberg.ModuleDescriptor{},
			),
			expectError: true,
		},
		{
			scenario: "no API from LibreOffice API provider",
			ctx: func() *gotenberg.Context {
				provider := &struct {
					gotenberg.ModuleMock
					api.ProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LibreOfficeMock = func() (api.Uno, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOfficePdfEngine).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "provision success",
			ctx: func() *gotenberg.Context {
				provider := &struct {
					gotenberg.ModuleMock
					api.ProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LibreOfficeMock = func() (api.Uno, error) {
					return new(api.ApiMock), nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(LibreOfficePdfEngine).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(LibreOfficePdfEngine)
			err := engine.Provision(tc.ctx)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestLibreOfficePdfEngine_Merge(t *testing.T) {
	engine := new(LibreOfficePdfEngine)
	err := engine.Merge(context.Background(), zap.NewNop(), nil, "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}

func TestLibreOfficePdfEngine_Convert(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		api         api.Uno
		expectError bool
	}{
		{
			scenario: "convert success",
			api: &api.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options api.Options) error {
					return nil
				},
			},
			expectError: false,
		},
		{
			scenario: "invalid PDF format",
			api: &api.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options api.Options) error {
					return api.ErrInvalidPdfFormats
				},
			},
			expectError: true,
		},
		{
			scenario: "convert fail",
			api: &api.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options api.Options) error {
					return errors.New("foo")
				},
			},
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := &LibreOfficePdfEngine{unoApi: tc.api}
			err := engine.Convert(context.Background(), zap.NewNop(), gotenberg.PdfFormats{}, "", "")

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}
