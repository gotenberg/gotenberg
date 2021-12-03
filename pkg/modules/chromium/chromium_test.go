package chromium

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

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

	if len(metrics) != 1 {
		t.Fatalf("expected %d metrics, but got %d", 1, len(metrics))
	}

	actual := metrics[0].Read()
	if actual != 0 {
		t.Errorf("expected %d Chromium instances, but got %f", 0, actual)
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
	for i, tc := range []struct {
		timeout                  time.Duration
		cancel                   context.CancelFunc
		URL                      string
		options                  Options
		userAgent                string
		incognito                bool
		ignoreCertificateErrors  bool
		disableWebSecurity       bool
		allowFileAccessFromFiles bool
		proxyServer              string
		allowList                *regexp.Regexp
		denyList                 *regexp.Regexp
		disableJavaScript        bool
		expectErr                bool
	}{
		{
			URL:       "file:///tests/test/testdata/chromium/html/sample4/index.html",
			allowList: regexp.MustCompile("file:///tmp/*"),
			expectErr: true,
		},
		{
			URL:       "file:///tests/test/testdata/chromium/html/sample4/index.html",
			denyList:  regexp.MustCompile("file:///tests/*"),
			expectErr: true,
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				UserAgent: "foo",
			},
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample10/index.html",
			options: Options{
				FailOnConsoleExceptions: true,
			},
			expectErr: true,
		},
		{
			URL:               "file:///tests/test/testdata/chromium/html/sample9/index.html",
			disableJavaScript: true,
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				ExtraHTTPHeaders: map[string]string{
					"foo": "bar",
				},
			},
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample11/index.html",
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
			URL: "file:///tests/test/testdata/chromium/html/sample8/index.html",
			options: Options{
				EmulatedMediaType: "foo",
			},
			expectErr: true,
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample8/index.html",
			options: Options{
				EmulatedMediaType: "screen",
			},
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample8/index.html",
			options: Options{
				EmulatedMediaType: "print",
			},
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample11/index.html",
			options: Options{
				ExtraScriptTags: []ScriptTag{
					{
						Src: "script.js",
					},
				},
			},
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				WaitDelay: time.Duration(1) * time.Nanosecond,
			},
		},
		{
			timeout: time.Duration(3) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample2/index.html",
			options: Options{
				WaitWindowStatus: "foo",
			},
			expectErr: true,
		},
		{
			timeout: time.Duration(3) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample2/index.html",
			options: Options{
				WaitWindowStatus: "ready",
			},
		},
		{
			timeout: time.Duration(3) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample2/index.html",
			options: Options{
				WaitForExpression: "window.status === 'foo'",
			},
			expectErr: true,
		},
		{
			timeout: time.Duration(3) * time.Second,
			URL:     "file:///tests/test/testdata/chromium/html/sample2/index.html",
			options: Options{
				WaitForExpression: "window.status === 'ready'",
			},
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				WaitForExpression: "return undefined",
			},
			expectErr: true,
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				MarginBottom: 100,
			},
			expectErr: true,
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				PageRanges: "foo",
			},
			expectErr: true,
		},
		{
			URL:                      "file:///tests/test/testdata/chromium/html/sample4/index.html",
			userAgent:                "foo",
			incognito:                true,
			ignoreCertificateErrors:  true,
			disableWebSecurity:       true,
			allowFileAccessFromFiles: true,
			proxyServer:              "foo",
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample1/index.html",
		},
		{
			URL:       "file:///tests/test/testdata/chromium/html/sample3/index.html",
			allowList: regexp.MustCompile("file:///tests/*"),
		},
		{
			URL:      "file:///tests/test/testdata/chromium/html/sample3/index.html",
			denyList: regexp.MustCompile("file:///etc/*"),
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample4/index.html",
			options: Options{
				HeaderTemplate: func() string {
					b, err := ioutil.ReadFile("/tests/test/testdata/chromium/url/sample2/header.html")
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}

					return string(b)
				}(),
				FooterTemplate: func() string {
					b, err := ioutil.ReadFile("/tests/test/testdata/chromium/url/sample2/footer.html")
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}

					return string(b)
				}(),
			},
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample5/index.html",
		},
		{
			URL:                      "file:///tests/test/testdata/chromium/html/sample6/index.html",
			allowFileAccessFromFiles: true,
		},
		{
			URL: "file:///tests/test/testdata/chromium/html/sample7/index.html",
		},
	} {
		func() {
			mod := new(Chromium)
			mod.binPath = os.Getenv("CHROMIUM_BIN_PATH")
			mod.userAgent = tc.userAgent
			mod.incognito = tc.incognito
			mod.ignoreCertificateErrors = tc.ignoreCertificateErrors
			mod.disableWebSecurity = tc.disableWebSecurity
			mod.allowFileAccessFromFiles = tc.allowFileAccessFromFiles
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
				t.Fatalf("test %d: expected error but got: %v", i, err)
			}

			defer func() {
				err := os.RemoveAll(outputDir)
				if err != nil {
					t.Fatalf("test %d: expected no error but got: %v", i, err)
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
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
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
