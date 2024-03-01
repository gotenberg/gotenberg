package api

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestNewContext(t *testing.T) {
	for _, tc := range []struct {
		scenario         string
		request          *http.Request
		expectError      bool
		expectHttpError  bool
		expectHttpStatus int
	}{
		{
			scenario:         "http.ErrNotMultipart",
			request:          httptest.NewRequest(http.MethodPost, "/", nil),
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusUnsupportedMediaType,
		},
		{
			scenario: "http.ErrMissingBoundary",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", nil)
				req.Header.Set(echo.HeaderContentType, echo.MIMEMultipartForm)
				return req
			}(),
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusUnsupportedMediaType,
		},
		{
			scenario: "malformed body",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("foo", "foo")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", nil)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "success",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("foo", "foo")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				part, err := writer.CreateFormFile("foo.txt", "foo.txt")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				_, err = part.Write([]byte("foo"))
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			expectError:     false,
			expectHttpError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			handler := func(c echo.Context) error {
				_, cancel, err := newContext(c, zap.NewNop(), gotenberg.NewFileSystem(), time.Duration(10)*time.Second)
				defer cancel()
				// Context already cancelled.
				defer cancel()

				if err != nil {
					return err
				}

				return nil
			}

			recorder := httptest.NewRecorder()

			srv := echo.New()
			srv.HideBanner = true
			srv.HidePort = true

			c := srv.NewContext(tc.request, recorder)
			err := handler(c)

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			var httpErr HttpError
			isHttpError := errors.As(err, &httpErr)

			if tc.expectHttpError && !isHttpError {
				t.Errorf("expected an HTTP error but got: %v", err)
			}

			if !tc.expectHttpError && isHttpError {
				t.Errorf("expected no HTTP error but got one: %v", httpErr)
			}

			if err != nil && tc.expectHttpError && isHttpError {
				status, _ := httpErr.HttpError()
				if status != tc.expectHttpStatus {
					t.Errorf("expected %d as HTTP status code but got %d", tc.expectHttpStatus, status)
				}
			}
		})
	}
}

func TestContext_Request(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	recorder := httptest.NewRecorder()
	c := echo.New().NewContext(request, recorder)

	ctx := &Context{
		echoCtx: c,
	}

	if !reflect.DeepEqual(ctx.Request(), c.Request()) {
		t.Errorf("expected %v but got %v", ctx.Request(), c.Request())
	}
}

func TestContext_FormData(t *testing.T) {
	ctx := &Context{
		values: map[string][]string{
			"foo": {"foo"},
		},
		files: map[string]string{
			"foo.txt": "/foo.txt",
		},
	}

	actual := ctx.FormData()
	expect := &FormData{
		values: ctx.values,
		files:  ctx.files,
	}

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %+v but got %+v", expect, actual)
	}
}

func TestContext_GeneratePath(t *testing.T) {
	ctx := &Context{
		dirPath: "/foo",
	}

	path := ctx.GeneratePath("", ".pdf")
	if !strings.HasPrefix(path, ctx.dirPath) {
		t.Errorf("expected '%s' to start with '%s'", path, ctx.dirPath)
	}

	path = ctx.GeneratePath("foo.txt", ".pdf")
	if !strings.Contains(path, "foo.txt.pdf") {
		t.Errorf("expected '%s' to start with '%s'", path, ctx.dirPath)
	}
}

func TestContext_AddOutputPaths(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         *Context
		path        string
		expectCount int
		expectError bool
	}{
		{
			scenario:    "ErrContextAlreadyClosed",
			ctx:         &Context{cancelled: true},
			expectCount: 0,
			expectError: true,
		},
		{
			scenario:    "ErrOutOfBoundsOutputPath",
			ctx:         &Context{dirPath: "/foo"},
			path:        "/bar/foo.txt",
			expectCount: 0,
			expectError: true,
		},
		{
			scenario:    "success",
			ctx:         &Context{dirPath: "/foo"},
			path:        "/foo/foo.txt",
			expectCount: 1,
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			err := tc.ctx.AddOutputPaths(tc.path)

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if len(tc.ctx.outputPaths) != tc.expectCount {
				t.Errorf("expected %d output paths but got %d", tc.expectCount, len(tc.ctx.outputPaths))
			}
		})
	}
}

func TestContext_Log(t *testing.T) {
	expect := zap.NewNop()
	ctx := Context{logger: expect}
	actual := ctx.Log()

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %v but got %v", expect, actual)
	}
}

func TestContext_BuildOutputFile(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         *Context
		expectError bool
	}{
		{
			scenario:    "ErrContextAlreadyClosed",
			ctx:         &Context{cancelled: true},
			expectError: true,
		},
		{
			scenario:    "no output path",
			ctx:         &Context{},
			expectError: true,
		},
		{
			scenario:    "success: one output path",
			ctx:         &Context{outputPaths: []string{"foo.txt"}},
			expectError: false,
		},
		{
			scenario:    "cannot archive: invalid output paths",
			ctx:         &Context{outputPaths: []string{"foo.txt", "foo.pdf"}},
			expectError: true,
		},
		{
			scenario: "success: many output paths",
			ctx: &Context{
				outputPaths: []string{
					"/tests/test/testdata/api/sample1.txt",
					"/tests/test/testdata/api/sample1.txt",
				},
			},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := gotenberg.NewFileSystem()
			dirPath, err := fs.MkdirAll()
			if err != nil {
				t.Fatalf("expected no erro but got: %v", err)
			}

			defer func() {
				err := os.RemoveAll(fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up but got: %v", err)
				}
			}()

			tc.ctx.dirPath = dirPath
			tc.ctx.logger = zap.NewNop()

			_, err = tc.ctx.BuildOutputFile()

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

func TestContext_OutputFilename(t *testing.T) {
	for _, tc := range []struct {
		scenario             string
		ctx                  *Context
		outputPath           string
		expectOutputFilename string
	}{
		{
			scenario: "with Gotenberg-Output-Filename header",
			ctx: func() *Context {
				c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/foo", nil), nil)
				c.Request().Header.Set("Gotenberg-Output-Filename", "foo")
				return &Context{echoCtx: c}
			}(),
			outputPath:           "/foo/bar.txt",
			expectOutputFilename: "foo.txt",
		},
		{
			scenario: "without custom filename",
			ctx: func() *Context {
				c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/foo", nil), nil)
				return &Context{echoCtx: c}
			}(),
			outputPath:           "/foo/foo.txt",
			expectOutputFilename: "foo.txt",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			actual := tc.ctx.OutputFilename(tc.outputPath)

			if actual != tc.expectOutputFilename {
				t.Errorf("expected '%s' but got '%s'", tc.expectOutputFilename, actual)
			}
		})
	}
}
