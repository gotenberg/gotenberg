package chromium

import (
	"context"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// ApiMock is a mock for the [Api] interface.
type ApiMock struct {
	PdfMock        func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error
	ScreenshotMock func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error
}

func (api *ApiMock) Pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
	return api.PdfMock(ctx, logger, url, outputPath, options)
}

func (api *ApiMock) Screenshot(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
	return api.ScreenshotMock(ctx, logger, url, outputPath, options)
}

// browserMock is a mock for the [browser] interface.
type browserMock struct {
	gotenberg.ProcessMock
	pdfMock        func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error
	screenshotMock func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error
}

func (b *browserMock) pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
	return b.pdfMock(ctx, logger, url, outputPath, options)
}

func (b *browserMock) screenshot(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
	return b.screenshotMock(ctx, logger, url, outputPath, options)
}

// Interface guards.
var (
	_ Api     = (*ApiMock)(nil)
	_ browser = (*browserMock)(nil)
)
