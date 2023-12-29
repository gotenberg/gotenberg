package chromium

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestApiMock(t *testing.T) {
	mock := &ApiMock{
		PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
			return nil
		},
		ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
			return nil
		},
	}

	err := mock.Pdf(context.Background(), zap.NewNop(), "", "", PdfOptions{})
	if err != nil {
		t.Errorf("expected no error from ApiMock.Pdf, but got: %v", err)
	}

	err = mock.Screenshot(context.Background(), zap.NewNop(), "", "", ScreenshotOptions{})
	if err != nil {
		t.Errorf("expected no error from ApiMock.Screenshot, but got: %v", err)
	}
}

func TestBrowserMock(t *testing.T) {
	mock := &browserMock{
		pdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
			return nil
		},
		screenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
			return nil
		},
	}

	err := mock.pdf(context.Background(), zap.NewNop(), "", "", PdfOptions{})
	if err != nil {
		t.Errorf("expected no error from browserMock.pdf, but got: %v", err)
	}

	err = mock.screenshot(context.Background(), zap.NewNop(), "", "", ScreenshotOptions{})
	if err != nil {
		t.Errorf("expected no error from browserMock.screenshot, but got: %v", err)
	}
}
