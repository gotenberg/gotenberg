package api

import (
	"context"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// ApiMock is a mock for the [Uno] interface.
type ApiMock struct {
	PdfMock        func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error
	ExtensionsMock func() []string
}

func (api *ApiMock) Pdf(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error {
	return api.PdfMock(ctx, logger, inputPath, outputPath, options)
}

func (api *ApiMock) Extensions() []string {
	return api.ExtensionsMock()
}

// ProviderMock is a mock for the [Provider] interface.
type ProviderMock struct {
	LibreOfficeMock func() (Uno, error)
}

func (provider *ProviderMock) LibreOffice() (Uno, error) {
	return provider.LibreOfficeMock()
}

// libreOfficeMock is a mock for the [libreOffice] interface.
type libreOfficeMock struct {
	gotenberg.ProcessMock
	pdfMock func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error
}

func (b *libreOfficeMock) pdf(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error {
	return b.pdfMock(ctx, logger, inputPath, outputPath, options)
}

// Interface guards.
var (
	_ Uno         = (*ApiMock)(nil)
	_ Provider    = (*ProviderMock)(nil)
	_ libreOffice = (*libreOfficeMock)(nil)
)
