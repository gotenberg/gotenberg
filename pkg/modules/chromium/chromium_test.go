package chromium

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/alexliesenfeld/health"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestDefaultOptions(t *testing.T) {
	actual := DefaultPdfOptions()
	notExpect := PdfOptions{}

	if reflect.DeepEqual(actual, notExpect) {
		t.Errorf("expected %v and got identical %v", actual, notExpect)
	}
}

func TestChromium_Descriptor(t *testing.T) {
	descriptor := new(Chromium).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Chromium))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestChromium_Provision(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         *gotenberg.Context
		expectError bool
	}{
		{
			scenario: "no logger provider",
			ctx: func() *gotenberg.Context {
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Chromium).Descriptor().FlagSet,
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
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
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
			expectError: true,
		},
		{
			scenario: "no PDF engine provider",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
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
			expectError: true,
		},
		{
			scenario: "no PDF engine from PDF engine provider",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
					gotenberg.PdfEngineProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}
				mod.PdfEngineMock = func() (gotenberg.PdfEngine, error) {
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
			expectError: true,
		},
		{
			scenario: "provision success",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
					gotenberg.PdfEngineProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
				}
				mod.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}
				mod.PdfEngineMock = func() (gotenberg.PdfEngine, error) {
					return new(gotenberg.PdfEngineMock), nil
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
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Chromium)
			err := mod.Provision(tc.ctx)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestChromium_Validate(t *testing.T) {
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
			binPath:     os.Getenv("CHROMIUM_BIN_PATH"),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Chromium)
			mod.args = browserArguments{
				binPath: tc.binPath,
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

func TestChromium_Start(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		autoStart   bool
		supervisor  *gotenberg.ProcessSupervisorMock
		expectError bool
	}{
		{
			scenario:    "no auto-start",
			autoStart:   false,
			expectError: false,
		},
		{
			scenario:  "auto-start success",
			autoStart: true,
			supervisor: &gotenberg.ProcessSupervisorMock{LaunchMock: func() error {
				return nil
			}},
			expectError: false,
		},
		{
			scenario:  "auto-start failed",
			autoStart: true,
			supervisor: &gotenberg.ProcessSupervisorMock{LaunchMock: func() error {
				return errors.New("foo")
			}},
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Chromium)
			mod.autoStart = tc.autoStart
			mod.supervisor = tc.supervisor

			err := mod.Start()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestChromium_StartupMessage(t *testing.T) {
	mod := new(Chromium)

	mod.autoStart = true
	autoStartMsg := mod.StartupMessage()

	mod.autoStart = false
	noAutoStartMsg := mod.StartupMessage()

	if autoStartMsg == noAutoStartMsg {
		t.Errorf("expected differrent startup messages based on auto start, but got '%s'", autoStartMsg)
	}
}

func TestChromium_Stop(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		supervisor  *gotenberg.ProcessSupervisorMock
		expectError bool
	}{
		{
			scenario: "stop success",
			supervisor: &gotenberg.ProcessSupervisorMock{ShutdownMock: func() error {
				return nil
			}},
			expectError: false,
		},
		{
			scenario: "stop failed",
			supervisor: &gotenberg.ProcessSupervisorMock{ShutdownMock: func() error {
				return errors.New("foo")
			}},
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Chromium)
			mod.logger = zap.NewNop()
			mod.supervisor = tc.supervisor

			ctx, cancel := context.WithTimeout(context.Background(), 0*time.Second)
			cancel()

			err := mod.Stop(ctx)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestChromium_Metrics(t *testing.T) {
	mod := new(Chromium)
	mod.supervisor = &gotenberg.ProcessSupervisorMock{
		ReqQueueSizeMock: func() int64 {
			return 10
		},
		RestartsCountMock: func() int64 {
			return 0
		},
	}

	metrics, err := mod.Metrics()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("expected %d metrics, but got %d", 2, len(metrics))
	}

	actual := metrics[0].Read()
	if actual != float64(10) {
		t.Errorf("expected %f for chromium_requests_queue_size, but got %f", float64(10), actual)
	}

	actual = metrics[1].Read()
	if actual != float64(0) {
		t.Errorf("expected %f for chromium_restarts_count, but got %f", float64(0), actual)
	}
}

func TestChromium_Checks(t *testing.T) {
	for _, tc := range []struct {
		scenario                 string
		supervisor               gotenberg.ProcessSupervisor
		expectAvailabilityStatus health.AvailabilityStatus
	}{
		{
			scenario: "healthy module",
			supervisor: &gotenberg.ProcessSupervisorMock{HealthyMock: func() bool {
				return true
			}},
			expectAvailabilityStatus: health.StatusUp,
		},
		{
			scenario: "unhealthy module",
			supervisor: &gotenberg.ProcessSupervisorMock{HealthyMock: func() bool {
				return false
			}},
			expectAvailabilityStatus: health.StatusDown,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Chromium)
			mod.supervisor = tc.supervisor

			checks, err := mod.Checks()
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			checker := health.NewChecker(checks...)
			result := checker.Check(context.Background())

			if result.Status != tc.expectAvailabilityStatus {
				t.Errorf("expected '%s' as availability status, but got '%s'", tc.expectAvailabilityStatus, result.Status)
			}
		})
	}
}

func TestChromium_Ready(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		autoStart    bool
		startTimeout time.Duration
		browser      browser
		expectError  bool
	}{
		{
			scenario:     "no auto-start",
			autoStart:    false,
			startTimeout: time.Duration(30) * time.Second,
			browser: &browserMock{ProcessMock: gotenberg.ProcessMock{HealthyMock: func(logger *zap.Logger) bool {
				return false
			}}},
			expectError: false,
		},
		{
			scenario:     "auto-start: context done",
			autoStart:    true,
			startTimeout: time.Duration(200) * time.Millisecond,
			browser: &browserMock{ProcessMock: gotenberg.ProcessMock{HealthyMock: func(logger *zap.Logger) bool {
				return false
			}}},
			expectError: true,
		},
		{
			scenario:     "auto-start success",
			autoStart:    true,
			startTimeout: time.Duration(30) * time.Second,
			browser: &browserMock{ProcessMock: gotenberg.ProcessMock{HealthyMock: func(logger *zap.Logger) bool {
				return true
			}}},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Chromium)
			mod.autoStart = tc.autoStart
			mod.args = browserArguments{wsUrlReadTimeout: tc.startTimeout}
			mod.browser = tc.browser

			err := mod.Ready()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
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
	for _, tc := range []struct {
		scenario      string
		expectRoutes  int
		disableRoutes bool
	}{
		{
			scenario:      "routes not disabled",
			expectRoutes:  6,
			disableRoutes: false,
		},
		{
			scenario:      "routes disabled",
			expectRoutes:  0,
			disableRoutes: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Chromium)
			mod.disableRoutes = tc.disableRoutes

			routes, err := mod.Routes()
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectRoutes != len(routes) {
				t.Errorf("expected %d routes but got %d", tc.expectRoutes, len(routes))
			}
		})
	}
}

func TestChromium_Pdf(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		supervisor  gotenberg.ProcessSupervisor
		browser     browser
		expectError bool
	}{
		{
			scenario: "PDF task success",
			browser: &browserMock{pdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return nil
			}},
			expectError: false,
		},
		{
			scenario: "PDF task error",
			browser: &browserMock{pdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return errors.New("PDF task error")
			}},
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Chromium)
			mod.supervisor = &gotenberg.ProcessSupervisorMock{RunMock: func(ctx context.Context, logger *zap.Logger, task func() error) error {
				return task()
			}}
			mod.browser = tc.browser

			err := mod.Pdf(context.Background(), zap.NewNop(), "", "", PdfOptions{})

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestChromium_Screenshot(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		supervisor  gotenberg.ProcessSupervisor
		browser     browser
		expectError bool
	}{
		{
			scenario: "Screenshot task success",
			browser: &browserMock{screenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return nil
			}},
			expectError: false,
		},
		{
			scenario: "Screenshot task error",
			browser: &browserMock{screenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return errors.New("screenshot task error")
			}},
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Chromium)
			mod.supervisor = &gotenberg.ProcessSupervisorMock{RunMock: func(ctx context.Context, logger *zap.Logger, task func() error) error {
				return task()
			}}
			mod.browser = tc.browser

			err := mod.Screenshot(context.Background(), zap.NewNop(), "", "", ScreenshotOptions{})

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}
