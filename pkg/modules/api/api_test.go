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
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestApi_Descriptor(t *testing.T) {
	descriptor := new(Api).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Api))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestApi_Provision(t *testing.T) {
	for _, tc := range []struct {
		scenario          string
		ctx               *gotenberg.Context
		setEnv            func()
		expectPort        int
		expectMiddlewares []Middleware
		expectError       bool
	}{
		{
			scenario: "port from env: non-existing environment variable",
			ctx: func() *gotenberg.Context {
				fs := new(Api).Descriptor().FlagSet
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
			expectError: true,
		},
		{
			scenario: "port from env: invalid environment variable value",
			ctx: func() *gotenberg.Context {
				fs := new(Api).Descriptor().FlagSet
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
			setEnv: func() {
				err := os.Setenv("PORT", "foo")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			},
			expectError: true,
		},
		{
			scenario: "basic auth: non-existing GOTENBERG_API_BASIC_AUTH_USERNAME environment variable",
			ctx: func() *gotenberg.Context {
				fs := new(Api).Descriptor().FlagSet
				err := fs.Parse([]string{"--api-enable-basic-auth=true"})
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
			expectError: true,
		},
		{
			scenario: "basic auth: non-existing GOTENBERG_API_BASIC_AUTH_PASSWORD environment variable",
			ctx: func() *gotenberg.Context {
				fs := new(Api).Descriptor().FlagSet
				err := fs.Parse([]string{"--api-enable-basic-auth=true"})
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
			setEnv: func() {
				err := os.Setenv("GOTENBERG_API_BASIC_AUTH_USERNAME", "foo")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			},
			expectError: true,
		},
		{
			scenario: "no valid routers",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					RouterMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.ValidateMock = func() error {
					return errors.New("foo")
				}
				mod.RoutesMock = func() ([]Route, error) {
					return nil, nil
				}
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Api).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "cannot retrieve routes from router",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					RouterMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.RoutesMock = func() ([]Route, error) {
					return nil, errors.New("foo")
				}
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Api).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "no valid middleware providers",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					MiddlewareProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.ValidateMock = func() error {
					return errors.New("foo")
				}
				mod.MiddlewaresMock = func() ([]Middleware, error) {
					return nil, nil
				}
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Api).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "cannot retrieve middlewares from middleware provider",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					MiddlewareProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.MiddlewaresMock = func() ([]Middleware, error) {
					return nil, errors.New("foo")
				}
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Api).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "no valid health checkers",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					HealthCheckerMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.ValidateMock = func() error {
					return errors.New("foo")
				}
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Api).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "cannot retrieve health checks from health checker",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					HealthCheckerMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.ChecksMock = func() ([]health.CheckerOption, error) {
					return nil, errors.New("foo")
				}
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Api).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "no logger provider",
			ctx: func() *gotenberg.Context {
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Api).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "no logger from logger provider",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return nil, errors.New("foo")
				}
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Api).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectError: true,
		},
		{
			scenario: "success",
			ctx: func() *gotenberg.Context {
				mod1 := &struct {
					gotenberg.ModuleMock
					RouterMock
				}{}
				mod1.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod1 }}
				}
				mod1.RoutesMock = func() ([]Route, error) {
					return []Route{{}}, nil
				}

				mod2 := &struct {
					gotenberg.ModuleMock
					MiddlewareProviderMock
				}{}
				mod2.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod2 }}
				}
				mod2.MiddlewaresMock = func() ([]Middleware, error) {
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

				mod3 := &struct {
					gotenberg.ModuleMock
					HealthCheckerMock
				}{}
				mod3.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "baz", New: func() gotenberg.Module { return mod3 }}
				}
				mod3.ChecksMock = func() ([]health.CheckerOption, error) {
					return []health.CheckerOption{health.WithDisabledAutostart()}, nil
				}
				mod3.ReadyMock = func() error {
					return nil
				}

				mod4 := &struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				mod4.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "qux", New: func() gotenberg.Module { return mod4 }}
				}
				mod4.LoggerMock = func(_ gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}

				fs := new(Api).Descriptor().FlagSet
				err := fs.Parse([]string{"--api-port-from-env=PORT", "--api-enable-basic-auth=true"})
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					[]gotenberg.ModuleDescriptor{
						mod1.Descriptor(),
						mod2.Descriptor(),
						mod3.Descriptor(),
						mod4.Descriptor(),
					},
				)
			}(),
			setEnv: func() {
				err := os.Setenv("PORT", "1337")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				err = os.Setenv("GOTENBERG_API_BASIC_AUTH_USERNAME", "foo")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				err = os.Setenv("GOTENBERG_API_BASIC_AUTH_PASSWORD", "bar")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			},
			expectPort: 1337,
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
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.setEnv != nil {
				tc.setEnv()
			}

			mod := new(Api)
			err := mod.Provision(tc.ctx)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectPort != 0 && mod.port != tc.expectPort {
				t.Errorf("expected port %d but got %d", tc.expectPort, mod.port)
			}

			if !reflect.DeepEqual(mod.externalMiddlewares, tc.expectMiddlewares) {
				t.Errorf("expected %+v, but got: %+v", tc.expectMiddlewares, mod.externalMiddlewares)
			}
		})
	}
}

func TestApi_Validate(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		port        int
		bindIp      string
		tlsCertFile string
		tlsKeyFile  string
		rootPath    string
		traceHeader string
		routes      []Route
		middlewares []Middleware
		expectError bool
	}{
		{
			scenario:    "invalid port (< 1)",
			port:        0,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes:      nil,
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid port (> 65535)",
			port:        65536,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes:      nil,
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid IP",
			port:        10,
			bindIp:      "foo",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes:      nil,
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid TLS files: only cert file provided",
			port:        10,
			bindIp:      "127.0.0.1",
			tlsCertFile: "cert.pem",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes:      nil,
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid TLS files: only key file provided",
			port:        10,
			bindIp:      "127.0.0.1",
			tlsKeyFile:  "key.pem",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes:      nil,
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid root path: missing / prefix",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "foo/",
			traceHeader: "foo",
			routes:      nil,
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid root path: missing / suffix",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo",
			traceHeader: "foo",
			routes:      nil,
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid trace header",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "",
			routes:      nil,
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid route: empty path",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []Route{
				{
					Path: "",
				},
			},
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid route: missing / prefix in path",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []Route{
				{
					Path: "foo",
				},
			},
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid multipart route: no /forms prefix in path",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []Route{
				{
					Path:        "/foo",
					IsMultipart: true,
				},
			},
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid route: no method",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []Route{
				{
					Path:   "/foo",
					Method: "",
				},
			},
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid route: nil handler",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []Route{
				{
					Method:  http.MethodPost,
					Path:    "/foo",
					Handler: nil,
				},
			},
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid route: path already existing",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []Route{
				{
					Method:  http.MethodPost,
					Path:    "/foo",
					Handler: func(_ echo.Context) error { return nil },
				},
				{
					Method:  http.MethodPost,
					Path:    "/foo",
					Handler: func(_ echo.Context) error { return nil },
				},
			},
			middlewares: nil,
			expectError: true,
		},
		{
			scenario:    "invalid middleware: nil handler",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes:      nil,
			middlewares: []Middleware{
				{
					Priority: HighPriority,
					Handler:  nil,
				},
			},
			expectError: true,
		},
		{
			scenario:    "success",
			port:        10,
			bindIp:      "127.0.0.1",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []Route{
				{
					Method:  http.MethodGet,
					Path:    "/foo",
					Handler: func(_ echo.Context) error { return nil },
				},
				{
					Method:      http.MethodGet,
					Path:        "/forms/foo",
					Handler:     func(_ echo.Context) error { return nil },
					IsMultipart: true,
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
		{
			scenario:    "success with TLS",
			port:        10,
			tlsCertFile: "cert.pem",
			tlsKeyFile:  "key.pem",
			rootPath:    "/foo/",
			traceHeader: "foo",
			routes: []Route{
				{
					Method:  http.MethodGet,
					Path:    "/foo",
					Handler: func(_ echo.Context) error { return nil },
				},
				{
					Method:      http.MethodGet,
					Path:        "/forms/foo",
					Handler:     func(_ echo.Context) error { return nil },
					IsMultipart: true,
				},
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := Api{
				port:                tc.port,
				bindIp:              tc.bindIp,
				tlsCertFile:         tc.tlsCertFile,
				tlsKeyFile:          tc.tlsKeyFile,
				rootPath:            tc.rootPath,
				traceHeader:         tc.traceHeader,
				routes:              tc.routes,
				externalMiddlewares: tc.middlewares,
			}

			err := mod.Validate()
			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestApi_Start(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		readyFn     []func() error
		tlsCertFile string
		tlsKeyFile  string
		expectError bool
	}{
		{
			scenario: "at least one module not ready",
			readyFn: []func() error{
				func() error { return nil },
				func() error { return errors.New("not ready") },
			},
			expectError: true,
		},
		{
			scenario: "success",
			readyFn: []func() error{
				func() error { return nil },
				func() error { return nil },
			},
			expectError: false,
		},
		{
			scenario: "success with TLS",
			readyFn: []func() error{
				func() error { return nil },
				func() error { return nil },
			},
			tlsCertFile: "/tests/test/testdata/api/cert.pem",
			tlsKeyFile:  "/tests/test/testdata/api/key.pem",
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Api)
			mod.port = 3000
			mod.tlsCertFile = tc.tlsCertFile
			mod.tlsKeyFile = tc.tlsKeyFile
			mod.startTimeout = time.Duration(30) * time.Second
			mod.rootPath = "/"
			mod.basicAuthUsername = "foo"
			mod.basicAuthPassword = "bar"
			mod.disableHealthCheckLogging = true
			mod.enableDebugRoute = true
			mod.routes = []Route{
				{
					Method:         http.MethodPost,
					Path:           "/forms/foo",
					IsMultipart:    true,
					DisableLogging: true,
					Handler: func(c echo.Context) error {
						ctx := c.Get("context").(*Context)
						ctx.outputPaths = []string{
							"/tests/test/testdata/api/sample1.txt",
						}

						return nil
					},
				},
				{
					Method:      http.MethodPost,
					Path:        "/forms/bar",
					IsMultipart: true,
					Handler:     func(_ echo.Context) error { return errors.New("foo") },
				},
			}
			mod.externalMiddlewares = []Middleware{
				{
					Stack: PreRouterStack,
					Handler: func() echo.MiddlewareFunc {
						return func(next echo.HandlerFunc) echo.HandlerFunc {
							return func(c echo.Context) error {
								return next(c)
							}
						}
					}(),
				},
				{
					Stack: MultipartStack,
					Handler: func() echo.MiddlewareFunc {
						return func(next echo.HandlerFunc) echo.HandlerFunc {
							return func(c echo.Context) error {
								return next(c)
							}
						}
					}(),
				},
				{
					Stack: DefaultStack,
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
			mod.readyFn = tc.readyFn
			mod.fs = gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
			mod.logger = zap.NewNop()

			err := mod.Start()
			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectError {
				return
			}

			// root request.
			recorder := httptest.NewRecorder()
			rootRequest := httptest.NewRequest(http.MethodGet, "/", nil)
			rootRequest.SetBasicAuth(mod.basicAuthUsername, mod.basicAuthPassword)
			mod.srv.ServeHTTP(recorder, rootRequest)
			if recorder.Code != http.StatusOK {
				t.Errorf("expected %d status code but got %d", http.StatusOK, recorder.Code)
			}

			// favicon request.
			recorder = httptest.NewRecorder()
			faviconRequest := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
			faviconRequest.SetBasicAuth(mod.basicAuthUsername, mod.basicAuthPassword)
			mod.srv.ServeHTTP(recorder, faviconRequest)
			if recorder.Code != http.StatusNoContent {
				t.Errorf("expected %d status code but got %d", http.StatusNoContent, recorder.Code)
			}

			// health requests.
			recorder = httptest.NewRecorder()
			healthGetRequest := httptest.NewRequest(http.MethodGet, "/health", nil)
			mod.srv.ServeHTTP(recorder, healthGetRequest)
			if recorder.Code != http.StatusOK {
				t.Errorf("expected %d status code but got %d", http.StatusOK, recorder.Code)
			}

			recorder = httptest.NewRecorder()
			healthHeadRequest := httptest.NewRequest(http.MethodHead, "/health", nil)
			mod.srv.ServeHTTP(recorder, healthHeadRequest)
			if recorder.Code != http.StatusOK {
				t.Errorf("expected %d status code but got %d", http.StatusOK, recorder.Code)
			}

			// version request.
			recorder = httptest.NewRecorder()
			versionRequest := httptest.NewRequest(http.MethodGet, "/version", nil)
			versionRequest.SetBasicAuth(mod.basicAuthUsername, mod.basicAuthPassword)
			mod.srv.ServeHTTP(recorder, versionRequest)
			if recorder.Code != http.StatusOK {
				t.Errorf("expected %d status code but got %d", http.StatusOK, recorder.Code)
			}

			// debug request.
			recorder = httptest.NewRecorder()
			debugRequest := httptest.NewRequest(http.MethodGet, "/debug", nil)
			debugRequest.SetBasicAuth(mod.basicAuthUsername, mod.basicAuthPassword)
			mod.srv.ServeHTTP(recorder, debugRequest)
			if recorder.Code != http.StatusOK {
				t.Errorf("expected %d status code but got %d", http.StatusOK, recorder.Code)
			}

			// "multipart/form-data" request.
			multipartRequest := func(url string) *http.Request {
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

				req := httptest.NewRequest(http.MethodPost, url, body)
				req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
				req.SetBasicAuth(mod.basicAuthUsername, mod.basicAuthPassword)

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
		})
	}
}

func TestApi_StartupMessage(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		port          int
		bindIp        string
		expectMessage string
	}{
		{
			scenario:      "no custom IP",
			port:          3000,
			bindIp:        "",
			expectMessage: "server started on [::]:3000",
		},
		{
			scenario:      "custom IP",
			port:          3000,
			bindIp:        "127.0.0.1",
			expectMessage: "server started on 127.0.0.1:3000",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := Api{
				port:   tc.port,
				bindIp: tc.bindIp,
			}

			actual := mod.StartupMessage()
			if actual != tc.expectMessage {
				t.Errorf("expected '%s' but got '%s'", tc.expectMessage, actual)
			}
		})
	}
}

func TestApi_Stop(t *testing.T) {
	mod := &Api{
		port: 3000,
		routes: []Route{
			{
				Method:  http.MethodGet,
				Path:    "/foo",
				Handler: func(_ echo.Context) error { return nil },
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
