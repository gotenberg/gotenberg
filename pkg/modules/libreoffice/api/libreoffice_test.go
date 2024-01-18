package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestLibreOfficeProcess_Start(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		libreOffice libreOffice
		expectError bool
		cleanup     bool
	}{
		{
			scenario: "successful start",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			expectError: false,
			cleanup:     true,
		},
		{
			scenario: "LibreOffice already started",
			libreOffice: func() libreOffice {
				p := new(libreOfficeProcess)
				p.isStarted.Store(true)
				return p
			}(),
			expectError: true,
			cleanup:     false,
		},
		{
			scenario: "non-exit code 81 on first start",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      "foo",
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			expectError: true,
			cleanup:     false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := zap.NewNop()
			err := tc.libreOffice.Start(logger)

			if tc.cleanup {
				defer func(p libreOffice, logger *zap.Logger) {
					err = p.Stop(logger)
					if err != nil {
						t.Fatalf("expected no error while cleaning up, but got: %v", err)
					}
				}(tc.libreOffice, logger)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestLibreOfficeProcess_Stop(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		libreOffice libreOffice
		setup       func(libreOffice libreOffice, logger *zap.Logger) error
		expectError bool
	}{
		{
			scenario: "successful stop",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			setup: func(p libreOffice, logger *zap.Logger) error {
				return p.Start(logger)
			},
			expectError: false,
		},
		{
			scenario: "LibreOffice already stopped",
			libreOffice: func() libreOffice {
				p := new(libreOfficeProcess)
				p.isStarted.Store(false)
				return p
			}(),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := zap.NewNop()

			if tc.setup != nil {
				err := tc.setup(tc.libreOffice, logger)
				if err != nil {
					t.Fatalf("setup error: %v", err)
				}
			}

			err := tc.libreOffice.Stop(logger)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestLibreOfficeProcess_Healthy(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		libreOffice   libreOffice
		setup         func(libreOffice libreOffice, logger *zap.Logger) error
		expectHealthy bool
		cleanup       bool
	}{
		{
			scenario: "healthy LibreOffice",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			setup: func(p libreOffice, logger *zap.Logger) error {
				return p.Start(logger)
			},
			expectHealthy: true,
			cleanup:       true,
		},
		{
			scenario: "LibreOffice not started",
			libreOffice: func() libreOffice {
				p := new(libreOfficeProcess)
				p.isStarted.Store(false)
				return p
			}(),
			expectHealthy: false,
			cleanup:       false,
		},
		{
			scenario: "unhealthy LibreOffice",
			libreOffice: func() libreOffice {
				p := new(libreOfficeProcess)
				p.isStarted.Store(true)
				p.socketPort = 12345
				return p
			}(),
			expectHealthy: false,
			cleanup:       false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := zap.NewNop()

			if tc.setup != nil {
				err := tc.setup(tc.libreOffice, logger)
				if err != nil {
					t.Fatalf("setup error: %v", err)
				}
			}

			if tc.cleanup {
				defer func(p libreOffice, logger *zap.Logger) {
					err := p.Stop(logger)
					if err != nil {
						t.Fatalf("expected no error while cleaning up, but got: %v", err)
					}
				}(tc.libreOffice, logger)
			}

			healthy := tc.libreOffice.Healthy(logger)

			if !tc.expectHealthy && healthy {
				t.Fatal("expected unhealthy LibreOffice but got an healthy one")
			}

			if tc.expectHealthy && !healthy {
				t.Fatal("expected a healthy LibreOffice but got an unhealthy one")
			}
		})
	}
}

func TestLibreOfficeProcess_pdf(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		libreOffice   libreOffice
		fs            *gotenberg.FileSystem
		options       Options
		cancelledCtx  bool
		start         bool
		expectError   bool
		expectedError error
	}{
		{
			scenario: "LibreOffice not started",
			libreOffice: func() libreOffice {
				p := new(libreOfficeProcess)
				p.isStarted.Store(false)
				return p
			}(),
			fs:           gotenberg.NewFileSystem(),
			cancelledCtx: false,
			start:        false,
			expectError:  true,
		},
		{
			scenario: "ErrInvalidPdfFormats",
			libreOffice: func() libreOffice {
				p := new(libreOfficeProcess)
				p.socketPort = 12345
				p.isStarted.Store(true)
				return p
			}(),
			fs:            gotenberg.NewFileSystem(),
			options:       Options{PdfFormats: gotenberg.PdfFormats{PdfA: "foo"}},
			cancelledCtx:  false,
			start:         false,
			expectError:   true,
			expectedError: ErrInvalidPdfFormats,
		},
		{
			scenario: "ErrMalformedPageRanges",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			options: Options{PageRanges: "foo"},
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("ErrMalformedPageRanges"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			cancelledCtx:  false,
			start:         true,
			expectError:   true,
			expectedError: ErrMalformedPageRanges,
		},
		{
			scenario: "context done",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("Context done"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			cancelledCtx: true,
			start:        true,
			expectError:  true,
		},
		{
			scenario: "success (default options)",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("Success"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			cancelledCtx: false,
			start:        true,
			expectError:  false,
		},
		{
			scenario: "success (landscape)",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("Landscape"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options:      Options{Landscape: true},
			cancelledCtx: false,
			start:        true,
			expectError:  false,
		},
		{
			scenario: "success (page ranges)",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("Landscape"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options:      Options{PageRanges: "1-1"},
			cancelledCtx: false,
			start:        true,
			expectError:  false,
		},
		{
			scenario: "success (PDF/A-1b)",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("Landscape"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options:      Options{PdfFormats: gotenberg.PdfFormats{PdfA: gotenberg.PdfA1b}},
			cancelledCtx: false,
			start:        true,
			expectError:  false,
		},
		{
			scenario: "success (PDF/A-2b)",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("Landscape"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options:      Options{PdfFormats: gotenberg.PdfFormats{PdfA: gotenberg.PdfA2b}},
			cancelledCtx: false,
			start:        true,
			expectError:  false,
		},
		{
			scenario: "success (PDF/A-3b)",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("Landscape"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options:      Options{PdfFormats: gotenberg.PdfFormats{PdfA: gotenberg.PdfA3b}},
			cancelledCtx: false,
			start:        true,
			expectError:  false,
		},
		{
			scenario: "success (PDF/UA)",
			libreOffice: newLibreOfficeProcess(
				libreOfficeArguments{
					binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
					unoBinPath:   os.Getenv("UNOCONVERTER_BIN_PATH"),
					startTimeout: 5 * time.Second,
				},
			),
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("Landscape"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			options:      Options{PdfFormats: gotenberg.PdfFormats{PdfUa: true}},
			cancelledCtx: false,
			start:        true,
			expectError:  false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			// Force the debug level.
			logger := zap.NewExample()

			defer func() {
				err := os.RemoveAll(tc.fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up, but got: %v", err)
				}
			}()

			if tc.start {
				err := tc.libreOffice.Start(logger)
				if err != nil {
					t.Fatalf("setup error: %v", err)
				}

				defer func(p libreOffice, logger *zap.Logger) {
					err = p.Stop(logger)
					if err != nil {
						t.Fatalf("expected no error while cleaning up, but got: %v", err)
					}
				}(tc.libreOffice, logger)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
			defer cancel()

			if tc.cancelledCtx {
				cancel()
			}

			err := tc.libreOffice.pdf(
				ctx,
				logger,
				fmt.Sprintf("%s/document.txt", tc.fs.WorkingDirPath()),
				fmt.Sprintf("%s/%s.pdf", tc.fs.WorkingDirPath(), uuid.NewString()),
				tc.options,
			)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v but got: %v", tc.expectedError, err)
			}
		})
	}
}

func TestNonBasicLatinCharactersGuard(t *testing.T) {
	for _, tc := range []struct {
		scenario            string
		fs                  *gotenberg.FileSystem
		filename            string
		expectSameInputPath bool
		expectError         bool
	}{
		{
			scenario: "basic latin characters",
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/document.txt", fs.WorkingDirPath()), []byte("Basic latin characters"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			filename:            "document.txt",
			expectSameInputPath: true,
			expectError:         false,
		},
		{
			scenario: "non-basic latin characters",
			fs: func() *gotenberg.FileSystem {
				fs := gotenberg.NewFileSystem()

				err := os.MkdirAll(fs.WorkingDirPath(), 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/éèßàùä.txt", fs.WorkingDirPath()), []byte("Non-basic latin characters"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return fs
			}(),
			filename:            "éèßàùä.txt",
			expectSameInputPath: false,
			expectError:         false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			defer func() {
				err := os.RemoveAll(tc.fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up, but got: %v", err)
				}
			}()

			inputPath := fmt.Sprintf("%s/%s", tc.fs.WorkingDirPath(), tc.filename)
			newInputPath, err := nonBasicLatinCharactersGuard(
				zap.NewNop(),
				inputPath,
			)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectSameInputPath && newInputPath != inputPath {
				t.Fatalf("expected same input path, but got '%s'", newInputPath)
			}

			if !tc.expectSameInputPath && newInputPath == inputPath {
				t.Fatalf("expected different input path, but got same '%s'", newInputPath)
			}
		})
	}
}
