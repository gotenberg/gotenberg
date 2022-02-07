package gotenberg

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestModuleMock(t *testing.T) {
	mock := ModuleMock{
		DescriptorMock: func() ModuleDescriptor {
			return ModuleDescriptor{ID: "foo", New: func() Module {
				return nil
			}}
		},
	}

	if mock.Descriptor().ID != "foo" {
		t.Errorf("expected ID '%s' from mock.Descriptor(), but got '%s'", "foo", mock.Descriptor().ID)
	}
}

func TestValidatorMock(t *testing.T) {
	mock := ValidatorMock{
		ValidateMock: func() error {
			return nil
		},
	}

	err := mock.Validate()
	if err != nil {
		t.Errorf("expected no error from mock.Validate(), but got: %v", err)
	}
}

func TestPDFEngineMock(t *testing.T) {
	mock := PDFEngineMock{
		MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
			return nil
		},
		ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
			return nil
		},
	}

	err := mock.Merge(context.Background(), zap.NewNop(), nil, "")
	if err != nil {
		t.Errorf("expected no error from mock.Merge(), but got: %v", err)
	}

	err = mock.Convert(context.Background(), zap.NewNop(), "", "", "")
	if err != nil {
		t.Errorf("expected no error from mock.Convert(), but got: %v", err)
	}
}

func TestPDFEngineProvider(t *testing.T) {
	mock := PDFEngineProviderMock{
		PDFEngineMock: func() (PDFEngine, error) {
			return PDFEngineMock{}, nil
		},
	}

	_, err := mock.PDFEngine()
	if err != nil {
		t.Errorf("expected no error from mock.PDFEngine(), but got: %v", err)
	}
}

func TestLoggerProviderMock(t *testing.T) {
	mock := LoggerProviderMock{
		LoggerMock: func(mod Module) (*zap.Logger, error) {
			return nil, nil
		},
	}

	_, err := mock.Logger(ModuleMock{})
	if err != nil {
		t.Errorf("expected no error from mock.Logger(), but got: %v", err)
	}
}
