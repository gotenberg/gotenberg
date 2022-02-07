package gotenberg

import (
	"context"

	"go.uber.org/zap"
)

// ModuleMock is a mock for the Module interface.
type ModuleMock struct {
	DescriptorMock func() ModuleDescriptor
}

func (mod ModuleMock) Descriptor() ModuleDescriptor {
	return mod.DescriptorMock()
}

// ValidatorMock is a mock for the Validator interface.
type ValidatorMock struct {
	ValidateMock func() error
}

func (mod ValidatorMock) Validate() error {
	return mod.ValidateMock()
}

// PDFEngineMock is a mock for the PDFEngine interface.
type PDFEngineMock struct {
	MergeMock   func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error
	ConvertMock func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error
}

func (engine PDFEngineMock) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	return engine.MergeMock(ctx, logger, inputPaths, outputPath)
}

func (engine PDFEngineMock) Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
	return engine.ConvertMock(ctx, logger, format, inputPath, outputPath)
}

// PDFEngineProviderMock is a mock for the PDFEngineProvider interface.
type PDFEngineProviderMock struct {
	PDFEngineMock func() (PDFEngine, error)
}

func (provider PDFEngineProviderMock) PDFEngine() (PDFEngine, error) {
	return provider.PDFEngineMock()
}

// LoggerProviderMock is a mock for the LoggerProvider interface.
type LoggerProviderMock struct {
	LoggerMock func(mod Module) (*zap.Logger, error)
}

func (provider LoggerProviderMock) Logger(mod Module) (*zap.Logger, error) {
	return provider.LoggerMock(mod)
}

// Interface guards.
var (
	_ Module            = (*ModuleMock)(nil)
	_ Validator         = (*ValidatorMock)(nil)
	_ PDFEngine         = (*PDFEngineMock)(nil)
	_ PDFEngineProvider = (*PDFEngineProviderMock)(nil)
	_ LoggerProvider    = (*LoggerProviderMock)(nil)
)
