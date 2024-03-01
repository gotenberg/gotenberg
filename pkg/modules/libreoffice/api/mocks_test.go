package api

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestApiMock(t *testing.T) {
	mock := &ApiMock{
		PdfMock: func(ctx context.Context, logger *zap.Logger, input, outputPath string, options Options) error {
			return nil
		},
		ExtensionsMock: func() []string {
			return nil
		},
	}

	err := mock.Pdf(context.Background(), zap.NewNop(), "", "", Options{})
	if err != nil {
		t.Errorf("expected no error from ApiMock.Pdf, but got: %v", err)
	}

	ext := mock.Extensions()
	if ext != nil {
		t.Errorf("expected nil result from ApiMock.Extensions, but got: %v", ext)
	}
}

func TestProviderMock(t *testing.T) {
	mock := &ProviderMock{
		LibreOfficeMock: func() (Uno, error) {
			return nil, nil
		},
	}

	_, err := mock.LibreOffice()
	if err != nil {
		t.Errorf("expected no error from ProviderMock.LibreOffice, but got: %v", err)
	}
}

func TestLibreOfficeMock(t *testing.T) {
	mock := &libreOfficeMock{
		pdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error {
			return nil
		},
	}

	err := mock.pdf(context.Background(), zap.NewNop(), "", "", Options{})
	if err != nil {
		t.Errorf("expected no error from libreOfficeMock.pdf, but got: %v", err)
	}
}
