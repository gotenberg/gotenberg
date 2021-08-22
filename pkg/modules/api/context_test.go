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

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func TestNewContext(t *testing.T) {
	for i, tc := range []struct {
		request          *http.Request
		expectErr        bool
		expectHTTPErr    bool
		expectHTTPStatus int
	}{
		{
			request:          httptest.NewRequest(http.MethodPost, "/", nil),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusUnsupportedMediaType,
		},
		{
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", nil)
				req.Header.Set(echo.HeaderContentType, echo.MIMEMultipartForm)

				return req
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusUnsupportedMediaType,
		},
		{
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
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
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
		},
	} {
		handler := func(c echo.Context) error {
			_, cancel, err := newContext(c, zap.NewNop(), time.Duration(10)*time.Second)
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

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		var httpErr HTTPError
		isHTTPErr := errors.As(err, &httpErr)

		if tc.expectHTTPErr && !isHTTPErr {
			t.Errorf("test %d: expected HTTP error but got: %v", i, err)
		}

		if !tc.expectHTTPErr && isHTTPErr {
			t.Errorf("test %d: expected no HTTP error but got one: %v", i, httpErr)
		}

		if err != nil && tc.expectHTTPErr && isHTTPErr {
			status, _ := httpErr.HTTPError()
			if status != tc.expectHTTPStatus {
				t.Errorf("test %d: expected %d HTTP status code but got %d", i, tc.expectHTTPStatus, status)
			}
		}
	}
}

func TestContext_Request(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	recorder := httptest.NewRecorder()
	c := echo.New().NewContext(request, recorder)

	ctx := Context{
		echoCtx: c,
	}

	if !reflect.DeepEqual(ctx.Request(), c.Request()) {
		t.Errorf("expected %v but got %v", ctx.Request(), c.Request())
	}
}

func TestContext_FormData(t *testing.T) {
	ctx := Context{
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
	ctx := Context{
		dirPath: "/foo",
	}

	path := ctx.GeneratePath(".pdf")

	if !strings.HasPrefix(path, ctx.dirPath) {
		t.Errorf("expected '%s' to start with '%s'", path, ctx.dirPath)
	}
}

func TestContext_AddOutputPaths(t *testing.T) {
	for i, tc := range []struct {
		ctx         *Context
		path        string
		expectCount int
		expectErr   bool
	}{
		{
			ctx:       &Context{cancelled: true},
			expectErr: true,
		},
		{
			ctx:       &Context{dirPath: "/foo"},
			path:      "/bar/foo.txt",
			expectErr: true,
		},
		{
			ctx:         &Context{dirPath: "/foo"},
			path:        "/foo/foo.txt",
			expectCount: 1,
		},
	} {
		err := tc.ctx.AddOutputPaths(tc.path)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		if len(tc.ctx.outputPaths) != tc.expectCount {
			t.Errorf("test %d: expected %d output paths but got %d", i, tc.expectCount, len(tc.ctx.outputPaths))
		}
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

func TestContext_buildOutputFile(t *testing.T) {
	for i, tc := range []struct {
		ctx       *Context
		expectErr bool
	}{
		{
			ctx:       &Context{cancelled: true},
			expectErr: true,
		},
		{
			ctx:       &Context{},
			expectErr: true,
		},
		{
			ctx: &Context{outputPaths: []string{"foo.txt"}},
		},
		{
			ctx:       &Context{outputPaths: []string{"foo.txt", "foo.pdf"}},
			expectErr: true,
		},
		{
			ctx: &Context{
				outputPaths: []string{
					"/tests/test/testdata/api/sample1.txt",
					"/tests/test/testdata/api/sample1.txt",
				},
			},
		},
	} {
		dirPath, err := gotenberg.MkdirAll()
		if err != nil {
			t.Fatalf("%d: expected no erro but got: %v", i, err)
		}

		tc.ctx.dirPath = dirPath
		tc.ctx.logger = zap.NewNop()

		_, err = tc.ctx.buildOutputFile()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		err = os.RemoveAll(dirPath)
		if err != nil {
			t.Fatalf("%d: expected no erro but got: %v", i, err)
		}
	}
}

func TestMockContext_SetDirPath(t *testing.T) {
	mock := &MockContext{&Context{}}
	mock.SetDirPath("/foo")

	actual := mock.dirPath
	expect := "/foo"

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestMockContext_SetValues(t *testing.T) {
	mock := &MockContext{&Context{}}
	mock.SetValues(map[string][]string{
		"foo": {"foo"},
	})

	actual := mock.values
	expect := map[string][]string{
		"foo": {"foo"},
	}

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %+v but got: %+v", expect, actual)
	}
}

func TestMockContext_SetFiles(t *testing.T) {
	mock := &MockContext{&Context{}}
	mock.SetFiles(map[string]string{
		"foo": "/foo",
	})

	actual := mock.files
	expect := map[string]string{
		"foo": "/foo",
	}

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %+v but got: %+v", expect, actual)
	}
}

func TestMockContext_SetCancelled(t *testing.T) {
	mock := &MockContext{&Context{}}
	mock.SetCancelled(true)

	actual := mock.cancelled

	if !actual {
		t.Errorf("expected %t but got %t", true, actual)
	}
}

func TestMockContext_OutputPaths(t *testing.T) {
	mock := MockContext{
		&Context{
			outputPaths: []string{"/foo"},
		},
	}

	actual := mock.OutputPaths()
	expect := []string{"/foo"}

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %+v but got: %+v", expect, actual)
	}
}
