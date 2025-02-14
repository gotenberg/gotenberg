package api

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestNewContext(t *testing.T) {
	defaultAllowList, err := regexp2.Compile("", 0)
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}
	defaultDenyList, err := regexp2.Compile("", 0)
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}
	defaultDownloadFromCfg := downloadFromConfig{
		allowList: defaultAllowList,
		denyList:  defaultDenyList,
		maxRetry:  1,
		disable:   false,
	}

	for _, tc := range []struct {
		scenario         string
		request          *http.Request
		bodyLimit        int64
		downloadFromCfg  downloadFromConfig
		downloadFromSrv  *echo.Echo
		expectContext    *Context
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
			scenario: "request entity too large: form values",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("key", "value")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			bodyLimit:        1,
			downloadFromCfg:  defaultDownloadFromCfg,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusRequestEntityTooLarge,
		},
		{
			scenario: "request entity too large: downloadFrom",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("downloadFrom", `[{"url":"http://localhost:80/"}]`)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			bodyLimit: 45, // form values = 44 bytes.
			downloadFromSrv: func() *echo.Echo {
				srv := echo.New()
				srv.HideBanner = true
				srv.GET("/", func(c echo.Context) error {
					c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="bar.txt"`)
					c.Response().Header().Set(echo.HeaderContentType, "text/plain")
					return c.String(http.StatusOK, http.StatusText(http.StatusOK))
				})
				return srv
			}(),
			downloadFromCfg:  defaultDownloadFromCfg,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusRequestEntityTooLarge,
		},
		{
			scenario: "request entity too large: form files",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
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
			bodyLimit:        1,
			downloadFromCfg:  defaultDownloadFromCfg,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusRequestEntityTooLarge,
		},
		{
			scenario: "invalid downloadFrom form field: cannot unmarshal",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("downloadFrom", "foo")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			downloadFromCfg:  defaultDownloadFromCfg,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "invalid downloadFrom form field: no URL",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("downloadFrom", `[{}]`)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			downloadFromCfg:  defaultDownloadFromCfg,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "invalid downloadFrom form field: filtered URL",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("downloadFrom", `[{"url":"https://foo.bar"}]`)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			downloadFromCfg: func() downloadFromConfig {
				denyList, err := regexp2.Compile("https://foo.bar", 0)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return downloadFromConfig{allowList: defaultAllowList, denyList: denyList, maxRetry: 1, disable: false}
			}(),
			expectError: true,
		},
		{
			scenario: "invalid downloadFrom form field: unreachable URL",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("downloadFrom", `[{"url":"http://localhost:80/"}]`)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			downloadFromCfg:  defaultDownloadFromCfg,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "invalid downloadFrom form field: invalid status code",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("downloadFrom", `[{"url":"http://localhost:80/"}]`)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			downloadFromSrv: func() *echo.Echo {
				srv := echo.New()
				srv.HideBanner = true
				srv.GET("/", func(c echo.Context) error {
					return c.String(http.StatusNotFound, http.StatusText(http.StatusNotFound))
				})
				return srv
			}(),
			downloadFromCfg:  defaultDownloadFromCfg,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "invalid downloadFrom form field: no 'Content-Disposition' header",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("downloadFrom", `[{"url":"http://localhost:80/"}]`)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			downloadFromSrv: func() *echo.Echo {
				srv := echo.New()
				srv.HideBanner = true
				srv.GET("/", func(c echo.Context) error {
					return c.String(http.StatusOK, http.StatusText(http.StatusOK))
				})
				return srv
			}(),
			downloadFromCfg:  defaultDownloadFromCfg,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "invalid downloadFrom form field: malformed 'Content-Disposition' header",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("downloadFrom", `[{"url":"http://localhost:80/"}]`)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			downloadFromSrv: func() *echo.Echo {
				srv := echo.New()
				srv.HideBanner = true
				srv.GET("/", func(c echo.Context) error {
					c.Response().Header().Set(echo.HeaderContentDisposition, ";;")
					return c.String(http.StatusOK, http.StatusText(http.StatusOK))
				})
				return srv
			}(),
			downloadFromCfg:  defaultDownloadFromCfg,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "invalid downloadFrom form field: no filename parameter in 'Content-Disposition' header",
			request: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				defer func() {
					err := writer.Close()
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
				err := writer.WriteField("downloadFrom", `[{"url":"http://localhost:80/"}]`)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			downloadFromSrv: func() *echo.Echo {
				srv := echo.New()
				srv.HideBanner = true
				srv.GET("/", func(c echo.Context) error {
					c.Response().Header().Set(echo.HeaderContentDisposition, "inline;")
					return c.String(http.StatusOK, http.StatusText(http.StatusOK))
				})
				return srv
			}(),
			downloadFromCfg:  defaultDownloadFromCfg,
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
				err = writer.WriteField("downloadFrom", `[{"url":"http://localhost:80/","extraHttpHeaders":{"X-Foo":"Bar"}}]`)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				return req
			}(),
			downloadFromSrv: func() *echo.Echo {
				srv := echo.New()
				srv.HideBanner = true
				srv.GET("/", func(c echo.Context) error {
					if c.Request().Header.Get("User-Agent") != "Gotenberg" {
						t.Fatalf("expected 'Gotenberg' from header 'User-Agent', but got '%s'", c.Request().Header.Get("User-Agent"))
					}
					if c.Request().Header.Get("X-Foo") != "Bar" {
						t.Fatalf("expected 'Bar' from header 'X-Foo', but got '%s'", c.Request().Header.Get("X-Foo"))
					}
					if c.Request().Header.Get("Gotenberg-Trace") != "123" {
						t.Fatalf("expected '123' from header 'Gotenberg-Trace', but got '%s'", c.Request().Header.Get("Gotenberg-Trace"))
					}
					c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="bar.txt"`)
					c.Response().Header().Set(echo.HeaderContentType, "text/plain")
					return c.String(http.StatusOK, http.StatusText(http.StatusOK))
				})
				return srv
			}(),
			downloadFromCfg: defaultDownloadFromCfg,
			expectContext: &Context{
				values: map[string][]string{
					"foo": {"foo"},
					"downloadFrom": {
						`[{"url":"http://localhost:80/","extraHttpHeaders":{"X-Foo":"Bar"}}]`,
					},
				},
				files: map[string]string{
					"foo.txt": "foo.txt",
					"bar.txt": "bar.txt", // downloadFrom.
				},
			},
			expectError:     false,
			expectHttpError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.downloadFromSrv != nil {
				go func() {
					err := tc.downloadFromSrv.Start(":80")
					if !errors.Is(err, http.ErrServerClosed) {
						t.Error(err)
						return
					}
				}()
				defer func() {
					err := tc.downloadFromSrv.Shutdown(context.TODO())
					if err != nil {
						t.Error(err)
					}
				}()
			}

			handler := func(c echo.Context) error {
				ctx, cancel, err := newContext(c, zap.NewNop(), gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll)), time.Duration(10)*time.Second, tc.bodyLimit, tc.downloadFromCfg, "Gotenberg-Trace", "123")
				defer cancel()
				// Context already cancelled.
				defer cancel()

				if err != nil {
					return err
				}

				if tc.expectContext != nil {
					if !reflect.DeepEqual(tc.expectContext.values, ctx.values) {
						t.Fatalf("expected context.values to be %v but got %v", tc.expectContext.values, ctx.values)
					}
					if len(tc.expectContext.files) != len(ctx.files) {
						t.Fatalf("expected context.files to contain %d items but got %d", len(tc.expectContext.files), len(ctx.files))
					}
					for key, value := range tc.expectContext.files {
						if !strings.HasSuffix(ctx.files[key], value) {
							t.Fatalf("expected context.files to contain '%s' but got '%s'", value, ctx.files[key])
						}
					}
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

func TestContext_CreateSubDirectory(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         *Context
		expectError bool
	}{
		{
			scenario: "failure",
			ctx: &Context{mkdirAll: &gotenberg.MkdirAllMock{MkdirAllMock: func(path string, perm os.FileMode) error {
				return errors.New("cannot rename")
			}}},
			expectError: true,
		},
		{
			scenario: "success",
			ctx: &Context{mkdirAll: &gotenberg.MkdirAllMock{MkdirAllMock: func(path string, perm os.FileMode) error {
				return nil
			}}},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.logger = zap.NewNop()
			_, err := tc.ctx.CreateSubDirectory("foo")

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

func TestContext_GeneratePath(t *testing.T) {
	ctx := &Context{
		dirPath: "/foo",
	}

	path := ctx.GeneratePath(".pdf")
	if !strings.HasPrefix(path, ctx.dirPath) {
		t.Errorf("expected '%s' to start with '%s'", path, ctx.dirPath)
	}
}

func TestContext_Rename(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         *Context
		expectError bool
	}{
		{
			scenario: "failure",
			ctx: &Context{pathRename: &gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
				return errors.New("cannot rename")
			}}},
			expectError: true,
		},
		{
			scenario: "success",
			ctx: &Context{pathRename: &gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
				return nil
			}}},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.logger = zap.NewNop()
			err := tc.ctx.Rename("", "")

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
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
			fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
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
			tc.ctx.Context = context.Background()
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
