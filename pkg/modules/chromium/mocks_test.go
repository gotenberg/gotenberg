package chromium

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestApiMock(t *testing.T) {
	mock := &ApiMock{
		PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
			return nil
		},
	}

	err := mock.Pdf(context.Background(), zap.NewNop(), "", "", Options{})
	if err != nil {
		t.Errorf("expected no error from ApiMock.Pdf, but got: %v", err)
	}
}

func TestBrowserMock(t *testing.T) {
	mock := &browserMock{
		pdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
			return nil
		},
	}

	err := mock.pdf(context.Background(), zap.NewNop(), "", "", Options{})
	if err != nil {
		t.Errorf("expected no error from browserMock.pdf, but got: %v", err)
	}
}
