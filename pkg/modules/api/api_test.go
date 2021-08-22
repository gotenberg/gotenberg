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
	"testing"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ProtoModule struct {
	descriptor func() gotenberg.ModuleDescriptor
}

func (mod ProtoModule) Descriptor() gotenberg.ModuleDescriptor {
	return mod.descriptor()
}

type ProtoValidator struct {
	ProtoModule
	validate func() error
}

func (mod ProtoValidator) Validate() error {
	return mod.validate()
}

type ProtoMultipartFormDataRouter struct {
	ProtoValidator
	routes func() ([]MultipartFormDataRoute, error)
}

func (mod ProtoMultipartFormDataRouter) Routes() ([]MultipartFormDataRoute, error) {
	return mod.routes()
}

type ProtoMiddlewareProvider struct {
	ProtoValidator
	middlewares func() ([]Middleware, error)
}

func (mod ProtoMiddlewareProvider) Middlewares() ([]Middleware, error) {
	return mod.middlewares()
}

type ProtoHealthChecker struct {
	ProtoValidator
	checks func() ([]health.CheckerOption, error)
}

func (mod ProtoHealthChecker) Checks() ([]health.CheckerOption, error) {
	return mod.checks()
}

type ProtoLoggerProvider struct {
	ProtoModule
	logger func(mod gotenberg.Module) (*zap.Logger, error)
}

func (factory ProtoLoggerProvider) Logger(mod gotenberg.Module) (*zap.Logger, error) {
	return factory.logger(mod)
}

func TestAPI_Descriptor(t *testing.T) {
	descriptor := API{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(API))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestAPI_Provision(t *testing.T) {
	for i, tc := range []struct {
		ctx               *gotenberg.Context
		setEnv            func(i int)
		expectPort        int
		expectMiddlewares []Middleware
		expectErr         bool
	}{
		{
			ctx: func() *gotenberg.Context {
				fs := new(API).Descriptor().FlagSet
				err := fs.Parse([]string{"--api-port-from-env=FOO"})

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					nil,
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				fs := new(API).Descriptor().FlagSet
				err := fs.Parse([]string{"--api-port-from-env=PORT"})

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					nil,
				)
			}(),
			setEnv: func(i int) {
				err := os.Setenv("PORT", "")
				if err != nil {
					t.Fatalf("test %d: expected no error but got: %v", i, err)
				}
			},
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				fs := new(API).Descriptor().FlagSet
				err := fs.Parse([]string{"--api-port-from-env=PORT"})

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					nil,
				)
			}(),
			setEnv: func(i int) {
				err := os.Setenv("PORT", "foo")
				if err != nil {
					t.Fatalf("test %d: expected no error but got: %v", i, err)
				}
			},
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				fs := new(API).Descriptor().FlagSet
				err := fs.Parse([]string{"--api-port-from-env=PORT"})

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					nil,
				)
			}(),
			setEnv: func(i int) {
				err := os.Setenv("PORT", "1337")
				if err != nil {
					t.Fatalf("test %d: expected no error but got: %v", i, err)
				}
			},
			expectPort: 1337,
			expectErr:  true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoMultipartFormDataRouter }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error {
					return errors.New("foo")
				}
				mod.routes = func() ([]MultipartFormDataRoute, error) {
					return nil, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(API).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoMiddlewareProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error {
					return errors.New("foo")
				}
				mod.middlewares = func() ([]Middleware, error) {
					return nil, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(API).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoMiddlewareProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error {
					return nil
				}
				mod.middlewares = func() ([]Middleware, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(API).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoMultipartFormDataRouter }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error {
					return nil
				}
				mod.routes = func() ([]MultipartFormDataRoute, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(API).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoHealthChecker }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error {
					return errors.New("foo")
				}
				mod.checks = func() ([]health.CheckerOption, error) {
					return nil, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(API).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoHealthChecker }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error {
					return nil
				}
				mod.checks = func() ([]health.CheckerOption, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(API).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(API).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoLoggerProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.logger = func(_ gotenberg.Module) (*zap.Logger, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(API).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod1 := struct{ ProtoMultipartFormDataRouter }{}
				mod1.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod1 }}
				}
				mod1.validate = func() error {
					return nil
				}
				mod1.routes = func() ([]MultipartFormDataRoute, error) {
					return []MultipartFormDataRoute{{}}, nil
				}

				mod2 := struct{ ProtoMiddlewareProvider }{}
				mod2.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod2 }}
				}
				mod2.validate = func() error {
					return nil
				}
				mod2.middlewares = func() ([]Middleware, error) {
					return []Middleware{
						{
							Priority: VeryLowPriority,
						},
						{
							Priority: LowPriority,
						},
						{
							Priority: MediumPriority,
						},
						{
							Priority: HighPriority,
						},
						{
							Priority: VeryHighPriority,
						},
					}, nil
				}

				mod3 := struct{ ProtoHealthChecker }{}
				mod3.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "baz", New: func() gotenberg.Module { return mod3 }}
				}
				mod3.validate = func() error {
					return nil
				}
				mod3.checks = func() ([]health.CheckerOption, error) {
					return []health.CheckerOption{health.WithDisabledAutostart()}, nil
				}

				mod4 := struct{ ProtoLoggerProvider }{}
				mod4.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "qux", New: func() gotenberg.Module { return mod4 }}
				}
				mod4.logger = func(_ gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(API).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod1.Descriptor(),
						mod2.Descriptor(),
						mod3.Descriptor(),
						mod4.Descriptor(),
					},
				)
			}(),
			expectMiddlewares: []Middleware{
				{
					Priority: VeryHighPriority,
				},
				{
					Priority: HighPriority,
				},
				{
					Priority: MediumPriority,
				},
				{
					Priority: LowPriority,
				},
				{
					Priority: VeryLowPriority,
				},
			},
		},
	} {
		if tc.setEnv != nil {
			tc.setEnv(i)
		}

		mod := new(API)
		err := mod.Provision(tc.ctx)

		if tc.expectPort != 0 && mod.port != tc.expectPort {
			t.Errorf("expected port %d but got %d", tc.expectPort, mod.port)
		}

		if !reflect.DeepEqual(mod.externalMiddlewares, tc.expectMiddlewares) {
			t.Errorf("expected %+v, but got: %+v", tc.expectMiddlewares, mod.externalMiddlewares)
		}

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestAPI_Validate(t *testing.T) {
	for i, tc := range []struct {
		port        int
		rootPath    string
		traceHeader string
		routes      []MultipartFormDataRoute
		middlewares []Middleware
		expectErr   bool
	}{
		{
			port:      0,
			expectErr: true,
		},
		{
			port:      65536,
			rootPath:  "foo",
			expectErr: true,
		},
		{
			port:        10,
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []MultipartFormDataRoute{
				{
					Path: "",
				},
			},
			expectErr: true,
		},
		{
			port:        10,
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []MultipartFormDataRoute{
				{
					Path: "foo",
				},
			},
			expectErr: true,
		},
		{
			port:        10,
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []MultipartFormDataRoute{
				{
					Path: "/foo",
				},
			},
			expectErr: true,
		},
		{
			port:        10,
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []MultipartFormDataRoute{
				{
					Path:    "/foo",
					Handler: func(_ *Context) error { return nil },
				},
				{
					Path:    "/foo",
					Handler: func(_ *Context) error { return nil },
				},
			},
			expectErr: true,
		},
		{
			port:        10,
			rootPath:    "/foo/",
			traceHeader: "foo",
			middlewares: []Middleware{
				{
					Priority: HighPriority,
				},
			},
			expectErr: true,
		},
		{
			port:        10,
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []MultipartFormDataRoute{
				{
					Path:    "/foo",
					Handler: func(_ *Context) error { return nil },
				},
			},
			middlewares: []Middleware{
				{
					Priority: HighPriority,
					Handler: func() echo.MiddlewareFunc {
						return func(next echo.HandlerFunc) echo.HandlerFunc {
							return func(c echo.Context) error {
								return next(c)
							}
						}
					}(),
				},
			},
		},
	} {
		mod := API{
			port:                    tc.port,
			rootPath:                tc.rootPath,
			traceHeader:             tc.traceHeader,
			multipartFormDataRoutes: tc.routes,
			externalMiddlewares:     tc.middlewares,
		}

		err := mod.Validate()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestAPI_Start(t *testing.T) {
	mod := new(API)
	mod.port = 3000
	mod.rootPath = "/"
	mod.multipartFormDataRoutes = []MultipartFormDataRoute{
		{
			Path: "/foo",
			Handler: func(ctx *Context) error {
				ctx.outputPaths = []string{
					"/tests/test/testdata/api/sample1.txt",
				}

				return nil
			},
		},
		{
			Path:    "/bar",
			Handler: func(_ *Context) error { return errors.New("foo") },
		},
	}
	mod.externalMiddlewares = []Middleware{
		{
			RunBeforeRouter: true,
			Handler: func() echo.MiddlewareFunc {
				return func(next echo.HandlerFunc) echo.HandlerFunc {
					return func(c echo.Context) error {
						return next(c)
					}
				}
			}(),
		},
		{
			Handler: func() echo.MiddlewareFunc {
				return func(next echo.HandlerFunc) echo.HandlerFunc {
					return func(c echo.Context) error {
						return next(c)
					}
				}
			}(),
		},
	}
	mod.logger = zap.NewNop()

	err := mod.Start()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	// health request.
	recorder := httptest.NewRecorder()
	healthRequest := httptest.NewRequest(http.MethodGet, "/health", nil)

	mod.srv.ServeHTTP(recorder, healthRequest)
	if recorder.Code != http.StatusOK {
		t.Errorf("expected %d status code but got %d", http.StatusOK, recorder.Code)
	}

	// "multipart/form-data" request.
	multipartRequest := func(URL string) *http.Request {
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

		req := httptest.NewRequest(http.MethodPost, URL, body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())

		return req
	}

	recorder = httptest.NewRecorder()
	mod.srv.ServeHTTP(recorder, multipartRequest("/forms/foo"))

	if recorder.Code != http.StatusOK {
		t.Errorf("expected %d status code but got %d", http.StatusOK, recorder.Code)
	}

	recorder = httptest.NewRecorder()
	mod.srv.ServeHTTP(recorder, multipartRequest("/forms/bar"))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected %d status code but got %d", http.StatusInternalServerError, recorder.Code)
	}

	err = mod.srv.Shutdown(context.TODO())
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestAPI_StartupMessage(t *testing.T) {
	mod := API{
		port: 3000,
	}

	actual := mod.StartupMessage()
	expect := "server listening on port 3000"

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestAPI_Stop(t *testing.T) {
	mod := API{
		port: 3000,
		multipartFormDataRoutes: []MultipartFormDataRoute{
			{
				Path:    "/foo",
				Handler: func(_ *Context) error { return nil },
			},
		},
		logger: zap.NewNop(),
	}

	err := mod.Start()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	err = mod.Stop(context.TODO())
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestAPI_GraceDuration(t *testing.T) {
	for i, tc := range []struct {
		mod    API
		expect time.Duration
	}{
		{
			mod: API{
				readTimeout:    time.Duration(1) * time.Second,
				processTimeout: time.Duration(1) * time.Second,
				writeTimeout:   time.Duration(1) * time.Second,
				disableWebhook: true,
			},
			expect: time.Duration(3) * time.Second,
		},
		{
			mod: API{
				readTimeout:         time.Duration(1) * time.Second,
				processTimeout:      time.Duration(1) * time.Second,
				writeTimeout:        time.Duration(1) * time.Second,
				webhookMaxRetry:     5,
				webhookRetryMaxWait: time.Duration(5) * time.Second,
			},
			expect: time.Duration(28) * time.Second,
		},
	} {
		actual := tc.mod.GraceDuration()

		if actual != tc.expect {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expect, actual)
		}
	}
}

// Interface guards.
var (
	_ gotenberg.Module         = (*ProtoModule)(nil)
	_ gotenberg.Validator      = (*ProtoValidator)(nil)
	_ gotenberg.Module         = (*ProtoValidator)(nil)
	_ MultipartFormDataRouter  = (*ProtoMultipartFormDataRouter)(nil)
	_ gotenberg.Module         = (*ProtoMultipartFormDataRouter)(nil)
	_ gotenberg.Validator      = (*ProtoMultipartFormDataRouter)(nil)
	_ MiddlewareProvider       = (*ProtoMiddlewareProvider)(nil)
	_ gotenberg.Module         = (*ProtoMiddlewareProvider)(nil)
	_ gotenberg.Validator      = (*ProtoMiddlewareProvider)(nil)
	_ HealthChecker            = (*ProtoHealthChecker)(nil)
	_ gotenberg.Module         = (*ProtoHealthChecker)(nil)
	_ gotenberg.Validator      = (*ProtoHealthChecker)(nil)
	_ gotenberg.LoggerProvider = (*ProtoLoggerProvider)(nil)
	_ gotenberg.Module         = (*ProtoLoggerProvider)(nil)
)
