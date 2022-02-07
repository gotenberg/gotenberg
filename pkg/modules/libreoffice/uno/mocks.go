package uno

import (
	"context"

	"go.uber.org/zap"
)

// APIMock is a mock for the API interface.
type APIMock struct {
	PDFMock        func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error
	ExtensionsMock func() []string
}

func (api APIMock) PDF(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error {
	return api.PDFMock(ctx, logger, inputPath, outputPath, options)
}

func (api APIMock) Extensions() []string {
	return api.ExtensionsMock()
}

// ProviderMock is a mock for the Provider interface.
type ProviderMock struct {
	UNOMock func() (API, error)
}

func (provider ProviderMock) UNO() (API, error) {
	return provider.UNOMock()
}

// Interface guards.
var (
	_ API      = (*APIMock)(nil)
	_ Provider = (*ProviderMock)(nil)
)
