package pdfengines

import (
	"context"
	"errors"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

func TestMultiPDFEngines_Merge(t *testing.T) {
	tests := []struct {
		name           string
		engine         *multiPDFEngines
		ctx            context.Context
		expectMergeErr bool
	}{
		{
			name: "nominal behavior",
			engine: newMultiPDFEngines(
				gotenberg.PDFEngineMock{
					MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
						return nil
					},
				},
			),
			ctx: context.Background(),
		},
		{
			name: "at least one engine does not return an error",
			engine: newMultiPDFEngines(
				gotenberg.PDFEngineMock{
					MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
						return errors.New("foo")
					},
				},
				gotenberg.PDFEngineMock{
					MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
						return nil
					},
				},
			),
			ctx: context.Background(),
		},
		{
			name: "all engines return an error",
			engine: newMultiPDFEngines(
				gotenberg.PDFEngineMock{
					MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
						return errors.New("foo")
					},
				},
				gotenberg.PDFEngineMock{
					MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
						return errors.New("foo")
					},
				},
			),
			ctx:            context.Background(),
			expectMergeErr: true,
		},
		{
			name: "context expired",
			engine: newMultiPDFEngines(
				gotenberg.PDFEngineMock{
					MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
						return nil
					},
				},
			),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			}(),
			expectMergeErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.engine.Merge(tc.ctx, zap.NewNop(), nil, "")

			if tc.expectMergeErr && err == nil {
				t.Errorf("expected engine.Merge() error, but got none")
			}

			if !tc.expectMergeErr && err != nil {
				t.Errorf("expected no error from engine.Merge(), but got: %v", err)
			}
		})
	}
}

func TestMultiPDFEngines_Convert(t *testing.T) {
	tests := []struct {
		name             string
		engine           *multiPDFEngines
		ctx              context.Context
		expectConvertErr bool
	}{
		{
			name: "nominal behavior",
			engine: newMultiPDFEngines(
				gotenberg.PDFEngineMock{
					ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
						return nil
					},
				},
			),
			ctx: context.Background(),
		},
		{
			name: "at least one engine does not return an error",
			engine: newMultiPDFEngines(
				gotenberg.PDFEngineMock{
					ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
						return errors.New("foo")
					},
				},
				gotenberg.PDFEngineMock{
					ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
						return nil
					},
				},
			),
			ctx: context.Background(),
		},
		{
			name: "all engines return an error",
			engine: newMultiPDFEngines(
				gotenberg.PDFEngineMock{
					ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
						return errors.New("foo")
					},
				},
				gotenberg.PDFEngineMock{
					ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
						return errors.New("foo")
					},
				},
			),
			ctx:              context.Background(),
			expectConvertErr: true,
		},
		{
			name: "context expired",
			engine: newMultiPDFEngines(
				gotenberg.PDFEngineMock{
					ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
						return nil
					},
				},
			),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			}(),
			expectConvertErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.engine.Convert(tc.ctx, zap.NewNop(), "", "", "")

			if tc.expectConvertErr && err == nil {
				t.Errorf("expected engine.Convert() error, but got none")
			}

			if !tc.expectConvertErr && err != nil {
				t.Errorf("expected no error from engine.Convert(), but got: %v", err)
			}
		})
	}
}
