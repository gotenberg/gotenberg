package chromium

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestChromiumBrowser_Start(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		browser     browser
		expectError bool
		cleanup     bool
	}{
		{
			scenario: "successful start",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
				},
			),
			expectError: false,
			cleanup:     true,
		},
		{
			scenario: "all browser arguments",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:                  os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout:         5 * time.Second,
					incognito:                true,
					allowInsecureLocalhost:   true,
					ignoreCertificateErrors:  true,
					disableWebSecurity:       true,
					allowFileAccessFromFiles: true,
					hostResolverRules:        "MAP forgery.docker.localhost traefik",
					proxyServer:              "1.2.3.4",
				},
			),
			expectError: false,
			cleanup:     true,
		},
		{
			scenario: "browser already started",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.isStarted.Store(true)
				return b
			}(),
			expectError: true,
			cleanup:     false,
		},
		{
			scenario: "browser start error",
			browser: func() browser {
				b := newChromiumBrowser(
					browserArguments{
						binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
						wsUrlReadTimeout: 5 * time.Second,
					},
				).(*chromiumBrowser)

				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				b.initialCtx = ctx

				return b
			}(),
			expectError: true,
			cleanup:     false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := zap.NewNop()
			err := tc.browser.Start(logger)

			if tc.cleanup {
				defer func(b browser, logger *zap.Logger) {
					err = b.Stop(logger)
					if err != nil {
						t.Fatalf("expected no error while cleaning up, but got: %v", err)
					}
				}(tc.browser, logger)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestChromiumBrowser_Stop(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		browser     browser
		setup       func(browser browser, logger *zap.Logger) error
		expectError bool
	}{
		{
			scenario: "successful stop",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
				},
			),
			setup: func(b browser, logger *zap.Logger) error {
				return b.Start(logger)
			},
			expectError: false,
		},
		{
			scenario: "browser already stopped",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.isStarted.Store(false)
				return b
			}(),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := zap.NewNop()

			if tc.setup != nil {
				err := tc.setup(tc.browser, logger)
				if err != nil {
					t.Fatalf("setup error: %v", err)
				}
			}

			err := tc.browser.Stop(logger)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestChromiumBrowser_Healthy(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		browser       browser
		setup         func(browser browser, logger *zap.Logger) error
		expectHealthy bool
		cleanup       bool
	}{
		{
			scenario: "healthy browser",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
				},
			),
			setup: func(b browser, logger *zap.Logger) error {
				return b.Start(logger)
			},
			expectHealthy: true,
			cleanup:       true,
		},
		{
			scenario: "browser not started",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.isStarted.Store(false)
				return b
			}(),
			expectHealthy: false,
			cleanup:       false,
		},
		{
			scenario: "unhealthy browser",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
				},
			),
			setup: func(b browser, logger *zap.Logger) error {
				_ = b.Start(logger)
				b.(*chromiumBrowser).cancelFunc()

				return nil
			},
			expectHealthy: false,
			cleanup:       true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := zap.NewNop()

			if tc.setup != nil {
				err := tc.setup(tc.browser, logger)
				if err != nil {
					t.Fatalf("setup error: %v", err)
				}
			}

			if tc.cleanup {
				defer func(b browser, logger *zap.Logger) {
					err := b.Stop(logger)
					if err != nil {
						t.Fatalf("expected no error while cleaning up, but got: %v", err)
					}
				}(tc.browser, logger)
			}

			healthy := tc.browser.Healthy(logger)

			if !tc.expectHealthy && healthy {
				t.Fatal("expected unhealthy browser but got an healthy one")
			}

			if tc.expectHealthy && !healthy {
				t.Fatal("expected a healthy browser but got an unhealthy one")
			}
		})
	}
}

func TestChromiumBrowser_pdf(t *testing.T) {
	for _, tc := range []struct {
		scenario           string
		browser            browser
		fs                 *gotenberg.FileSystem
		options            PdfOptions
		noDeadline         bool
		start              bool
		expectError        bool
		expectedError      error
		expectedLogEntries []string
	}{
		{
			scenario: "browser not started",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.isStarted.Store(false)
				return b
			}(),
			fs:          gotenberg.NewFileSystem(),
			noDeadline:  false,
			start:       false,
			expectError: true,
		},
		{
			scenario: "context has no deadline",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.isStarted.Store(true)
				return b
			}(),
			fs:          gotenberg.NewFileSystem(),
			noDeadline:  true,
			start:       false,
			expectError: true,
		},
		{
			scenario: "ErrFiltered: main URL does not match the allowed list",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.arguments = browserArguments{
					allowList: regexp2.MustCompile(`^file:(?!//\/tmp/).*`, 0),
					denyList:  regexp2.MustCompile("", 0),
				}
				b.isStarted.Store(true)
				return b
			}(),
			fs:            gotenberg.NewFileSystem(),
			noDeadline:    false,
			start:         false,
			expectError:   true,
			expectedError: gotenberg.ErrFiltered,
		},
		{
			scenario: "ErrFiltered: main URL does match the denied list",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.arguments = browserArguments{
					allowList: regexp2.MustCompile("", 0),
					denyList:  regexp2.MustCompile("^file:///tmp.*", 0),
				}
				b.isStarted.Store(true)
				return b
			}(),
			fs:            gotenberg.NewFileSystem(),
			noDeadline:    false,
			start:         false,
			expectError:   true,
			expectedError: gotenberg.ErrFiltered,
		},
		{
			scenario: "a request does not match the allowed list",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("^file:///tmp.*", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<iframe src='file:///etc/passwd'></iframe>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"'file:///etc/passwd' does not match the expression from the allowed list",
			},
		},
		{
			scenario: "a request does match the denied list",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile(`^file:(?!//\/tmp/).*`, 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<iframe src='file:///etc/passwd'></iframe>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"'file:///etc/passwd' matches the expression from the denied list",
			},
		},
		{
			scenario: "skip networkIdle event",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Skip networkIdle event</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{SkipNetworkIdleEvent: true},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"skipping network idle event",
			},
		},
		{
			scenario: "ErrInvalidHttpStatusCode",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrInvalidHttpStatusCode</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{FailOnHttpStatusCodes: []int64{299}},
			},
			noDeadline:    false,
			start:         true,
			expectError:   true,
			expectedError: ErrInvalidHttpStatusCode,
		},
		{
			scenario: "ErrConsoleExceptions",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<script type=\"application/javascript\">throw new Error(\"Exception\")</script>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{FailOnConsoleExceptions: true},
			},
			noDeadline:    false,
			start:         true,
			expectError:   true,
			expectedError: ErrConsoleExceptions,
		},
		{
			scenario: "clear cache",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
					clearCache:       true,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Clear cache</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"clear cache",
			},
		},
		{
			scenario: "clear cookies",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
					clearCookies:     true,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Clear cookies</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"clear cookies",
			},
		},
		{
			scenario: "disable JavaScript",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:           os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout:  5 * time.Second,
					allowList:         regexp2.MustCompile("", 0),
					denyList:          regexp2.MustCompile("", 0),
					disableJavaScript: true,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<script type=\"application/javascript\">throw new Error(\"Exception\")</script>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"disable JavaScript",
				"JavaScript disabled, skipping wait delay",
				"JavaScript disabled, skipping wait expression",
			},
		},
		{
			scenario: "extra HTTP headers",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Extra HTTP headers</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{ExtraHttpHeaders: map[string]string{
					"X-Foo": "Bar",
				}},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"extra HTTP headers:",
			},
		},
		{
			scenario: "ErrOmitBackgroundWithoutPrintBackground",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrOmitBackgroundWithoutPrintBackground</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{OmitBackground: true},
			},
			noDeadline:    false,
			start:         true,
			expectError:   true,
			expectedError: ErrOmitBackgroundWithoutPrintBackground,
		},
		{
			scenario: "hide default white background",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Hide default white background</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options:         Options{OmitBackground: true},
				PrintBackground: true,
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"hide default white background",
			},
		},
		{
			scenario: "ErrInvalidEmulatedMediaType",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrInvalidEmulatedMediaType</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{EmulatedMediaType: "foo"},
			},
			noDeadline:    false,
			start:         true,
			expectError:   true,
			expectedError: ErrInvalidEmulatedMediaType,
		},
		{
			scenario: "emulate a media type",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<style>@media print { #screen { display: none } }</style><p id=\"screen\">Screen media type</p>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{EmulatedMediaType: "screen"},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"emulate media type 'screen'",
			},
		},
		{
			scenario: "wait delay: context done",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<script type=\"application/javascript\">await new Promise(r => setTimeout(r, 10000));</script>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{WaitDelay: time.Duration(10) * time.Second},
			},
			noDeadline:  false,
			start:       true,
			expectError: true,
			expectedLogEntries: []string{
				"wait '10s' before print",
			},
		},
		{
			scenario: "wait delay",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<script type=\"application/javascript\">await new Promise(r => setTimeout(r, 10000));</script>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{WaitDelay: time.Duration(1) * time.Millisecond},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"wait '1ms' before print",
			},
		},
		{
			scenario: "wait for expression: context done",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				html := `
<script type="application/javascript">
    const delay = ms => new Promise(res => setTimeout(res, ms))
    const changeStatus = async (status) => {
        await delay(10000)
        window.status = status
    };
    changeStatus('ready')
</script>
`

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte(html), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{WaitForExpression: "window.status === 'ready'"},
			},
			noDeadline:  false,
			start:       true,
			expectError: true,
			expectedLogEntries: []string{
				"wait until 'window.status === 'ready'' is true before print",
			},
		},
		{
			scenario: "ErrInvalidEvaluationExpression",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrInvalidEvaluationExpression</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{WaitForExpression: "return undefined"},
			},
			noDeadline:  false,
			start:       true,
			expectError: true,
			expectedLogEntries: []string{
				"wait until 'return undefined' is true before print",
			},
		},
		{
			scenario: "wait for expression",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				html := `
<script type="application/javascript">
	var globalVar = 'notReady'

    const delay = ms => new Promise(res => setTimeout(res, ms))
    delay(2000).then(() => {
        window.globalVar = 'ready'
    })
</script>
`

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte(html), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				Options: Options{WaitForExpression: "window.globalVar === 'ready'"},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"wait until 'window.globalVar === 'ready'' is true before print",
			},
		},
		{
			scenario: "single page",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Custom header and footer</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				SinglePage: true,
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"single page PDF",
			},
		},
		{
			scenario: "custom header and footer",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Custom header and footer</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				HeaderTemplate: "<h1>Header</h1>",
				FooterTemplate: "<h1>Footer</h1>",
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"with custom header and/or footer",
			},
		},
		{
			scenario: "ErrInvalidPrinterSettings",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrInvalidPrinterSettings</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				PaperWidth:   0,
				PaperHeight:  0,
				MarginTop:    1000000,
				MarginBottom: 1000000,
				MarginLeft:   1000000,
				MarginRight:  1000000,
			},
			noDeadline:    false,
			start:         true,
			expectError:   true,
			expectedError: ErrInvalidPrinterSettings,
		},
		{
			scenario: "ErrPageRangesSyntaxError",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrPageRangesSyntaxError</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: PdfOptions{
				PageRanges: "foo",
			},
			noDeadline:    false,
			start:         true,
			expectError:   true,
			expectedError: ErrPageRangesSyntaxError,
		},
		{
			scenario: "success (default options)",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Default options</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options:     DefaultPdfOptions(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"cache not cleared",
				"cookies not cleared",
				"JavaScript not disabled",
				"no extra HTTP headers",
				"navigate to",
				"default white background not hidden",
				"no emulated media type",
				"no wait delay",
				"no wait expression",
				"no custom header nor footer",
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			core, recorded := observer.New(zapcore.DebugLevel)
			logger := zap.New(core)

			defer func() {
				err := os.RemoveAll(tc.fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up, but got: %v", err)
				}
			}()

			if tc.start {
				err := tc.browser.Start(logger)
				if err != nil {
					t.Fatalf("setup error: %v", err)
				}

				defer func(b browser, logger *zap.Logger) {
					err = b.Stop(logger)
					if err != nil {
						t.Fatalf("expected no error while cleaning up, but got: %v", err)
					}
				}(tc.browser, logger)
			}

			var (
				ctx    context.Context
				cancel context.CancelFunc
			)

			if tc.noDeadline {
				ctx = context.Background()
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
				defer cancel()
			}

			err := tc.browser.pdf(
				ctx,
				logger,
				fmt.Sprintf("file://%s/index.html", tc.fs.WorkingDirPath()),
				fmt.Sprintf("%s/%s.pdf", tc.fs.WorkingDirPath(), uuid.NewString()),
				tc.options,
			)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v but got: %v", tc.expectedError, err)
			}

			for _, entry := range tc.expectedLogEntries {
				doExist := true
				for _, log := range recorded.All() {
					doExist = strings.Contains(log.Message, entry)
					if doExist {
						break
					}
				}

				if !doExist {
					t.Errorf("expected '%s' to exist as log entry", entry)
				}
			}
		})
	}
}

func TestChromiumBrowser_screenshot(t *testing.T) {
	for _, tc := range []struct {
		scenario           string
		browser            browser
		fs                 *gotenberg.FileSystem
		options            ScreenshotOptions
		noDeadline         bool
		start              bool
		expectError        bool
		expectedError      error
		expectedLogEntries []string
	}{
		{
			scenario: "browser not started",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.isStarted.Store(false)
				return b
			}(),
			fs:          gotenberg.NewFileSystem(),
			noDeadline:  false,
			start:       false,
			expectError: true,
		},
		{
			scenario: "context has not deadline",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.arguments = browserArguments{
					allowList: regexp2.MustCompile("", 0),
					denyList:  regexp2.MustCompile("", 0),
				}
				b.isStarted.Store(true)
				return b
			}(),
			fs:          gotenberg.NewFileSystem(),
			noDeadline:  true,
			start:       false,
			expectError: true,
		},
		{
			scenario: "ErrFiltered: main URL does not match the allowed list",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.arguments = browserArguments{
					allowList: regexp2.MustCompile(`^file:(?!//\/tmp/).*`, 0),
					denyList:  regexp2.MustCompile("", 0),
				}
				b.isStarted.Store(true)
				return b
			}(),
			fs:            gotenberg.NewFileSystem(),
			noDeadline:    false,
			start:         false,
			expectError:   true,
			expectedError: gotenberg.ErrFiltered,
		},
		{
			scenario: "ErrFiltered: main URL does match the denied list",
			browser: func() browser {
				b := new(chromiumBrowser)
				b.arguments = browserArguments{
					allowList: regexp2.MustCompile("", 0),
					denyList:  regexp2.MustCompile("^file:///tmp.*", 0),
				}
				b.isStarted.Store(true)
				return b
			}(),
			fs:            gotenberg.NewFileSystem(),
			noDeadline:    false,
			start:         false,
			expectError:   true,
			expectedError: gotenberg.ErrFiltered,
		},
		{
			scenario: "a request does not match the allowed list",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("^file:///tmp.*", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<iframe src='file:///etc/passwd'></iframe>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"'file:///etc/passwd' does not match the expression from the allowed list",
			},
		},
		{
			scenario: "a request does match the denied list",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile(`^file:(?!//\/tmp/).*`, 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<iframe src='file:///etc/passwd'></iframe>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"'file:///etc/passwd' matches the expression from the denied list",
			},
		},
		{
			scenario: "skip networkIdle event",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Skip networkIdle event</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{SkipNetworkIdleEvent: true},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"skipping network idle event",
			},
		},
		{
			scenario: "ErrInvalidHttpStatusCode",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrInvalidHttpStatusCode</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{FailOnHttpStatusCodes: []int64{299}},
			},
			noDeadline:    false,
			start:         true,
			expectError:   true,
			expectedError: ErrInvalidHttpStatusCode,
		},
		{
			scenario: "ErrConsoleExceptions",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<script type=\"application/javascript\">throw new Error(\"Exception\")</script>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{FailOnConsoleExceptions: true},
			},
			noDeadline:    false,
			start:         true,
			expectError:   true,
			expectedError: ErrConsoleExceptions,
		},
		{
			scenario: "clear cache",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
					clearCache:       true,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Clear cache</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"clear cache",
			},
		},
		{
			scenario: "clear cookies",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
					clearCookies:     true,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Clear cookies</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"clear cookies",
			},
		},
		{
			scenario: "disable JavaScript",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:           os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout:  5 * time.Second,
					allowList:         regexp2.MustCompile("", 0),
					denyList:          regexp2.MustCompile("", 0),
					disableJavaScript: true,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<script type=\"application/javascript\">throw new Error(\"Exception\")</script>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"disable JavaScript",
				"JavaScript disabled, skipping wait delay",
				"JavaScript disabled, skipping wait expression",
			},
		},
		{
			scenario: "extra HTTP headers",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Extra HTTP headers</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{ExtraHttpHeaders: map[string]string{
					"X-Foo": "Bar",
				}},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"extra HTTP headers:",
			},
		},
		{
			scenario: "ErrOmitBackgroundWithoutPrintBackground",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrOmitBackgroundWithoutPrintBackground</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{OmitBackground: true},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"hide default white background",
			},
		},
		{
			scenario: "ErrInvalidEmulatedMediaType",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrInvalidEmulatedMediaType</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{EmulatedMediaType: "foo"},
			},
			noDeadline:    false,
			start:         true,
			expectError:   true,
			expectedError: ErrInvalidEmulatedMediaType,
		},
		{
			scenario: "emulate a media type",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<style>@media print { #screen { display: none } }</style><p id=\"screen\">Screen media type</p>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{EmulatedMediaType: "screen"},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"emulate media type 'screen'",
			},
		},
		{
			scenario: "wait delay: context done",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<script type=\"application/javascript\">await new Promise(r => setTimeout(r, 10000));</script>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{WaitDelay: time.Duration(10) * time.Second},
			},
			noDeadline:  false,
			start:       true,
			expectError: true,
			expectedLogEntries: []string{
				"wait '10s' before print",
			},
		},
		{
			scenario: "wait delay",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<script type=\"application/javascript\">await new Promise(r => setTimeout(r, 10000));</script>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{WaitDelay: time.Duration(1) * time.Millisecond},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"wait '1ms' before print",
			},
		},
		{
			scenario: "wait for expression: context done",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				html := `
<script type="application/javascript">
    const delay = ms => new Promise(res => setTimeout(res, ms))
    const changeStatus = async (status) => {
        await delay(10000)
        window.status = status
    };
    changeStatus('ready')
</script>
`

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte(html), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{WaitForExpression: "window.status === 'ready'"},
			},
			noDeadline:  false,
			start:       true,
			expectError: true,
			expectedLogEntries: []string{
				"wait until 'window.status === 'ready'' is true before print",
			},
		},
		{
			scenario: "ErrInvalidEvaluationExpression",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>ErrInvalidEvaluationExpression</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{WaitForExpression: "return undefined"},
			},
			noDeadline:  false,
			start:       true,
			expectError: true,
			expectedLogEntries: []string{
				"wait until 'return undefined' is true before print",
			},
		},
		{
			scenario: "wait for expression",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				html := `
<script type="application/javascript">
	var globalVar = 'notReady'

    const delay = ms => new Promise(res => setTimeout(res, ms))
    delay(2000).then(() => {
        window.globalVar = 'ready'
    })
</script>
`

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte(html), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: ScreenshotOptions{
				Options: Options{WaitForExpression: "window.globalVar === 'ready'"},
			},
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"wait until 'window.globalVar === 'ready'' is true before print",
			},
		},
		{
			scenario: "success (default options)",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Default options</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options:     DefaultScreenshotOptions(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"cache not cleared",
				"cookies not cleared",
				"JavaScript not disabled",
				"no extra HTTP headers",
				"navigate to",
				"default white background not hidden",
				"no emulated media type",
				"no wait delay",
				"no wait expression",
			},
		},
		{
			scenario: "success (jpeg)",
			browser: newChromiumBrowser(
				browserArguments{
					binPath:          os.Getenv("CHROMIUM_BIN_PATH"),
					wsUrlReadTimeout: 5 * time.Second,
					allowList:        regexp2.MustCompile("", 0),
					denyList:         regexp2.MustCompile("", 0),
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", fs.WorkingDirPath()), []byte("<h1>Default options</h1>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Format = "jpeg"
				return options
			}(),
			noDeadline:  false,
			start:       true,
			expectError: false,
			expectedLogEntries: []string{
				"cache not cleared",
				"cookies not cleared",
				"JavaScript not disabled",
				"no extra HTTP headers",
				"navigate to",
				"default white background not hidden",
				"no emulated media type",
				"no wait delay",
				"no wait expression",
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			core, recorded := observer.New(zapcore.DebugLevel)
			logger := zap.New(core)

			defer func() {
				err := os.RemoveAll(tc.fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up, but got: %v", err)
				}
			}()

			if tc.start {
				err := tc.browser.Start(logger)
				if err != nil {
					t.Fatalf("setup error: %v", err)
				}

				defer func(b browser, logger *zap.Logger) {
					err = b.Stop(logger)
					if err != nil {
						t.Fatalf("expected no error while cleaning up, but got: %v", err)
					}
				}(tc.browser, logger)
			}

			var (
				ctx    context.Context
				cancel context.CancelFunc
			)

			if tc.noDeadline {
				ctx = context.Background()
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
				defer cancel()
			}

			err := tc.browser.screenshot(
				ctx,
				logger,
				fmt.Sprintf("file://%s/index.html", tc.fs.WorkingDirPath()),
				fmt.Sprintf("%s/%s.pdf", tc.fs.WorkingDirPath(), uuid.NewString()),
				tc.options,
			)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v but got: %v", tc.expectedError, err)
			}

			for _, entry := range tc.expectedLogEntries {
				doExist := true
				for _, log := range recorded.All() {
					doExist = strings.Contains(log.Message, entry)
					if doExist {
						break
					}
				}

				if !doExist {
					t.Errorf("expected '%s' to exist as log entry", entry)
				}
			}
		})
	}
}
