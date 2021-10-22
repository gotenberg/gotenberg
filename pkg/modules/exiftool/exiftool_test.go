package exiftool

import (
	"context"
	"fmt"
	exiftoolLib "github.com/barasher/go-exiftool"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"os"
	"reflect"
	"testing"
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

func TestExiftool_Descriptor(t *testing.T) {
	descriptor := Exiftool{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Exiftool))

	if actual != expect {
		t.Errorf("expected '%'s' but got '%s'", expect, actual)
	}
}

func TestExiftool_Provision(t *testing.T) {
	for i, tc := range []struct {
		setup     func()
		mod       *Exiftool
		ctx       *gotenberg.Context
		expectErr bool
	}{
		{
			mod: new(Exiftool),
			ctx: gotenberg.NewContext(gotenberg.ParsedFlags{}, nil),
		},
		{
			setup: func() {
				binPath := os.Getenv("EXIF_BIN_PATH")

				err := os.Unsetenv("EXIF_BIN_PATH")
				if err != nil {
					return
				}

				t.Cleanup(func() {
					err := os.Setenv("EXIF_BIN_PATH", binPath)
					if err != nil {
						return
					}
				})
			},
			mod:       new(Exiftool),
			ctx:       gotenberg.NewContext(gotenberg.ParsedFlags{}, nil),
			expectErr: true,
		},
		{
			setup: func() {
				binPath := os.Getenv("EXIF_BIN_PATH")

				err := os.Setenv("EXIF_BIN_PATH", "/foo")
				if err != nil {
					return
				}

				t.Cleanup(func() {
					err := os.Setenv("EXIF_BIN_PATH", binPath)
					if err != nil {
						return
					}
				})
			},
			mod:       new(Exiftool),
			ctx:       gotenberg.NewContext(gotenberg.ParsedFlags{}, nil),
			expectErr: true,
		},
	} {
		if tc.setup != nil {
			tc.setup()
		}

		err := tc.mod.Provision(tc.ctx)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestExiftool_Validate(t *testing.T) {
	for i, tc := range []struct {
		binPath   string
		exiftool  *exiftoolLib.Exiftool
		expectErr bool
	}{
		{
			expectErr: true,
		},
		{
			binPath:   "/foo",
			expectErr: true,
		},
		{
			binPath:   os.Getenv("EXIF_BIN_PATH"),
			expectErr: true,
		},
		{
			binPath: os.Getenv("EXIF_BIN_PATH"),
			exiftool: func() *exiftoolLib.Exiftool {
				exif, err := exiftoolLib.NewExiftool(exiftoolLib.SetExiftoolBinaryPath(os.Getenv("EXIF_BIN_PATH")))
				if err != nil {
					print(fmt.Errorf("unable to create NewExiftool: %w", err))
				}
				return exif
			}(),
		},
		{
			exiftool: func() *exiftoolLib.Exiftool {
				exif, err := exiftoolLib.NewExiftool(exiftoolLib.SetExiftoolBinaryPath(os.Getenv("EXIF_BIN_PATH")))
				if err != nil {
					print(fmt.Errorf("unable to create NewExiftool: %w", err))
				}
				return exif
			}(),
			expectErr: true,
		},
	} {
		mod := new(Exiftool)
		mod.binPath = tc.binPath
		mod.exiftool = tc.exiftool
		err := mod.Validate()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestExiftool_Exiftool(t *testing.T) {
	mod := new(Exiftool)

	_, err := mod.Exiftool()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestExiftool_ReadMetadata(t *testing.T) {

	type Subset struct {
		fileMetadata FileMetadata
		expectDiff   bool
	}

	for i, tc := range []struct {
		ctx        context.Context
		inputPaths []string
		subsets    []Subset
		expectErr  bool
	}{
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{},
			subsets: []Subset{
				{
					fileMetadata: FileMetadata{},
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{"/foo"},
			expectErr:  true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{"/tests/test/testdata/exiftool/sample1.pdf"},
			subsets: []Subset{
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/exiftool/sample1.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "LibreOffice",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample1.pdf",
						},
					},
					expectDiff: true,
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{"/foo"},
			expectErr:  true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{"/tests/test/testdata/exiftool/sample1.pdf"},
			subsets: []Subset{
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/exiftool/sample1.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample1.pdf",
						},
					},
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{
				"/tests/test/testdata/exiftool/sample1.pdf",
				"/tests/test/testdata/exiftool/sample2.pdf",
			},
			subsets: []Subset{
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/exiftool/sample1.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample1.pdf",
						},
					},
				},
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/exiftool/sample2.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample2.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "LibreOffice",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample2.pdf",
						},
					},
					expectDiff: true,
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{
				"/tests/test/testdata/exiftool/sample1.pdf",
				"/tests/test/testdata/exiftool/sample2.pdf",
			},
			subsets: []Subset{
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/exiftool/sample1.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample1.pdf",
						},
					},
				},
				{
					fileMetadata: FileMetadata{
						path: "/tests/test/testdata/exiftool/sample2.pdf",
						metadata: map[string]interface{}{
							"FileName":          "sample2.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample2.pdf",
						},
					},
				},
			},
		},
	} {
		mod := new(Exiftool)
		err := mod.Provision(nil)

		if err != nil {
			t.Fatalf("test %d: unable to provision module exif: %v", i, err)
		}

		actual, err := mod.ReadMetadata(tc.ctx, zap.NewNop(), tc.inputPaths)

		if tc.subsets != nil && err == nil {
			for _, subset := range tc.subsets {
				for _, actualFileMetadata := range *actual {
					if subset.fileMetadata.path == actualFileMetadata.path {
						if !subset.expectDiff && !IsMapSubset(actualFileMetadata.metadata, subset.fileMetadata.metadata) {
							t.Errorf("test %d: expected: %+v to be a subset of: %+v at path: %s",
								i, subset.fileMetadata.metadata, actualFileMetadata.metadata, actualFileMetadata.path)
						} else if subset.expectDiff && IsMapSubset(actualFileMetadata.metadata, subset.fileMetadata.metadata) {
							t.Errorf("test %d: expected: %+v to be not be a subset of: %+v at path: %s",
								i, subset.fileMetadata.metadata, actualFileMetadata.metadata, actualFileMetadata.path)
						}
					}
				}
			}
		}

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestExtractMetadata(t *testing.T) {

	type Subset struct {
		fileMetadata exiftoolLib.FileMetadata
		expectDiff   bool
	}

	for i, tc := range []struct {
		ctx        context.Context
		inputPaths []string
		subsets    []Subset
		expectErr  bool
	}{
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{},
			subsets: []Subset{
				{
					fileMetadata: exiftoolLib.FileMetadata{},
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{"/foo"},
			expectErr:  true,
		},

		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{"/foo", "/bar"},
			expectErr:  true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{"/tests/test/testdata/exiftool/sample1.pdf"},
			subsets: []Subset{
				{
					fileMetadata: exiftoolLib.FileMetadata{
						File: "/tests/test/testdata/exiftool/sample1.pdf",
						Fields: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "LibreOffice",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample1.pdf",
						},
					},
					expectDiff: true,
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{"/tests/test/testdata/exiftool/sample1.pdf"},
			subsets: []Subset{
				{
					fileMetadata: exiftoolLib.FileMetadata{
						File: "/tests/test/testdata/exiftool/sample1.pdf",
						Fields: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample1.pdf",
						},
					},
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{
				"/tests/test/testdata/exiftool/sample1.pdf",
				"/tests/test/testdata/exiftool/sample2.pdf",
			},
			subsets: []Subset{
				{
					fileMetadata: exiftoolLib.FileMetadata{
						File: "/tests/test/testdata/exiftool/sample1.pdf",
						Fields: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample1.pdf",
						},
					},
				},
				{
					fileMetadata: exiftoolLib.FileMetadata{
						File: "/tests/test/testdata/exiftool/sample2.pdf",
						Fields: map[string]interface{}{
							"FileName":          "sample2.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "LibreOffice",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample2.pdf",
						},
					},
					expectDiff: true,
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{
				"/tests/test/testdata/exiftool/sample1.pdf",
				"/tests/test/testdata/exiftool/sample2.pdf",
			},
			subsets: []Subset{
				{
					fileMetadata: exiftoolLib.FileMetadata{
						File: "/tests/test/testdata/exiftool/sample1.pdf",
						Fields: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "LibreOffice",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample1.pdf",
						},
					},
					expectDiff: true,
				},
				{
					fileMetadata: exiftoolLib.FileMetadata{
						File: "/tests/test/testdata/exiftool/sample2.pdf",
						Fields: map[string]interface{}{
							"FileName":          "sample2.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "LibreOffice",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample2.pdf",
						},
					},
					expectDiff: true,
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			inputPaths: []string{
				"/tests/test/testdata/exiftool/sample1.pdf",
				"/tests/test/testdata/exiftool/sample2.pdf",
			},
			subsets: []Subset{
				{
					fileMetadata: exiftoolLib.FileMetadata{
						File: "/tests/test/testdata/exiftool/sample1.pdf",
						Fields: map[string]interface{}{
							"FileName":          "sample1.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample1.pdf",
						},
					},
				},
				{
					fileMetadata: exiftoolLib.FileMetadata{
						File: "/tests/test/testdata/exiftool/sample2.pdf",
						Fields: map[string]interface{}{
							"FileName":          "sample2.pdf",
							"FileTypeExtension": "pdf",
							"MIMEType":          "application/pdf",
							"PDFVersion":        1.4,
							"PageCount":         float64(3),
							"CreateDate":        "2018:12:06 17:50:06+00:00",
							"ModifyDate":        "2018:12:06 17:50:06+00:00",
							"Directory":         "/tests/test/testdata/exiftool",
							"FileType":          "PDF",
							"Linearized":        "No",
							"Creator":           "Chromium",
							"Producer":          "Skia/PDF m70",
							"SourceFile":        "/tests/test/testdata/exiftool/sample2.pdf",
						},
					},
				},
			},
		},
	} {
		mod := new(Exiftool)
		err := mod.Provision(nil)

		if err != nil {
			t.Fatalf("test %d: unable to provision module exif: %v", i, err)
		}

		actual, err := extractMetadata(tc.inputPaths, mod.exiftool, zap.NewNop())

		if tc.subsets != nil && err == nil {
			for _, subset := range tc.subsets {
				for _, actualFileMetadata := range *actual {
					if subset.fileMetadata.File == actualFileMetadata.File {
						if !subset.expectDiff && !IsMapSubset(actualFileMetadata.Fields, subset.fileMetadata.Fields) {
							t.Errorf("test %d: expected: %+v to be a subset of: %+v at path: %s",
								i, subset.fileMetadata.Fields, actualFileMetadata.Fields, actualFileMetadata.File)
						} else if subset.expectDiff && IsMapSubset(actualFileMetadata.Fields, subset.fileMetadata.Fields) {
							t.Errorf("test %d: expected: %+v to be not be a subset of: %+v at path: %s",
								i, subset.fileMetadata.Fields, actualFileMetadata.Fields, actualFileMetadata.File)
						}
					}
				}
			}
		}

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestExiftool_WriteMetadata(t *testing.T) {
	for i, tc := range []struct {
		ctx           context.Context
		before        func() error
		after         func() error
		paths         []string
		inputMetadata *map[string]interface{}
		contains      map[string]interface{}
		expectDiff    bool
		expectErr     bool
		expectedErr   error
	}{
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			paths:         []string{},
			inputMetadata: &map[string]interface{}{},
			contains:      map[string]interface{}{},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			paths:         []string{"/foo"},
			inputMetadata: &map[string]interface{}{},
			contains:      map[string]interface{}{},
			expectErr:     true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			paths:         []string{"/foo", "/bar", "/baz"},
			inputMetadata: &map[string]interface{}{},
			contains:      map[string]interface{}{},
			expectErr:     true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			paths:         []string{"/tests/test/testdata/exiftool/sample3.pdf"},
			inputMetadata: &map[string]interface{}{},
			contains:      map[string]interface{}{},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			paths: []string{
				"/tests/test/testdata/exiftool/sample3.pdf",
				"/tests/test/testdata/exiftool/sample4.pdf",
			},
			inputMetadata: &map[string]interface{}{},
			contains:      map[string]interface{}{},
			expectErr:     true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest := "/tests/test/testdata/exiftool/tmp-sample4.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}
				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/sample3.pdf",
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
			},
			inputMetadata: &map[string]interface{}{},
			contains:      map[string]interface{}{},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest := "/tests/test/testdata/exiftool/tmp-sample4.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}
				return nil
			},
			paths: []string{"/tests/test/testdata/exiftool/tmp-sample4.pdf"},
			inputMetadata: &map[string]interface{}{
				"Producer": "foo",
			},
			contains: map[string]interface{}{
				"Producer": "foo",
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"Producer": "bar",
			},
			contains: map[string]interface{}{
				"Producer": "bar",
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"Producer": "baz",
				"Creator":  "foo",
			},
			contains: map[string]interface{}{
				"Producer": "baz",
				"Creator":  "foo",
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"Producer": "qux",
				"Creator":  "bar",
			},
			contains: map[string]interface{}{
				"Producer": "qux",
				"Creator":  "bar",
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"PDFVersion": float32(0.0),
			},
			contains: map[string]interface{}{
				"PDFVersion": 0.0,
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"PDFVersion": 0.0,
			},
			contains: map[string]interface{}{
				"PDFVersion": 0.0,
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"Version": 0,
			},
			contains: map[string]interface{}{
				"Version": float64(0),
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"Version": int64(0),
			},
			contains: map[string]interface{}{
				"Version": float64(0),
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"PageCount": true,
			},
			expectErr: true,
			expectedErr: &MetadataValueTypeError{
				Entries: map[string]interface{}{
					"PageCount": true,
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"ModifyDate": "2021:08:24 11:40:59+00:00",
			},
			contains: map[string]interface{}{
				"ModifyDate": "2021:08:24 11:40:59+00:00",
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"Trapped": true,
			},
			expectErr: true,
			expectedErr: &MetadataValueTypeError{
				Entries: map[string]interface{}{
					"Trapped": true,
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"Trapped": nil,
			},
			expectErr: true,
			expectedErr: &MetadataValueTypeError{
				Entries: map[string]interface{}{
					"Trapped": nil,
				},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"
				dest2 := "/tests/test/testdata/exiftool/tmp-sample5.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest2, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				err = os.Remove("/tests/test/testdata/exiftool/tmp-sample5.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
				"/tests/test/testdata/exiftool/tmp-sample5.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"Keywords": []string{"foo", "bar", "baz"},
			},
			contains: map[string]interface{}{
				"Keywords": []string{"foo", "bar", "baz"},
			},
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			before: func() error {
				src := "/tests/test/testdata/exiftool/sample3.pdf"
				dest1 := "/tests/test/testdata/exiftool/tmp-sample4.pdf"

				bytesRead, err := os.ReadFile(src)
				if err != nil {
					return err
				}

				err = os.WriteFile(dest1, bytesRead, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			after: func() error {
				err := os.Remove("/tests/test/testdata/exiftool/tmp-sample4.pdf")
				if err != nil {
					return err
				}

				return nil
			},
			paths: []string{
				"/tests/test/testdata/exiftool/tmp-sample4.pdf",
			},
			inputMetadata: &map[string]interface{}{
				"Producer":   "quux",
				"Creator":    "baz",
				"ModifyDate": "2021:08:24 11:40:59+00:00",
				"Keywords":   []string{"foo", "bar"},
				"Version":    0,
				"PDFVersion": 0.0,
			},
			contains: map[string]interface{}{
				"Producer":   "quux",
				"Creator":    "baz",
				"Keywords":   []string{"foo", "bar"},
				"ModifyDate": "2021:08:24 11:40:59+00:00",
				"Version":    float64(0),
				"PDFVersion": 0.0,
			},
		},
	} {
		mod := new(Exiftool)
		err := mod.Provision(nil)
		if err != nil {
			t.Fatalf("test %d: unable to provision module exif: %v", i, err)
		}

		if tc.before != nil {
			err = tc.before()
			if err != nil {
				t.Errorf("test %d: unable to run function before test: %v", i, err)
				t.FailNow()
			}
		}

		err = mod.WriteMetadata(tc.ctx, zap.NewNop(), tc.paths, tc.inputMetadata)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		if err == nil {
			actuals, readErr := mod.ReadMetadata(tc.ctx, zap.NewNop(), tc.paths)

			if tc.contains != nil && readErr == nil {
				for _, actualFileMetadata := range *actuals {
					if !tc.expectDiff && !IsMapSubset(actualFileMetadata.metadata, tc.contains) {
						t.Errorf("test %d: expected: %+v to be a subset of: %+v at path: %s",
							i, tc.contains, actualFileMetadata.metadata, actualFileMetadata.path)
					} else if tc.expectDiff && IsMapSubset(actualFileMetadata.metadata, tc.contains) {
						t.Errorf("test %d: expected: %+v to be not be a subset of: %+v at path: %s",
							i, tc.contains, actualFileMetadata.metadata, actualFileMetadata.path)
					}

				}
			}

		}

		if tc.expectedErr != nil {
			valid := assert.IsType(t, err, tc.expectedErr)
			if !valid {
				t.Errorf("test %d: expected type %T but got: %T", i, tc.expectedErr, err)
			}
		}

		if tc.after != nil {
			err = tc.after()
			if err != nil {
				t.Logf("test %d: unable to run function after test: %v", i, err)
			}
		}

	}
}

func IsMapSubset(mapSet interface{}, mapSubset interface{}) bool {

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
			if !Equal(vSlice, v2Slice) {
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

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func Equal(a, b []interface{}) bool {
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
