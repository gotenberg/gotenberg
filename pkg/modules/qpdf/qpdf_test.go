package qpdf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
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
	engine := new(QPdf)
	ctx := gotenberg.NewContext(gotenberg.ParsedFlags{}, nil)

	err := engine.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
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

func TestQPdf_Debug(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		engine      *QPdf
		expect      map[string]interface{}
		doNotExpect map[string]interface{}
	}{
		{
			scenario: "cannot determine version",
			engine: &QPdf{
				binPath: "foo",
			},
			expect: map[string]interface{}{
				"version": `exec: "foo": executable file not found in $PATH`,
			},
		},
		{
			scenario: "success",
			engine: &QPdf{
				binPath: "echo",
			},
			doNotExpect: map[string]interface{}{
				"version": `exec: "echo": executable file not found in $PATH`,
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			d := tc.engine.Debug()

			if tc.expect != nil {
				if !reflect.DeepEqual(d, tc.expect) {
					t.Errorf("expected '%v' but got '%v'", tc.expect, d)
				}
			}

			if tc.doNotExpect != nil {
				if reflect.DeepEqual(d, tc.doNotExpect) {
					t.Errorf("did not expect '%v'", d)
				}
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
			expectError: false,
		},
		{
			scenario: "many files success",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(QPdf)
			err := engine.Provision(nil)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
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

func TestQPdf_Split(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    context.Context
		mode                   gotenberg.SplitMode
		inputPath              string
		expectError            bool
		expectedError          error
		expectOutputPathsCount int
		expectOutputPaths      []string
	}{
		{
			scenario:               "ErrPdfSplitModeNotSupported",
			expectError:            true,
			expectedError:          gotenberg.ErrPdfSplitModeNotSupported,
			expectOutputPathsCount: 0,
		},
		{
			scenario:               "ErrPdfSplitModeNotSupported (no unify with pages)",
			ctx:                    context.TODO(),
			mode:                   gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1", Unify: false},
			expectError:            true,
			expectedError:          gotenberg.ErrPdfSplitModeNotSupported,
			expectOutputPathsCount: 0,
		},
		{
			scenario:               "invalid context",
			ctx:                    nil,
			mode:                   gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1-2", Unify: true},
			expectError:            true,
			expectOutputPathsCount: 0,
		},
		{
			scenario:               "invalid input path",
			ctx:                    context.TODO(),
			mode:                   gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1-2", Unify: true},
			inputPath:              "",
			expectError:            true,
			expectOutputPathsCount: 0,
		},
		{
			scenario:               "success (pages & unify)",
			ctx:                    context.TODO(),
			mode:                   gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1-2", Unify: true},
			inputPath:              "/tests/test/testdata/pdfengines/sample1.pdf",
			expectError:            false,
			expectOutputPathsCount: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(QPdf)
			err := engine.Provision(nil)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
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

			outputPaths, err := engine.Split(tc.ctx, zap.NewNop(), tc.mode, tc.inputPath, outputDir)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v but got: %v", tc.expectedError, err)
			}

			if tc.expectOutputPathsCount != len(outputPaths) {
				t.Errorf("expected %d output paths but got %d", tc.expectOutputPathsCount, len(outputPaths))
			}
		})
	}
}

func TestQPdf_Flatten(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         context.Context
		inputPath   string
		createCopy  bool
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
			inputPath:   "foo.pdf",
			expectError: true,
		},
		{
			scenario:    "success",
			ctx:         context.TODO(),
			inputPath:   "/tests/test/testdata/pdfengines/sample3.pdf",
			createCopy:  true,
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(QPdf)
			err := engine.Provision(nil)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			var destinationPath string
			if tc.createCopy {
				fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
				outputDir, err := fs.MkdirAll()
				if err != nil {
					t.Fatalf("expected error no but got: %v", err)
				}

				defer func() {
					err = os.RemoveAll(fs.WorkingDirPath())
					if err != nil {
						t.Fatalf("expected no error while cleaning up but got: %v", err)
					}
				}()

				destinationPath = fmt.Sprintf("%s/copy_temp.pdf", outputDir)
				source, err := os.Open(tc.inputPath)
				if err != nil {
					t.Fatalf("open source file: %v", err)
				}

				defer func(source *os.File) {
					err := source.Close()
					if err != nil {
						t.Fatalf("close file: %v", err)
					}
				}(source)

				destination, err := os.Create(destinationPath)
				if err != nil {
					t.Fatalf("create destination file: %v", err)
				}

				defer func(destination *os.File) {
					err := destination.Close()
					if err != nil {
						t.Fatalf("close file: %v", err)
					}
				}(destination)

				_, err = io.Copy(destination, source)
				if err != nil {
					t.Fatalf("copy source into destination: %v", err)
				}
			} else {
				destinationPath = tc.inputPath
			}

			err = engine.Flatten(tc.ctx, zap.NewNop(), destinationPath)

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

func TestLibreOfficePdfEngine_ReadMetadata(t *testing.T) {
	engine := new(QPdf)
	_, err := engine.ReadMetadata(context.Background(), zap.NewNop(), "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}

func TestLibreOfficePdfEngine_WriteMetadata(t *testing.T) {
	engine := new(QPdf)
	err := engine.WriteMetadata(context.Background(), zap.NewNop(), nil, "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}
