package gotenberg

import (
	"context"

	"go.uber.org/zap"
)

// ModuleMock is a mock for the [Module] interface.
type ModuleMock struct {
	DescriptorMock func() ModuleDescriptor
}

func (mod *ModuleMock) Descriptor() ModuleDescriptor {
	return mod.DescriptorMock()
}

// ProvisionerMock is a mock for the [Provisioner] interface.
type ProvisionerMock struct {
	ProvisionMock func(*Context) error
}

func (mod *ProvisionerMock) Provision(ctx *Context) error {
	return mod.ProvisionMock(ctx)
}

// ValidatorMock is a mock for the [Validator] interface.
type ValidatorMock struct {
	ValidateMock func() error
}

func (mod *ValidatorMock) Validate() error {
	return mod.ValidateMock()
}

// PdfEngineMock is a mock for the [PdfEngine] interface.
type PdfEngineMock struct {
	MergeMock   func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error
	ConvertMock func(ctx context.Context, logger *zap.Logger, formats PdfFormats, inputPath, outputPath string) error
}

func (engine *PdfEngineMock) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	return engine.MergeMock(ctx, logger, inputPaths, outputPath)
}

func (engine *PdfEngineMock) Convert(ctx context.Context, logger *zap.Logger, formats PdfFormats, inputPath, outputPath string) error {
	return engine.ConvertMock(ctx, logger, formats, inputPath, outputPath)
}

// PdfEngineProviderMock is a mock for the [PdfEngineProvider] interface.
type PdfEngineProviderMock struct {
	PdfEngineMock func() (PdfEngine, error)
}

func (provider *PdfEngineProviderMock) PdfEngine() (PdfEngine, error) {
	return provider.PdfEngineMock()
}

// ProcessMock is a mock for the [Process] interface.
type ProcessMock struct {
	StartMock   func(logger *zap.Logger) error
	StopMock    func(logger *zap.Logger) error
	HealthyMock func(logger *zap.Logger) bool
}

func (p *ProcessMock) Start(logger *zap.Logger) error {
	return p.StartMock(logger)
}

func (p *ProcessMock) Stop(logger *zap.Logger) error {
	return p.StopMock(logger)
}

func (p *ProcessMock) Healthy(logger *zap.Logger) bool {
	return p.HealthyMock(logger)
}

// ProcessSupervisorMock is a mock for the [ProcessSupervisor] interface.
type ProcessSupervisorMock struct {
	LaunchMock        func() error
	ShutdownMock      func() error
	HealthyMock       func() bool
	RunMock           func(ctx context.Context, logger *zap.Logger, task func() error) error
	ReqQueueSizeMock  func() int64
	RestartsCountMock func() int64
}

func (s *ProcessSupervisorMock) Launch() error {
	return s.LaunchMock()
}

func (s *ProcessSupervisorMock) Shutdown() error {
	return s.ShutdownMock()
}

func (s *ProcessSupervisorMock) Healthy() bool {
	return s.HealthyMock()
}

func (s *ProcessSupervisorMock) Run(ctx context.Context, logger *zap.Logger, task func() error) error {
	return s.RunMock(ctx, logger, task)
}

func (s *ProcessSupervisorMock) ReqQueueSize() int64 {
	return s.ReqQueueSizeMock()
}

func (s *ProcessSupervisorMock) RestartsCount() int64 {
	return s.RestartsCountMock()
}

// LoggerProviderMock is a mock for the [LoggerProvider] interface.
type LoggerProviderMock struct {
	LoggerMock func(mod Module) (*zap.Logger, error)
}

func (provider *LoggerProviderMock) Logger(mod Module) (*zap.Logger, error) {
	return provider.LoggerMock(mod)
}

// MetricsProviderMock is a mock for the [MetricsProvider] interface.
type MetricsProviderMock struct {
	MetricsMock func() ([]Metric, error)
}

func (provider *MetricsProviderMock) Metrics() ([]Metric, error) {
	return provider.MetricsMock()
}

// Interface guards.
var (
	_ Module            = (*ModuleMock)(nil)
	_ Validator         = (*ValidatorMock)(nil)
	_ PdfEngine         = (*PdfEngineMock)(nil)
	_ PdfEngineProvider = (*PdfEngineProviderMock)(nil)
	_ Process           = (*ProcessMock)(nil)
	_ ProcessSupervisor = (*ProcessSupervisorMock)(nil)
	_ LoggerProvider    = (*LoggerProviderMock)(nil)
	_ MetricsProvider   = (*MetricsProviderMock)(nil)
)
