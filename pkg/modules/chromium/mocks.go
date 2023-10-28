package chromium

import (
	"context"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
)

// ApiMock is a mock for the [Api] interface.
type ApiMock struct {
	PdfMock func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error
}

func (api *ApiMock) Pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
	return api.PdfMock(ctx, logger, url, outputPath, options)
}

// browserMock is a mock for the [browser] interface.
type browserMock struct {
	gotenberg.ProcessMock
	pdfMock func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error
}

func (b *browserMock) pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
	return b.pdfMock(ctx, logger, url, outputPath, options)
}

// Interface guards.
var (
	_ Api     = (*ApiMock)(nil)
	_ browser = (*browserMock)(nil)
)
