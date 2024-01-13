package exiftool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestMetadataValueTypeError_Error(t *testing.T) {
	instance := MetadataValueTypeError{
		Entries: map[string]interface{}{
			"foo": "foo",
		},
	}

	assert.True(t, len(instance.Error()) > 0)
}

func TestMetadataValueTypeError_GetKeys(t *testing.T) {
	for i, tc := range []struct {
		instance MetadataValueTypeError
		expect   []string
	}{
		{
			instance: MetadataValueTypeError{
				Entries: map[string]interface{}{},
			},
			expect: []string{},
		},
		{
			instance: MetadataValueTypeError{
				Entries: map[string]interface{}{
					"foo": "foo",
				},
			},
			expect: []string{"foo"},
		},
		{
			instance: MetadataValueTypeError{
				Entries: map[string]interface{}{
					"foo":  "foo",
					"bar":  float64(123),
					"baz":  4.56,
					"qux":  true,
					"quux": nil,
				},
			},
			expect: []string{"foo", "bar", "baz", "qux", "quux"},
		},
	} {
		actual := tc.instance.GetKeys()

		if !assert.ElementsMatch(t, actual, tc.expect) {
			t.Errorf("test %d: expected %+v but got: %+v", i, tc.expect, actual)
		}
	}
}

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

// FileMetadata is a wrapper to identify which file the metadata came from.
type FileMetadata struct {
	path     string
	metadata map[string]interface{}
}

func TestExiftool_ReadMetadata(t *testing.T) {
	type Subset struct {
		fileMetadata FileMetadata
		expectDiff   bool
	}

	for _, tc := range []struct {
		scenario    string
		ctx         context.Context
		inputPaths  []string
		subsets     []Subset
		expectError bool
	}{
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
			subsets: []Subset{
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/pdfengines/sample1.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/pdfengines",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/pdfengines/sample1.pdf",
						},
					},
					expectDiff: false,
				},
			},
		},
		{
			scenario: "many files success",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
			subsets: []Subset{
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/pdfengines/sample1.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/pdfengines",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/pdfengines/sample1.pdf",
						},
					},
					expectDiff: false,
				},
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/pdfengines/sample2.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample2.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/pdfengines",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/pdfengines/sample2.pdf",
						},
					},
					expectDiff: false,
				},
			},
		},
		{
			scenario: "single file incorrect metadata",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
			},
			subsets: []Subset{
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/pdfengines/sample1.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/pdfengines",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "INVALID",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/pdfengines/sample1.pdf",
						},
					},
					expectDiff: true,
				},
			},
		},
		{
			scenario: "many files incorrect metadata",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
			subsets: []Subset{
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/pdfengines/sample1.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1,
							"PageCount":         float64(30),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/pdfengines",
							"FileType":          "INVALID",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/pdfengines/sample1.pdf",
						},
					},
					expectDiff: true,
				},
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/pdfengines/sample2.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample2.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/pdfengines",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/pdfengines/sample2.pdf",
						},
					},
					expectDiff: false,
				},
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(ExifTool)
			err := engine.Provision(nil)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			actualMetadatas := make([]gotenberg.FileMetadata, len(tc.inputPaths))
			err = engine.ReadMetadata(tc.ctx, zap.NewNop(), tc.inputPaths, actualMetadatas)
			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.subsets != nil && err == nil {
				for _, subset := range tc.subsets {
					for _, actualFileMetadata := range actualMetadatas {
						if subset.fileMetadata.path == actualFileMetadata.Path {
							if !subset.expectDiff && !isMapSubset(actualFileMetadata.Metadata, subset.fileMetadata.metadata) {
								t.Errorf("test %s: expected: %+v to be a subset of: %+v at path: %s",
									tc.scenario, subset.fileMetadata.metadata, actualFileMetadata.Metadata, actualFileMetadata.Path)
							} else if subset.expectDiff && isMapSubset(actualFileMetadata.Metadata, subset.fileMetadata.metadata) {
								t.Errorf("test %s: expected: %+v to be not be a subset of: %+v at path: %s",
									tc.scenario, subset.fileMetadata.metadata, actualFileMetadata.Metadata, actualFileMetadata.Path)
							}
						}
					}
				}
			}
		})
	}
}

func TestExiftool_WriteMetadata(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         context.Context
		inputPaths  []string
		newMetadata map[string]interface{}
		contains    map[string]interface{}
		expectError bool
		expectDiff  bool
	}{
		{
			scenario: "single file success",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
			},
			newMetadata: map[string]interface{}{
				"Producer": "foo",
			},
			contains: map[string]interface{}{
				"Producer": "foo",
			},
			expectError: false,
			expectDiff:  false,
		},
		{
			scenario: "many files success",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
			newMetadata: map[string]interface{}{
				"Producer":   "foo",
				"Keywords":   []string{"foo", "bar"},
				"PDFVersion": 0.0,
				"ModifyDate": "2023:12:06 17:50:06+00:00",
			},
			contains: map[string]interface{}{
				"Producer":   "foo",
				"Keywords":   []string{"foo", "bar"},
				"PDFVersion": 0.0,
				"ModifyDate": "2023:12:06 17:50:06+00:00",
			},
			expectError: false,
			expectDiff:  false,
		},
		{
			scenario: "single file not same metadata",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
			},
			newMetadata: map[string]interface{}{
				"Producer": "foo",
			},
			contains: map[string]interface{}{
				"Producer": "foobar",
			},
			expectError: false,
			expectDiff:  true,
		},
		{
			scenario: "many files not same metadata",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
			newMetadata: map[string]interface{}{
				"Producer":   "foo",
				"Keywords":   []string{"foo", "bar"},
				"PDFVersion": 0.0,
				"ModifyDate": "2023:12:06 17:50:06+00:00",
			},
			contains: map[string]interface{}{
				"Producer":   "foobar",
				"Keywords":   []string{"foo1", "bar1"},
				"PDFVersion": 2.0,
				"ModifyDate": "2022:12:06 17:50:06+00:00",
			},
			expectError: false,
			expectDiff:  true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(ExifTool)
			err := engine.Provision(nil)
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

			var copyPaths []string
			for idx, inputPath := range tc.inputPaths {
				copyPath := fmt.Sprintf("%s/copy_%d.pdf", outputDir, idx)
				// open the source file
				source, err := os.Open(inputPath)
				if err != nil {
					t.Fatalf("error in opening file: %v", err)
				}

				// create the destination file
				destination, err := os.Create(copyPath)
				if err != nil {
					t.Fatalf("error in creating file: %v", err)
				}

				// copy the contents of source to destination file
				_, err = io.Copy(destination, source)
				if err != nil {
					t.Fatalf("error in copying file: %v", err)
				}

				err = source.Close()
				if err != nil {
					t.Fatalf("error in source file close: %v", err)
				}
				err = destination.Close()
				if err != nil {
					t.Fatalf("error in destination file close: %v", err)
				}
				copyPaths = append(copyPaths, copyPath)
			}

			// write metadata to new copy files
			err = engine.WriteMetadata(tc.ctx, zap.NewNop(), copyPaths, tc.newMetadata)
			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if err == nil {
				readMetadatas := make([]gotenberg.FileMetadata, len(copyPaths))
				readErr := engine.ReadMetadata(tc.ctx, zap.NewNop(), copyPaths, readMetadatas)
				if tc.contains != nil && readErr == nil {
					// match metadata
					for _, readMetadata := range readMetadatas {
						if !tc.expectDiff && !isMapSubset(readMetadata.Metadata, tc.contains) {
							t.Errorf("test %s: expected: %+v to be a subset of: %+v at path: %s",
								tc.scenario, tc.contains, readMetadata.Metadata, readMetadata.Path)
						} else if tc.expectDiff && isMapSubset(readMetadata.Metadata, tc.contains) {
							t.Errorf("test %s: expected: %+v to be not be a subset of: %+v at path: %s",
								tc.scenario, tc.contains, readMetadata.Metadata, readMetadata.Path)
						}
					}
				}
			}
		})
	}
}

func isMapSubset(mapSet interface{}, mapSubset interface{}) bool {
	mapSetValue := reflect.ValueOf(mapSet)
	mapSubsetValue := reflect.ValueOf(mapSubset)

	if mapSetValue.Kind() != reflect.Map || mapSubsetValue.Kind() != reflect.Map {
		return false
	}
	if reflect.TypeOf(mapSetValue) != reflect.TypeOf(mapSubsetValue) {
		return false
	}
	if len(mapSubsetValue.MapKeys()) == 0 {
		return true
	}

	iterMapSubset := mapSubsetValue.MapRange()

	for iterMapSubset.Next() {
		k := iterMapSubset.Key()
		v := iterMapSubset.Value()

		v2 := mapSetValue.MapIndex(k)

		if !v2.IsValid() {
			return false
		}
		if isValueKind(v, reflect.Slice) && isValueKind(v2, reflect.Slice) {
			vSlice := convertSlice(v)
			v2Slice := convertSlice(v2)
			if !equal(vSlice, v2Slice) {
				return false
			}
		} else if v.Interface() != v2.Interface() {
			return false
		}
	}

	return true
}

func isValueKind(value reflect.Value, kind reflect.Kind) bool {
	return reflect.TypeOf(value.Interface()).Kind() == kind
}

func convertSlice(value reflect.Value) []interface{} {
	slice := make([]interface{}, reflect.ValueOf(value.Interface()).Len())
	for i := range slice {
		slice = append(slice, reflect.ValueOf(value.Interface()).Index(i).Interface())
	}
	return slice
}

// equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func equal(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
