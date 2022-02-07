package uno

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestAPIMock(t *testing.T) {
	mock := APIMock{
		PDFMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error {
			return nil
		},
		ExtensionsMock: func() []string {
			return nil
		},
	}

	err := mock.PDF(context.Background(), zap.NewNop(), "", "", Options{})
	if err != nil {
		t.Errorf("expected no error from mock.PDF(), but got: %v", err)
	}

	ext := mock.Extensions()
	if ext != nil {
		t.Errorf("expected no extensions from mock.Extensions(), but got: %+v", ext)
	}
}

func TestProviderMock(t *testing.T) {
	mock := ProviderMock{
		UNOMock: func() (API, error) {
			return APIMock{}, nil
		},
	}

	_, err := mock.UNO()
	if err != nil {
		t.Errorf("expected no error from mock.UNO(), but got: %v", err)
	}
}
