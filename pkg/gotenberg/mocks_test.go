package gotenberg

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestModuleMock(t *testing.T) {
	mock := &ModuleMock{
		DescriptorMock: func() ModuleDescriptor {
			return ModuleDescriptor{ID: "foo", New: func() Module {
				return nil
			}}
		},
	}

	if mock.Descriptor().ID != "foo" {
		t.Errorf("expected ID '%s' from ModuleMock.Descriptor, but got '%s'", "foo", mock.Descriptor().ID)
	}
}

func TestProvisionerMock(t *testing.T) {
	mock := &ProvisionerMock{
		ProvisionMock: func(*Context) error {
			return nil
		},
	}

	err := mock.Provision(&Context{})
	if err != nil {
		t.Errorf("expected no error from ProvisionerMock.Provision, but got: %v", err)
	}
}

func TestValidatorMock(t *testing.T) {
	mock := &ValidatorMock{
		ValidateMock: func() error {
			return nil
		},
	}

	err := mock.Validate()
	if err != nil {
		t.Errorf("expected no error from ValidatorMock.Validate, but got: %v", err)
	}
}

func TestPDFEngineMock(t *testing.T) {
	mock := &PdfEngineMock{
		MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
			return nil
		},
		ConvertMock: func(ctx context.Context, logger *zap.Logger, formats PdfFormats, inputPath, outputPath string) error {
			return nil
		},
	}

	err := mock.Merge(context.Background(), zap.NewNop(), nil, "")
	if err != nil {
		t.Errorf("expected no error from PdfEngineMock.Merge, but got: %v", err)
	}

	err = mock.Convert(context.Background(), zap.NewNop(), PdfFormats{}, "", "")
	if err != nil {
		t.Errorf("expected no error from PdfEngineMock.Convert, but got: %v", err)
	}
}

func TestPDFEngineProviderMock(t *testing.T) {
	mock := &PdfEngineProviderMock{
		PdfEngineMock: func() (PdfEngine, error) {
			return new(PdfEngineMock), nil
		},
	}

	_, err := mock.PdfEngine()
	if err != nil {
		t.Errorf("expected no error from PdfEngineProviderMock.PdfEngine, but got: %v", err)
	}
}

func TestProcessMock(t *testing.T) {
	mock := &ProcessMock{
		StartMock: func(logger *zap.Logger) error {
			return nil
		},
		StopMock: func(logger *zap.Logger) error {
			return nil
		},
		HealthyMock: func(logger *zap.Logger) bool {
			return true
		},
	}

	err := mock.Start(zap.NewNop())
	if err != nil {
		t.Errorf("expected no error from ProcessMock.Start, but got: %v", err)
	}

	err = mock.Stop(zap.NewNop())
	if err != nil {
		t.Errorf("expected no error from ProcessMock.Stop, but got: %v", err)
	}

	healthy := mock.Healthy(zap.NewNop())
	if !healthy {
		t.Error("expected true from ProcessMock.Healthy, but got false")
	}
}

func TestProcessSupervisorMock(t *testing.T) {
	mock := &ProcessSupervisorMock{
		LaunchMock: func() error {
			return nil
		},
		ShutdownMock: func() error {
			return nil
		},
		HealthyMock: func() bool {
			return true
		},
		RunMock: func(ctx context.Context, logger *zap.Logger, task func() error) error {
			return nil
		},
		ReqQueueSizeMock: func() int64 {
			return 0
		},
		RestartsCountMock: func() int64 {
			return 0
		},
	}

	err := mock.Launch()
	if err != nil {
		t.Errorf("expected no error from ProcessSupervisorMock.Launch, but got: %v", err)
	}

	err = mock.Shutdown()
	if err != nil {
		t.Errorf("expected no error from ProcessSupervisorMock.Shutdown, but got: %v", err)
	}

	healthy := mock.Healthy()
	if !healthy {
		t.Error("expected true from ProcessSupervisorMock.Healthy, but got false")
	}

	err = mock.Run(context.TODO(), zap.NewNop(), nil)
	if err != nil {
		t.Errorf("expected no error from ProcessSupervisorMock.Run, but got: %v", err)
	}

	size := mock.ReqQueueSize()
	if size != 0 {
		t.Errorf("expected 0 from ProcessSupervisorMock.ReqQueueSize, but got: %d", size)
	}

	restarts := mock.RestartsCount()
	if restarts != 0 {
		t.Errorf("expected 0 from ProcessSupervisorMock.RestartsCount, but got: %d", restarts)
	}
}

func TestLoggerProviderMock(t *testing.T) {
	mock := &LoggerProviderMock{
		LoggerMock: func(mod Module) (*zap.Logger, error) {
			return nil, nil
		},
	}

	_, err := mock.Logger(new(ModuleMock))
	if err != nil {
		t.Errorf("expected no error from LoggerProviderMock.Logger, but got: %v", err)
	}
}

func TestMetricsProviderMock(t *testing.T) {
	mock := &MetricsProviderMock{
		MetricsMock: func() ([]Metric, error) {
			return nil, nil
		},
	}

	_, err := mock.Metrics()
	if err != nil {
		t.Errorf("expected no error from MetricsProviderMock.Metrics, but got: %v", err)
	}
}
