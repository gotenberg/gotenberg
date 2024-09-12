package qpdf

import (
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
)

func TestQPdf_Descriptor(t *testing.T) {
	descriptor := new(QPdf).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(QPdf))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestQPdf_Provision(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         *gotenberg.Context
		expectError bool
	}{
		{
			scenario: "no LibreOffice API provider",
			ctx: gotenberg.NewContext(
				gotenberg.ParsedFlags{
					FlagSet: new(QPdf).Descriptor().FlagSet,
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
						FlagSet: new(QPdf).Descriptor().FlagSet,
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
						FlagSet: new(QPdf).Descriptor().FlagSet,
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
			engine := new(QPdf)
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

func TestQPdf_Validate(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		binPath     string
		expectError bool
	}{
		{
			scenario:    "empty bin path",
			binPath:     "",
			expectError: true,
		},
		{
			scenario:    "bin path does not exist",
			binPath:     "/foo",
			expectError: true,
		},
		{
			scenario:    "validate success",
			binPath:     os.Getenv("QPDF_BIN_PATH"),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(QPdf)
			engine.binPath = tc.binPath
			err := engine.Validate()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestQPdf_Merge(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         context.Context
		inputPaths  []string
		expectError bool
	}{
		{
			scenario:    "invalid context",
			ctx:         nil,
			expectError: true,
		},
		{
			scenario: "invalid input path",
			ctx:      context.TODO(),
			inputPaths: []string{
				"foo",
			},
			expectError: true,
		},
		{
			scenario: "single file success",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
			},
		},
		{
			scenario: "many files success",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ctx := func() *gotenberg.Context {
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
						FlagSet: new(QPdf).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}()

			engine := new(QPdf)
			err := engine.Provision(ctx)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			fs := gotenberg.NewFileSystem()
			outputDir, err := fs.MkdirAll()
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			defer func() {
				err = os.RemoveAll(fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up but got: %v", err)
				}
			}()

			err = engine.Merge(tc.ctx, zap.NewNop(), tc.inputPaths, outputDir+"/foo.pdf")

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestQPdf_Convert(t *testing.T) {
	engine := new(QPdf)
	err := engine.Convert(context.TODO(), zap.NewNop(), gotenberg.PdfFormats{}, "", "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}

func TestQPdf_optimizeImages(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		libreoffice api.Uno
		expectError bool
	}{
		{
			scenario: "error from LibreOffice",
			libreoffice: func() api.Uno {
				lo := new(api.ApiMock)
				lo.PdfMock = func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options api.Options) error {
					return errors.New("foo")
				}
				return lo
			}(),
			expectError: true,
		},
		{
			scenario: "success",
			libreoffice: func() api.Uno {
				lo := new(api.ApiMock)
				lo.PdfMock = func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options api.Options) error {
					return nil
				}
				return lo
			}(),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ctx := func() *gotenberg.Context {
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
					return tc.libreoffice, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(QPdf).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}()

			engine := new(QPdf)
			err := engine.Provision(ctx)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			err = engine.optimizeImages(context.TODO(), zap.NewNop(), 50, 75, "foo.pdf", "bar.pdf")

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestQPdf_linearize(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         context.Context
		inputPath   string
		expectError bool
	}{
		{
			scenario:    "invalid context",
			ctx:         nil,
			expectError: true,
		},
		{
			scenario:    "invalid input path",
			ctx:         context.TODO(),
			inputPath:   "foo",
			expectError: true,
		},
		{
			scenario:  "single file success",
			ctx:       context.TODO(),
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ctx := func() *gotenberg.Context {
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
						FlagSet: new(QPdf).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}()

			engine := new(QPdf)
			err := engine.Provision(ctx)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			fs := gotenberg.NewFileSystem()
			outputDir, err := fs.MkdirAll()
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			defer func() {
				err = os.RemoveAll(fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up but got: %v", err)
				}
			}()

			err = engine.linearize(tc.ctx, zap.NewNop(), tc.inputPath, outputDir+"/foo.pdf")

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestQPdf_compressStreams(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         context.Context
		inputPath   string
		expectError bool
	}{
		{
			scenario:    "invalid context",
			ctx:         nil,
			expectError: true,
		},
		{
			scenario:    "invalid input path",
			ctx:         context.TODO(),
			inputPath:   "foo",
			expectError: true,
		},
		{
			scenario:  "single file success",
			ctx:       context.TODO(),
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ctx := func() *gotenberg.Context {
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
						FlagSet: new(QPdf).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}()

			engine := new(QPdf)
			err := engine.Provision(ctx)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			fs := gotenberg.NewFileSystem()
			outputDir, err := fs.MkdirAll()
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			defer func() {
				err = os.RemoveAll(fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up but got: %v", err)
				}
			}()

			err = engine.compressStreams(tc.ctx, zap.NewNop(), tc.inputPath, outputDir+"/foo.pdf")

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestQPdf_Optimize(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         context.Context
		libreoffice api.Uno
		options     gotenberg.OptimizeOptions
		inputPath   string
		expectError bool
	}{
		{
			scenario:    "no images optimizations nor streams compression",
			ctx:         context.TODO(),
			options:     gotenberg.OptimizeOptions{CompressStreams: false, ImageQuality: 0, MaxImageResolution: 0},
			expectError: false,
		},
		{
			scenario: "cannot optimize images with LibreOffice",
			ctx:      context.TODO(),
			libreoffice: func() api.Uno {
				lo := new(api.ApiMock)
				lo.PdfMock = func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options api.Options) error {
					return errors.New("foo")
				}
				return lo
			}(),
			options:     gotenberg.OptimizeOptions{CompressStreams: false, ImageQuality: 50, MaxImageResolution: 75},
			expectError: true,
		},
		{
			scenario:    "cannot linearize with QPDF",
			ctx:         nil,
			libreoffice: new(api.ApiMock),
			options:     gotenberg.OptimizeOptions{CompressStreams: true, ImageQuality: 0, MaxImageResolution: 0},
			expectError: true,
		},
		{
			scenario: "success: only optimize images with LibreOffice",
			ctx:      context.TODO(),
			libreoffice: func() api.Uno {
				lo := new(api.ApiMock)
				lo.PdfMock = func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options api.Options) error {
					return nil
				}
				return lo
			}(),
			options:     gotenberg.OptimizeOptions{CompressStreams: false, ImageQuality: 50, MaxImageResolution: 75},
			expectError: false,
		},
		{
			scenario:    "success: only compress steams with QPDF",
			ctx:         context.TODO(),
			libreoffice: new(api.ApiMock),
			options:     gotenberg.OptimizeOptions{CompressStreams: true, ImageQuality: 0, MaxImageResolution: 0},
			inputPath:   "/tests/test/testdata/pdfengines/sample1.pdf",
			expectError: false,
		},
		{
			scenario: "success: all",
			ctx:      context.TODO(),
			libreoffice: func() api.Uno {
				lo := new(api.ApiMock)
				lo.PdfMock = func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options api.Options) error {
					err := os.Rename(inputPath, outputPath)
					if err != nil {
						t.Fatalf("expected no error while renaming %q to %q but got: %v", inputPath, outputPath, err)
					}
					return nil
				}
				return lo
			}(),
			options:     gotenberg.OptimizeOptions{CompressStreams: true, ImageQuality: 50, MaxImageResolution: 75},
			inputPath:   "/tests/test/testdata/pdfengines/sample1.pdf",
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ctx := func() *gotenberg.Context {
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
					return tc.libreoffice, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(QPdf).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}()

			engine := new(QPdf)
			err := engine.Provision(ctx)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			fs := gotenberg.NewFileSystem()
			outputDir, err := fs.MkdirAll()
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			defer func() {
				err = os.RemoveAll(fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up but got: %v", err)
				}
			}()

			inputPath := tc.inputPath
			if tc.inputPath != "" {
				inputPath = outputDir + "/bar.pdf"
				in, err := os.Open(tc.inputPath)
				if err != nil {
					t.Fatalf("expected no error while copying %q to %q but got: %v", tc.inputPath, inputPath, err)
				}

				defer func() {
					err := in.Close()
					if err != nil {
						t.Fatalf("expected no error while copying %q to %q but got: %v", tc.inputPath, inputPath, err)
					}
				}()

				out, err := os.Create(inputPath)
				if err != nil {
					t.Fatalf("expected no error while copying %q to %q but got: %v", tc.inputPath, inputPath, err)
				}

				defer func() {
					err := out.Close()
					if err != nil {
						t.Fatalf("expected no error while copying %q to %q but got: %v", tc.inputPath, inputPath, err)
					}
				}()

				_, err = io.Copy(out, in)
				if err != nil {
					t.Fatalf("expected no error while copying %q to %q but got: %v", tc.inputPath, inputPath, err)
				}
			}

			err = engine.Optimize(tc.ctx, zap.NewNop(), tc.options, inputPath, outputDir+"/foo.pdf")

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestQPdf_ReadMetadata(t *testing.T) {
	engine := new(QPdf)
	_, err := engine.ReadMetadata(context.Background(), zap.NewNop(), "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}

func TestQPdf_WriteMetadata(t *testing.T) {
	engine := new(QPdf)
	err := engine.WriteMetadata(context.Background(), zap.NewNop(), nil, "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}
