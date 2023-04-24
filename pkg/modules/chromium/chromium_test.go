package chromium

import (
	"context"
	"errors"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

type ProtoModule struct {
	descriptor func() gotenberg.ModuleDescriptor
}

func (mod ProtoModule) Descriptor() gotenberg.ModuleDescriptor {
	return mod.descriptor()
}

type ProtoAPI struct {
	pdf func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error
}

func (mod ProtoAPI) PDF(ctx context.Context, logger *zap.Logger, URL, outputPath string, options Options) error {
	return mod.pdf(ctx, logger, URL, outputPath, options)
}

type ProtoPDFEngineProvider struct {
	ProtoModule
	pdfEngine func() (gotenberg.PDFEngine, error)
}

func (mod ProtoPDFEngineProvider) PDFEngine() (gotenberg.PDFEngine, error) {
	return mod.pdfEngine()
}

type ProtoPDFEngine struct {
	merge   func(_ context.Context, _ *zap.Logger, _ []string, _ string) error
	convert func(_ context.Context, _ *zap.Logger, _, _, _ string) error
}

func (mod ProtoPDFEngine) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	return mod.merge(ctx, logger, inputPaths, outputPath)
}

func (mod ProtoPDFEngine) Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
	return mod.convert(ctx, logger, format, inputPath, outputPath)
}

func TestDefaultOptions(t *testing.T) {
	actual := DefaultOptions()
	notExpect := Options{}

	if reflect.DeepEqual(actual, notExpect) {
		t.Errorf("expected %v and got identical %v", actual, notExpect)
	}
}

func TestChromium_Descriptor(t *testing.T) {
	descriptor := Chromium{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Chromium))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestChromium_Provision(t *testing.T) {
	for i, tc := range []struct {
		ctx       *gotenberg.Context
		expectErr bool
	}{
		{
			ctx: func() *gotenberg.Context {
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Chromium).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoPDFEngineProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.pdfEngine = func() (gotenberg.PDFEngine, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Chromium).Descriptor().FlagSet,
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
				mod := struct{ ProtoPDFEngineProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.pdfEngine = func() (gotenberg.PDFEngine, error) {
					return struct{ ProtoPDFEngine }{}, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Chromium).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
		},
	} {
		mod := new(Chromium)
		err := mod.Provision(tc.ctx)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestChromium_Validate(t *testing.T) {
	for i, tc := range []struct {
		binPath   string
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
			binPath: os.Getenv("CHROMIUM_BIN_PATH"),
		},
	} {
		mod := new(Chromium)
		mod.binPath = tc.binPath
		err := mod.Validate()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestChromium_Metrics(t *testing.T) {
	metrics, err := new(Chromium).Metrics()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("expected %d metrics, but got %d", 2, len(metrics))
	}

	actual := metrics[0].Read()
	if actual != 0 {
		t.Errorf("expected %d Chromium instances, but got %f", 0, actual)
	}

	actual = metrics[1].Read()
	if actual != 0 {
		t.Errorf("expected %d Chromium failed starts, but got %f", 0, actual)
	}
}

func TestChromium_Checks(t *testing.T) {
	tests := []struct {
		name                     string
		mod                      Chromium
		tearUp                   func()
		tearDown                 func()
		expectAvailabilityStatus health.AvailabilityStatus
	}{
		{
			name: "ignore Chromium failed starts",
			mod: Chromium{
				failedStartsThreshold: 0,
			},
		},
		{
			name: "with Chromium failed starts threshold not reached",
			mod: Chromium{
				failedStartsThreshold: 1,
			},
			expectAvailabilityStatus: health.StatusUp,
		},
		{
			name: "with Chromium failed starts threshold reached",
			mod: Chromium{
				failedStartsThreshold: 1,
			},
			tearUp: func() {
				failedStartsCount = 1
			},
			tearDown: func() {
				failedStartsCount = 0
			},
			expectAvailabilityStatus: health.StatusDown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.tearUp != nil {
				tc.tearUp()
			}

			checks, err := tc.mod.Checks()
			if err != nil {
				t.Fatalf("expected no error from mod.Checks(), but got: %v", err)
			}

			if len(checks) == 0 {
				return
			}

			if len(checks) != 1 {
				t.Fatalf("expected 1 check from mod.Checks(), but got %d", len(checks))
			}

			checker := health.NewChecker(checks...)
			result := checker.Check(context.Background())

			if result.Status != tc.expectAvailabilityStatus {
				t.Errorf("expected '%s' as availability status, but got '%s'", tc.expectAvailabilityStatus, result.Status)
			}

			if tc.tearDown != nil {
				tc.tearDown()
			}
		})
	}
}

func TestChromium_Chromium(t *testing.T) {
	mod := new(Chromium)

	_, err := mod.Chromium()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestChromium_Routes(t *testing.T) {
	for i, tc := range []struct {
		expectRoutes  int
		disableRoutes bool
	}{
		{
			expectRoutes: 3,
		},
		{
			disableRoutes: true,
		},
	} {
		mod := new(Chromium)
		mod.disableRoutes = tc.disableRoutes

		routes, err := mod.Routes()
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		if tc.expectRoutes != len(routes) {
			t.Errorf("test %d: expected %d routes but got %d", i, tc.expectRoutes, len(routes))
		}
	}
}

func TestChromium_PDF(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		timeout                  time.Duration
		cancel                   context.CancelFunc
		URL                      string
		options                  Options
		userAgent                string
		incognito                bool
		allowInsecureLocalhost   bool
		ignoreCertificateErrors  bool
		disableWebSecurity       bool
		allowFileAccessFromFiles bool
		hostResolverRules        string
		proxyServer              string
		allowList                *regexp.Regexp
		denyList                 *regexp.Regexp
		disableJavaScript        bool
		expectErr                bool
	}{
		{
			name:      "context has no deadline",
			URL:       "file:///tests/test/testdata/chromium/html/sample1/index.html",
			expectErr: true,
		},
		{
			name:      "URL does not match the expression from the allowed list",
			timeout:   time.Duration(60) * time.Second,
			URL:       "file:///tests/test/testdata/chromium/html/sample4/index.html",
			allowList: regexp.MustCompile("file:///tmp/*"),
			expectErr: true,
		},
		{
			name:      "URL does not match the expression from the denied list",
			timeout:   time.Duration(60) * time.Second,
			URL:       "file:///tests/test/testdata/chromium/html/sample4/index.html",
			denyList:  regexp.MustCompile("file:///tests/*"),
			expectErr: true,
		},
		{
			name:    "with user agent",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				UserAgent: "foo",
			},
		},
		{
			name:    "fail on console exceptions",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample10/index.html",
			options: Options{
				FailOnConsoleExceptions: true,
			},
			expectErr: true,
		},
		{
			name:              "disable JavaScript",
			timeout:           time.Duration(60) * time.Second,
			URL:               "file:///tests/test/testdata/chromium/html/sample9/index.html",
			disableJavaScript: true,
		},
		{
			name:    "with extra HTTP headers",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				ExtraHTTPHeaders: map[string]string{
					"foo": "bar",
				},
			},
		},
		{
			name:    "with extra link tags",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample11/index.html",
			options: Options{
				ExtraLinkTags: []LinkTag{
					{
						Href: "font.woff",
					},
					{
						Href: "style.css",
					},
				},
			},
		},
		{
			name:    "with invalid emulated media type",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample8/index.html",
			options: Options{
				EmulatedMediaType: "foo",
			},
			expectErr: true,
		},
		{
			name:    "with screen emulated media type",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample8/index.html",
			options: Options{
				EmulatedMediaType: "screen",
			},
		},
		{
			name:    "with print emulated media type",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample8/index.html",
			options: Options{
				EmulatedMediaType: "print",
			},
		},
		{
			name:    "with omit background but not print background",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				OmitBackground: true,
			},
			expectErr: true,
		},
		{
			name:    "with omit background and print background",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				OmitBackground:  true,
				PrintBackground: true,
			},
		},
		{
			name:    "with extra script tags",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample11/index.html",
			options: Options{
				ExtraScriptTags: []ScriptTag{
					{
						Src: "script.js",
					},
				},
			},
		},
		{
			name:    "with wait delay",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				WaitDelay: time.Duration(1) * time.Nanosecond,
			},
		},
		{
			name:    "with invalid wait window status",
			timeout: time.Duration(3) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample2/index.html",
			options: Options{
				WaitWindowStatus: "foo",
			},
			expectErr: true,
		},
		{
			name:    "with wait window status",
			timeout: time.Duration(3) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample2/index.html",
			options: Options{
				WaitWindowStatus: "ready",
			},
		},
		{
			name:    "with wait for expression that should not happen",
			timeout: time.Duration(3) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample2/index.html",
			options: Options{
				WaitForExpression: "window.status === 'foo'",
			},
			expectErr: true,
		},
		{
			name:    "with valid wait for expression",
			timeout: time.Duration(3) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample2/index.html",
			options: Options{
				WaitForExpression: "window.status === 'ready'",
			},
		},
		{
			name:    "with invalid wait for expression",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				WaitForExpression: "return undefined",
			},
			expectErr: true,
		},
		{
			name:    "with too big margin bottom",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				MarginBottom: 100,
			},
			expectErr: true,
		},
		{
			name:    "with invalid page ranges",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				PageRanges: "foo",
			},
			expectErr: true,
		},
		{
			name:                     "with a lot of properties",
			timeout:                  time.Duration(60) * time.Second,
			URL:                      "file:///tests/test/testdata/chromium/html/sample4/index.html",
			userAgent:                "foo",
			incognito:                true,
			ignoreCertificateErrors:  true,
			allowInsecureLocalhost:   true,
			disableWebSecurity:       true,
			allowFileAccessFromFiles: true,
			hostResolverRules:        "foo",
			proxyServer:              "foo",
		},
		{
			name:    "with file using local and remote assets",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample1/index.html",
		},
		{
			name:      "URL does match the expression from the allowed list",
			timeout:   time.Duration(60) * time.Second,
			URL:       "file:///tests/test/testdata/chromium/html/sample3/index.html",
			allowList: regexp.MustCompile("file:///tests/*"),
		},
		{
			name:     "URL does match the expression from the denied list",
			timeout:  time.Duration(60) * time.Second,
			URL:      "file:///tests/test/testdata/chromium/html/sample3/index.html",
			denyList: regexp.MustCompile("file:///etc/*"),
		},
		{
			name:    "with custom header and footer templates",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				HeaderTemplate: func() string {
					b, err := os.ReadFile("/tests/test/testdata/chromium/url/sample2/header.html")
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}

					return string(b)
				}(),
				FooterTemplate: func() string {
					b, err := os.ReadFile("/tests/test/testdata/chromium/url/sample2/footer.html")
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}

					return string(b)
				}(),
			},
		},
		{
			name:    "with custom header template only",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				HeaderTemplate: func() string {
					b, err := os.ReadFile("/tests/test/testdata/chromium/url/sample2/header.html")
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}

					return string(b)
				}(),
				FooterTemplate: DefaultOptions().FooterTemplate,
			},
		},
		{
			name:    "with custom footer template only",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				HeaderTemplate: DefaultOptions().HeaderTemplate,
				FooterTemplate: func() string {
					b, err := os.ReadFile("/tests/test/testdata/chromium/url/sample2/footer.html")
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}

					return string(b)
				}(),
			},
		},
		{
			name:    "without custom header and footer templates",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				HeaderTemplate: DefaultOptions().HeaderTemplate,
				FooterTemplate: DefaultOptions().FooterTemplate,
			},
		},
		{
			name:    "with file using a .gif",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample5/index.html",
		},
		{
			name:                     "with allow file access from files",
			timeout:                  time.Duration(60) * time.Second,
			URL:                      "file:///tests/test/testdata/chromium/html/sample6/index.html",
			allowFileAccessFromFiles: true,
		},
		{
			name:    "with file using a style attribute",
			timeout: time.Duration(60) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample7/index.html",
		},
	} {
		func() {
			mod := new(Chromium)
			mod.binPath = os.Getenv("CHROMIUM_BIN_PATH")
			mod.userAgent = tc.userAgent
			mod.incognito = tc.incognito
			mod.allowInsecureLocalhost = tc.allowInsecureLocalhost
			mod.ignoreCertificateErrors = tc.ignoreCertificateErrors
			mod.disableWebSecurity = tc.disableWebSecurity
			mod.allowFileAccessFromFiles = tc.allowFileAccessFromFiles
			mod.hostResolverRules = tc.hostResolverRules
			mod.proxyServer = tc.proxyServer

			if tc.allowList == nil {
				tc.allowList = regexp.MustCompile("")
			}

			if tc.denyList == nil {
				tc.denyList = regexp.MustCompile("")
			}

			mod.allowList = tc.allowList
			mod.denyList = tc.denyList
			mod.disableJavaScript = tc.disableJavaScript

			outputDir, err := gotenberg.MkdirAll()
			if err != nil {
				t.Fatalf("test %s: expected error but got: %v", tc.name, err)
			}

			defer func() {
				err := os.RemoveAll(outputDir)
				if err != nil {
					t.Fatalf("test %s: expected no error but got: %v", tc.name, err)
				}
			}()

			if tc.timeout == 0 {
				err = mod.PDF(context.Background(), zap.NewNop(), tc.URL, outputDir+"/foo.pdf", tc.options)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
				defer cancel()

				err = mod.PDF(ctx, zap.NewNop(), tc.URL, outputDir+"/foo.pdf", tc.options)
			}

			if tc.expectErr && err == nil {
				t.Errorf("test %s: expected error but got: %v", tc.name, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %s: expected no error but got: %v", tc.name, err)
			}
		}()
	}
}

// Interface guards.
var (
	_ gotenberg.Module            = (*ProtoModule)(nil)
	_ API                         = (*ProtoAPI)(nil)
	_ gotenberg.PDFEngineProvider = (*ProtoPDFEngineProvider)(nil)
	_ gotenberg.Module            = (*ProtoPDFEngineProvider)(nil)
	_ gotenberg.PDFEngine         = (*ProtoPDFEngine)(nil)
)
