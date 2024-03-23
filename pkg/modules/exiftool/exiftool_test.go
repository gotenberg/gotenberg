package exiftool

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

func TestExifTool_Descriptor(t *testing.T) {
	descriptor := new(ExifTool).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(ExifTool))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestExifTool_Provision(t *testing.T) {
	engine := new(ExifTool)
	ctx := gotenberg.NewContext(gotenberg.ParsedFlags{}, nil)

	err := engine.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestExifTool_Validate(t *testing.T) {
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
			binPath:     os.Getenv("EXIFTOOL_BIN_PATH"),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(ExifTool)
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

func TestExiftool_Merge(t *testing.T) {
	engine := new(ExifTool)
	err := engine.Merge(context.Background(), zap.NewNop(), nil, "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}

func TestExiftool_Convert(t *testing.T) {
	engine := new(ExifTool)
	err := engine.Convert(context.Background(), zap.NewNop(), gotenberg.PdfFormats{}, "", "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}

func TestExiftool_ReadMetadata(t *testing.T) {
	for _, tc := range []struct {
		scenario       string
		inputPath      string
		expectMetadata map[string]interface{}
		expectError    bool
	}{
		{
			scenario:       "invalid input path",
			inputPath:      "foo",
			expectMetadata: nil,
			expectError:    true,
		},
		{
			scenario:  "success",
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
			expectMetadata: map[string]interface{}{
				"FileName":          "sample1.pdf",
				"FileTypeExtension": "pdf",
				"MIMEType":          "application/pdf",
				"PDFVersion":        1.4,
				"PageCount":         float64(3),
			},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(ExifTool)
			err := engine.Provision(nil)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			metadata, err := engine.ReadMetadata(context.Background(), zap.NewNop(), tc.inputPath)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectMetadata != nil && err == nil {
				for k, v := range tc.expectMetadata {
					if v2, ok := metadata[k]; !ok || v != v2 {
						t.Errorf("expected entry %s with value %v to exists", k, v)
					}
				}
			}
		})
	}
}

func TestExiftool_WriteMetadata(t *testing.T) {
	for _, tc := range []struct {
		scenario       string
		createCopy     bool
		inputPath      string
		metadata       map[string]interface{}
		expectMetadata map[string]interface{}
		expectError    bool
		expectedError  error
	}{
		{
			scenario:    "invalid input path",
			createCopy:  false,
			inputPath:   "foo",
			expectError: true,
		},
		{
			scenario:   "gotenberg.ErrPdfEngineMetadataValueNotSupported",
			createCopy: true,
			inputPath:  "/tests/test/testdata/pdfengines/sample1.pdf",
			metadata: map[string]interface{}{
				"Unsupported": map[string]interface{}{},
			},
			expectError:   true,
			expectedError: gotenberg.ErrPdfEngineMetadataValueNotSupported,
		},
		{
			scenario:   "success",
			createCopy: true,
			inputPath:  "/tests/test/testdata/pdfengines/sample1.pdf",
			metadata: map[string]interface{}{
				"Author":       "Julien Neuhart",
				"Copyright":    "Julien Neuhart",
				"CreationDate": "2006-09-18T16:27:50-04:00",
				"Creator":      "Gotenberg",
				"Keywords": []string{
					"first",
					"second",
				},
				"Marked":     true,
				"ModDate":    "2006-09-18T16:27:50-04:00",
				"PDFVersion": 1.7,
				"Producer":   "Gotenberg",
				"Subject":    "Sample",
				"Title":      "Sample",
				"Trapped":    "Unknown",
				// Those are not valid PDF metadata.
				"int":     1,
				"int64":   int64(2),
				"float32": float32(2.2),
				"float64": 3.3,
			},
			expectMetadata: map[string]interface{}{
				"Author":       "Julien Neuhart",
				"Copyright":    "Julien Neuhart",
				"CreationDate": "2006:09:18 16:27:50-04:00",
				"Creator":      "Gotenberg",
				"Keywords": []interface{}{
					"first",
					"second",
				},
				"Marked":     true,
				"ModDate":    "2006:09:18 16:27:50-04:00",
				"PDFVersion": 1.7,
				"Producer":   "Gotenberg",
				"Subject":    "Sample",
				"Title":      "Sample",
				"Trapped":    "Unknown",
			},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(ExifTool)
			err := engine.Provision(nil)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			var destinationPath string
			if tc.createCopy {
				fs := gotenberg.NewFileSystem()
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

			err = engine.WriteMetadata(context.Background(), zap.NewNop(), tc.metadata, destinationPath)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v but got: %v", tc.expectedError, err)
			}

			if tc.expectError {
				return
			}

			metadata, err := engine.ReadMetadata(context.Background(), zap.NewNop(), destinationPath)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectMetadata != nil && err == nil {
				for k, v := range tc.expectMetadata {
					v2, ok := metadata[k]
					if !ok {
						t.Errorf("expected entry %s with value %v to exists, but got none", k, v)
						continue
					}

					switch v2.(type) {
					case []interface{}:
						for i, entry := range v.([]interface{}) {
							if entry != v2.([]interface{})[i] {
								t.Errorf("expected entry %s to contain value %v, but got %v", k, entry, v2.([]interface{})[i])
							}
						}
					default:
						if v != v2 {
							t.Errorf("expected entry %s with value %v to exists, but got %v", k, v, v2)
						}
					}
				}
			}
		})
	}
}
