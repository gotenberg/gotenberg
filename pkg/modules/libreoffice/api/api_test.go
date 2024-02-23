package api

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
		scenario    string
		ctx         *gotenberg.Context
		expectError bool
	}{
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
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod }}
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
			scenario: "provision success",
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
						FlagSet: new(Api).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			a := new(Api)
			err := a.Provision(tc.ctx)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestApi_Validate(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		binPath     string
		unoBinPath  string
		expectError bool
	}{
		{
			scenario:    "empty LibreOffice bin path",
			binPath:     "",
			unoBinPath:  os.Getenv("UNOCONVERTER_BIN_PATH"),
			expectError: true,
		},
		{
			scenario:    "LibreOffice bin path does not exist",
			binPath:     "/foo",
			unoBinPath:  os.Getenv("UNOCONVERTER_BIN_PATH"),
			expectError: true,
		},
		{
			scenario:    "empty uno bin path",
			binPath:     os.Getenv("CHROMIUM_BIN_PATH"),
			unoBinPath:  "",
			expectError: true,
		},
		{
			scenario:    "uno bin path does not exist",
			binPath:     os.Getenv("CHROMIUM_BIN_PATH"),
			unoBinPath:  "/foo",
			expectError: true,
		},
		{
			scenario:    "validate success",
			binPath:     os.Getenv("CHROMIUM_BIN_PATH"),
			unoBinPath:  os.Getenv("UNOCONVERTER_BIN_PATH"),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			a := new(Api)
			a.args = libreOfficeArguments{
				binPath:    tc.binPath,
				unoBinPath: tc.unoBinPath,
			}
			err := a.Validate()

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
			a := new(Api)
			a.autoStart = tc.autoStart
			a.supervisor = tc.supervisor

			err := a.Start()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestApi_StartupMessage(t *testing.T) {
	a := new(Api)

	a.autoStart = true
	autoStartMsg := a.StartupMessage()

	a.autoStart = false
	noAutoStartMsg := a.StartupMessage()

	if autoStartMsg == noAutoStartMsg {
		t.Errorf("expected differrent startup messages based on auto start, but got '%s'", autoStartMsg)
	}
}

func TestApi_Stop(t *testing.T) {
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
			a := new(Api)
			a.logger = zap.NewNop()
			a.supervisor = tc.supervisor

			ctx, cancel := context.WithTimeout(context.Background(), 0*time.Second)
			cancel()

			err := a.Stop(ctx)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestApi_Metrics(t *testing.T) {
	a := new(Api)
	a.supervisor = &gotenberg.ProcessSupervisorMock{
		ReqQueueSizeMock: func() int64 {
			return 10
		},
		RestartsCountMock: func() int64 {
			return 0
		},
	}

	metrics, err := a.Metrics()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("expected %d metrics, but got %d", 2, len(metrics))
	}

	actual := metrics[0].Read()
	if actual != float64(10) {
		t.Errorf("expected %f for libreoffice_requests_queue_size, but got %f", float64(10), actual)
	}

	actual = metrics[1].Read()
	if actual != float64(0) {
		t.Errorf("expected %f for libreoffice_restarts_count, but got %f", float64(0), actual)
	}
}

func TestApi_Checks(t *testing.T) {
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
			a := new(Api)
			a.supervisor = tc.supervisor

			checks, err := a.Checks()
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
		libreOffice  libreOffice
		expectError  bool
	}{
		{
			scenario:     "no auto-start",
			autoStart:    false,
			startTimeout: time.Duration(30) * time.Second,
			libreOffice: &libreOfficeMock{ProcessMock: gotenberg.ProcessMock{HealthyMock: func(logger *zap.Logger) bool {
				return false
			}}},
			expectError: false,
		},
		{
			scenario:     "auto-start: context done",
			autoStart:    true,
			startTimeout: time.Duration(200) * time.Millisecond,
			libreOffice: &libreOfficeMock{ProcessMock: gotenberg.ProcessMock{HealthyMock: func(logger *zap.Logger) bool {
				return false
			}}},
			expectError: true,
		},
		{
			scenario:     "auto-start success",
			autoStart:    true,
			startTimeout: time.Duration(30) * time.Second,
			libreOffice: &libreOfficeMock{ProcessMock: gotenberg.ProcessMock{HealthyMock: func(logger *zap.Logger) bool {
				return true
			}}},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			a := new(Api)
			a.autoStart = tc.autoStart
			a.args = libreOfficeArguments{startTimeout: tc.startTimeout}
			a.libreOffice = tc.libreOffice

			err := a.Ready()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestApi_LibreOffice(t *testing.T) {
	a := new(Api)

	_, err := a.LibreOffice()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestApi_Pdf(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		supervisor  gotenberg.ProcessSupervisor
		libreOffice libreOffice
		expectError bool
	}{
		{
			scenario: "PDF task success",
			libreOffice: &libreOfficeMock{pdfMock: func(ctx context.Context, logger *zap.Logger, input, outputPath string, options Options) error {
				return nil
			}},
			expectError: false,
		},
		{
			scenario: "PDF task error",
			libreOffice: &libreOfficeMock{pdfMock: func(ctx context.Context, logger *zap.Logger, input, outputPath string, options Options) error {
				return errors.New("PDF task error")
			}},
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			a := new(Api)
			a.supervisor = &gotenberg.ProcessSupervisorMock{RunMock: func(ctx context.Context, logger *zap.Logger, task func() error) error {
				return task()
			}}
			a.libreOffice = tc.libreOffice

			err := a.Pdf(context.Background(), zap.NewNop(), "", "", Options{})

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestApi_Extensions(t *testing.T) {
	a := new(Api)
	extensions := a.Extensions()

	actual := len(extensions)
	expect := 80

	if actual != expect {
		t.Errorf("expected %d extensions, but got %d", expect, actual)
	}
}
