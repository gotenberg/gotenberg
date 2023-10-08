package uno

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/alexliesenfeld/health"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
)

func TestUNO_Descriptor(t *testing.T) {
	descriptor := UNO{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(UNO))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestUNO_Provision(t *testing.T) {
	tests := []struct {
		name               string
		ctx                *gotenberg.Context
		expectProvisionErr bool
	}{
		{
			name: "nominal behavior",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(UNO).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
		},
		{
			name: "threshold from deprecated flag --unoconv-disable-listener",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return zap.NewNop(), nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: func() *flag.FlagSet {
							fs := new(UNO).Descriptor().FlagSet
							err := fs.Parse([]string{"--unoconv-disable-listener=true"})
							if err != nil {
								t.Fatalf("expected no error from fs.Parse(), but got: %v", err)
							}

							return fs
						}(),
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
		},
		{
			name: "no logger provider",
			ctx: func() *gotenberg.Context {
				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(UNO).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{},
				)
			}(),
			expectProvisionErr: true,
		},
		{
			name: "no logger from logger provider",
			ctx: func() *gotenberg.Context {
				provider := struct {
					gotenberg.ModuleMock
					gotenberg.LoggerProviderMock
				}{}
				provider.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module {
						return provider
					}}
				}
				provider.LoggerMock = func(mod gotenberg.Module) (*zap.Logger, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(UNO).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						provider.Descriptor(),
					},
				)
			}(),
			expectProvisionErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mod := new(UNO)
			err := mod.Provision(tc.ctx)

			if tc.expectProvisionErr && err == nil {
				t.Errorf("expected mod.Provision() error, but got none")
			}

			if !tc.expectProvisionErr && err != nil {
				t.Errorf("expected no error from mod.Provision(), but got: %v", err)
			}
		})
	}
}

func TestUNO_Validate(t *testing.T) {
	tests := []struct {
		name               string
		unoconvBinPath     string
		libreOfficeBinPath string
		expectValidateErr  bool
	}{
		{
			name:               "nominal behavior",
			unoconvBinPath:     os.Getenv("UNOCONV_BIN_PATH"),
			libreOfficeBinPath: os.Getenv("LIBREOFFICE_BIN_PATH"),
		},
		{
			name:               "unoconv bin path does not exist",
			unoconvBinPath:     "/foo",
			libreOfficeBinPath: os.Getenv("LIBREOFFICE_BIN_PATH"),
			expectValidateErr:  true,
		},
		{
			name:               "LibreOffice bin path does not exist",
			unoconvBinPath:     os.Getenv("UNOCONV_BIN_PATH"),
			libreOfficeBinPath: "/foo",
			expectValidateErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mod := UNO{
				unoconvBinPath:     tc.unoconvBinPath,
				libreOfficeBinPath: tc.libreOfficeBinPath,
			}

			err := mod.Validate()

			if tc.expectValidateErr && err == nil {
				t.Errorf("expected mod.Validate() error, but got none")
			}

			if !tc.expectValidateErr && err != nil {
				t.Errorf("expected no error from mod.Validate(), but got: %v", err)
			}
		})
	}
}

func TestUNO_Start(t *testing.T) {
	mod := new(UNO)

	err := mod.Start()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestUNO_StartupMessage(t *testing.T) {
	actual := new(UNO).StartupMessage()
	expect := "long-running LibreOffice listener ready to start"

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestUNO_Stop(t *testing.T) {
	tests := []struct {
		name          string
		mod           UNO
		expectStopErr bool
	}{
		{
			name: "nominal behavior",
			mod: UNO{
				libreOfficeRestartThreshold: 10,
				listener: listenerMock{
					stopMock: func(logger *zap.Logger) error {
						return nil
					},
				},
				logger: zap.NewNop(),
			},
		},
		{
			name: "stop error",
			mod: UNO{
				libreOfficeRestartThreshold: 10,
				listener: listenerMock{
					stopMock: func(logger *zap.Logger) error {
						return errors.New("foo")
					},
				},
				logger: zap.NewNop(),
			},
			expectStopErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
			cancel()

			err := tc.mod.Stop(ctx)

			if tc.expectStopErr && err == nil {
				t.Errorf("expected mod.Stop() error, but got none")
			}

			if !tc.expectStopErr && err != nil {
				t.Errorf("expected no error from mod.Stop(), but got: %v", err)
			}
		})
	}
}

func TestUNO_Metrics(t *testing.T) {
	tests := []struct {
		name                                          string
		mod                                           UNO
		expectUnoconvActiveInstancesCount             float64
		expectLibreOfficeListenerActiveInstancesCount float64
		expectLibreOfficeListenerQueueLength          float64
	}{
		{
			name: "with healthy long-running LibreOffice listener",
			mod: UNO{
				libreOfficeRestartThreshold: 10,
				listener: listenerMock{
					queueMock: func() int {
						return 0
					},
					healthyMock: func() bool {
						return true
					},
				},
			},
			expectLibreOfficeListenerActiveInstancesCount: 1,
		},
		{
			name: "with unhealthy long-running LibreOffice listener",
			mod: UNO{
				libreOfficeRestartThreshold: 10,
				listener: listenerMock{
					queueMock: func() int {
						return 0
					},
					healthyMock: func() bool {
						return false
					},
				},
			},
		},
		{
			name: "with no long-running LibreOffice listener",
			mod: UNO{
				libreOfficeRestartThreshold: 0,
				listener: listenerMock{
					queueMock: func() int {
						return 0
					},
					healthyMock: func() bool {
						return false
					},
				},
			},
		},
		{
			name: "with a queue of 3",
			mod: UNO{
				libreOfficeRestartThreshold: 0,
				listener: listenerMock{
					queueMock: func() int {
						return 3
					},
					healthyMock: func() bool {
						return true
					},
				},
			},
			expectLibreOfficeListenerQueueLength: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metrics, err := tc.mod.Metrics()
			if err != nil {
				t.Fatalf("expected no error from mod.Metrics(), but got: %v", err)
			}

			for _, metric := range metrics {
				switch metric.Name {
				case "unoconv_active_instances_count":
					actual := metric.Read()
					if actual != tc.expectUnoconvActiveInstancesCount {
						t.Errorf("expected 'unoconv_active_instances_count' to be %.0f, but got %.0f", tc.expectUnoconvActiveInstancesCount, actual)
					}
				case "libreoffice_listener_active_instances_count":
					actual := metric.Read()
					if actual != tc.expectLibreOfficeListenerActiveInstancesCount {
						t.Errorf("expected 'libreoffice_listener_active_instances_count' to be %.0f, but got %.0f", tc.expectLibreOfficeListenerActiveInstancesCount, actual)
					}
				case "unoconv_listener_active_instances_count":
					actual := metric.Read()
					if actual != tc.expectLibreOfficeListenerActiveInstancesCount {
						t.Errorf("expected 'unoconv_listener_active_instances_count' to be %.0f, but got %.0f", tc.expectLibreOfficeListenerActiveInstancesCount, actual)
					}
				case "libreoffice_listener_queue_length":
					actual := metric.Read()
					if actual != tc.expectLibreOfficeListenerQueueLength {
						t.Errorf("expected 'libreoffice_listener_queue_length' to be %.0f, but got %.0f", tc.expectLibreOfficeListenerQueueLength, actual)
					}
				case "unoconv_listener_queue_length":
					actual := metric.Read()
					if actual != tc.expectLibreOfficeListenerQueueLength {
						t.Errorf("expected 'unoconv_listener_queue_length' to be %.0f, but got %.0f", tc.expectLibreOfficeListenerQueueLength, actual)
					}
				}
			}
		})
	}
}

func TestUNO_Checks(t *testing.T) {
	tests := []struct {
		name                     string
		mod                      UNO
		expectAvailabilityStatus health.AvailabilityStatus
	}{
		{
			name: "no long-running LibreOffice listener",
			mod: UNO{
				libreOfficeRestartThreshold: 0,
			},
		},
		{
			name: "with healthy long-running LibreOffice listener",
			mod: UNO{
				libreOfficeRestartThreshold: 10,
				listener: listenerMock{
					healthyMock: func() bool {
						return true
					},
				},
			},
			expectAvailabilityStatus: health.StatusUp,
		},
		{
			name: "with unhealthy long-running LibreOffice listener",
			mod: UNO{
				libreOfficeRestartThreshold: 10,
				listener: listenerMock{
					healthyMock: func() bool {
						return false
					},
				},
			},
			expectAvailabilityStatus: health.StatusDown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checks, err := tc.mod.Checks()
			if err != nil {
				t.Fatalf("expected no error from mod.Checks(), but got: %v", err)
			}

			if len(checks) == 0 {
				return
			}

			if len(checks) != 1 {
				t.Fatalf("expected 1 check from mod.Checks(), but got %d", len(checks))
			}

			checker := health.NewChecker(checks...)
			result := checker.Check(context.Background())

			if result.Status != tc.expectAvailabilityStatus {
				t.Errorf("expected '%s' as availability status, but got '%s'", tc.expectAvailabilityStatus, result.Status)
			}
		})
	}
}

func TestUNO_PDF(t *testing.T) {
	tests := []struct {
		scenario     string
		mod          UNO
		ctx          context.Context
		logger       *zap.Logger
		inputPath    string
		options      Options
		expectPDFErr bool
		teardown     func(mod UNO) error
	}{
		{
			scenario: "nominal behavior with a long-running LibreOffice listener",
			mod: func() UNO {
				mod := UNO{
					unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
					libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
					libreOfficeStartTimeout:     time.Duration(10) * time.Second,
					libreOfficeRestartThreshold: 10,
					logger:                      zap.NewNop(),
				}
				mod.listener = newLibreOfficeListener(
					mod.logger,
					gotenberg.NewFileSystem(),
					mod.libreOfficeBinPath,
					mod.libreOfficeStartTimeout,
					mod.libreOfficeRestartThreshold,
				)

				return mod
			}(),
			ctx:       context.Background(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			teardown: func(mod UNO) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return mod.Stop(ctx)
			},
		},
		//{
		//	scenario: "convert with a debug logger",
		//	mod: func() UNO {
		//		mod := UNO{
		//			unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
		//			libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
		//			libreOfficeStartTimeout:     time.Duration(10) * time.Second,
		//			libreOfficeRestartThreshold: 10,
		//			logger:                      zap.NewNop(),
		//		}
		//		mod.listener = newLibreOfficeListener(
		//			mod.logger,
		//			gotenberg.NewFileSystem(),
		//			mod.libreOfficeBinPath,
		//			mod.libreOfficeStartTimeout,
		//			mod.libreOfficeRestartThreshold,
		//		)
		//
		//		return mod
		//	}(),
		//	ctx:       context.Background(),
		//	logger:    zap.NewExample(),
		//	inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
		//	teardown: func(mod UNO) error {
		//		ctx, cancel := context.WithCancel(context.Background())
		//		cancel()
		//
		//		return mod.Stop(ctx)
		//	},
		//},
		{
			scenario: "convert with landscape",
			mod: func() UNO {
				mod := UNO{
					unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
					libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
					libreOfficeStartTimeout:     time.Duration(10) * time.Second,
					libreOfficeRestartThreshold: 10,
					logger:                      zap.NewNop(),
				}
				mod.listener = newLibreOfficeListener(
					mod.logger,
					gotenberg.NewFileSystem(),
					mod.libreOfficeBinPath,
					mod.libreOfficeStartTimeout,
					mod.libreOfficeRestartThreshold,
				)

				return mod
			}(),
			ctx:       context.Background(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				Landscape: true,
			},
			teardown: func(mod UNO) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return mod.Stop(ctx)
			},
		},
		{
			scenario: "convert with page ranges",
			mod: func() UNO {
				mod := UNO{
					unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
					libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
					libreOfficeStartTimeout:     time.Duration(10) * time.Second,
					libreOfficeRestartThreshold: 10,
					logger:                      zap.NewNop(),
				}
				mod.listener = newLibreOfficeListener(
					mod.logger,
					gotenberg.NewFileSystem(),
					mod.libreOfficeBinPath,
					mod.libreOfficeStartTimeout,
					mod.libreOfficeRestartThreshold,
				)

				return mod
			}(),
			ctx:       context.Background(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				PageRanges: "1-2",
			},
			teardown: func(mod UNO) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return mod.Stop(ctx)
			},
		},
		{
			scenario: "convert with invalid page ranges",
			mod: func() UNO {
				mod := UNO{
					unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
					libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
					libreOfficeStartTimeout:     time.Duration(10) * time.Second,
					libreOfficeRestartThreshold: 10,
					logger:                      zap.NewNop(),
				}
				mod.listener = newLibreOfficeListener(
					mod.logger,
					gotenberg.NewFileSystem(),
					mod.libreOfficeBinPath,
					mod.libreOfficeStartTimeout,
					mod.libreOfficeRestartThreshold,
				)

				return mod
			}(),
			ctx:       context.Background(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				PageRanges: "foo",
			},
			expectPDFErr: true,
			teardown: func(mod UNO) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return mod.Stop(ctx)
			},
		},
		{
			scenario: "convert to PDF/A-1a",
			mod: func() UNO {
				mod := UNO{
					unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
					libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
					libreOfficeStartTimeout:     time.Duration(10) * time.Second,
					libreOfficeRestartThreshold: 10,
					logger:                      zap.NewNop(),
				}
				mod.listener = newLibreOfficeListener(
					mod.logger,
					gotenberg.NewFileSystem(),
					mod.libreOfficeBinPath,
					mod.libreOfficeStartTimeout,
					mod.libreOfficeRestartThreshold,
				)

				return mod
			}(),
			ctx:       context.Background(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				PDFformat: gotenberg.FormatPDFA1a,
			},
			teardown: func(mod UNO) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return mod.Stop(ctx)
			},
		},
		{
			scenario: "convert to PDF/A-2b",
			mod: func() UNO {
				mod := UNO{
					unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
					libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
					libreOfficeStartTimeout:     time.Duration(10) * time.Second,
					libreOfficeRestartThreshold: 10,
					logger:                      zap.NewNop(),
				}
				mod.listener = newLibreOfficeListener(
					mod.logger,
					gotenberg.NewFileSystem(),
					mod.libreOfficeBinPath,
					mod.libreOfficeStartTimeout,
					mod.libreOfficeRestartThreshold,
				)

				return mod
			}(),
			ctx:       context.Background(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				PDFformat: gotenberg.FormatPDFA2b,
			},
			teardown: func(mod UNO) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return mod.Stop(ctx)
			},
		},
		{
			scenario: "convert to PDF/A-3b",
			mod: func() UNO {
				mod := UNO{
					unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
					libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
					libreOfficeStartTimeout:     time.Duration(10) * time.Second,
					libreOfficeRestartThreshold: 10,
					logger:                      zap.NewNop(),
				}
				mod.listener = newLibreOfficeListener(
					mod.logger,
					gotenberg.NewFileSystem(),
					mod.libreOfficeBinPath,
					mod.libreOfficeStartTimeout,
					mod.libreOfficeRestartThreshold,
				)

				return mod
			}(),
			ctx:       context.Background(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				PDFformat: gotenberg.FormatPDFA3b,
			},
			teardown: func(mod UNO) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return mod.Stop(ctx)
			},
		},
		{
			scenario: "convert to invalid PDF format",
			mod: func() UNO {
				mod := UNO{
					unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
					libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
					libreOfficeStartTimeout:     time.Duration(10) * time.Second,
					libreOfficeRestartThreshold: 10,
					logger:                      zap.NewNop(),
				}
				mod.listener = newLibreOfficeListener(
					mod.logger,
					gotenberg.NewFileSystem(),
					mod.libreOfficeBinPath,
					mod.libreOfficeStartTimeout,
					mod.libreOfficeRestartThreshold,
				)

				return mod
			}(),
			ctx:       context.Background(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				PDFformat: "foo",
			},
			expectPDFErr: true,
			teardown: func(mod UNO) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return mod.Stop(ctx)
			},
		},
		{
			scenario: "expired context",
			mod: func() UNO {
				mod := UNO{
					unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
					libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
					libreOfficeStartTimeout:     time.Duration(10) * time.Second,
					libreOfficeRestartThreshold: 10,
					logger:                      zap.NewNop(),
				}
				mod.listener = newLibreOfficeListener(
					mod.logger,
					gotenberg.NewFileSystem(),
					mod.libreOfficeBinPath,
					mod.libreOfficeStartTimeout,
					mod.libreOfficeRestartThreshold,
				)

				return mod
			}(),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			}(),
			logger:       zap.NewNop(),
			inputPath:    "/tests/test/testdata/libreoffice/sample1.docx",
			expectPDFErr: true,
		},
		{
			scenario: "cannot lock long-running LibreOffice listener",
			mod: UNO{
				unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
				libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
				libreOfficeStartTimeout:     time.Duration(10) * time.Second,
				libreOfficeRestartThreshold: 10,
				listener: listenerMock{
					lockMock: func(ctx context.Context, logger *zap.Logger) error {
						return errors.New("foo")
					},
				},
				logger: zap.NewNop(),
			},
			ctx:          context.Background(),
			logger:       zap.NewNop(),
			inputPath:    "/tests/test/testdata/libreoffice/sample1.docx",
			expectPDFErr: true,
		},
		{
			scenario: "cannot unlock long-running LibreOffice listener",
			mod: UNO{
				unoconvBinPath:              os.Getenv("UNOCONV_BIN_PATH"),
				libreOfficeBinPath:          os.Getenv("LIBREOFFICE_BIN_PATH"),
				libreOfficeStartTimeout:     time.Duration(10) * time.Second,
				libreOfficeRestartThreshold: 10,
				listener: listenerMock{
					lockMock: func(ctx context.Context, logger *zap.Logger) error {
						return nil
					},
					unlockMock: func(logger *zap.Logger) error {
						return errors.New("foo")
					},
					portMock: func() int {
						return 2002
					},
				},
				logger: zap.NewNop(),
			},
			ctx:          context.Background(),
			logger:       zap.NewNop(),
			inputPath:    "/tests/test/testdata/libreoffice/sample1.docx",
			expectPDFErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			defer func() {
				if tc.teardown == nil {
					return
				}

				err := tc.teardown(tc.mod)
				if err != nil {
					t.Errorf("expected no error from tc.teardown(), but got: %v", err)
				}
			}()

			fs := gotenberg.NewFileSystem()
			outputDir, err := fs.MkdirAll()
			if err != nil {
				t.Fatalf("test %s: expected error but got: %v", tc.scenario, err)
			}

			defer func() {
				err := os.RemoveAll(fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("test %s: expected no error while cleaning up but got: %v", tc.scenario, err)
				}
			}()

			err = tc.mod.PDF(tc.ctx, tc.logger, tc.inputPath, outputDir+"/foo.pdf", tc.options)

			if tc.expectPDFErr && err == nil {
				t.Fatalf("expected mod.PDF() error, but got none")
			}

			if !tc.expectPDFErr && err != nil {
				t.Fatalf("expected no error from mod.PDF(), but got: %v", err)
			}
		})
	}
}

func TestUNO_Extensions(t *testing.T) {
	mod := new(UNO)
	extensions := mod.Extensions()

	actual := len(extensions)
	expect := 79

	if actual != expect {
		t.Errorf("expected %d extensions, but got %d", expect, actual)
	}
}

func TestUNO_UNO(t *testing.T) {
	mod := new(UNO)

	_, err := mod.UNO()
	if err != nil {
		t.Errorf("expected no error from mod.UNO(), but got: %v", err)
	}
}

type listenerMock struct {
	startMock   func(logger *zap.Logger) error
	stopMock    func(logger *zap.Logger) error
	restartMock func(logger *zap.Logger) error
	lockMock    func(ctx context.Context, logger *zap.Logger) error
	unlockMock  func(logger *zap.Logger) error
	portMock    func() int
	queueMock   func() int
	healthyMock func() bool
}

func (listener listenerMock) start(logger *zap.Logger) error {
	return listener.startMock(logger)
}

func (listener listenerMock) stop(logger *zap.Logger) error {
	return listener.stopMock(logger)
}

func (listener listenerMock) restart(logger *zap.Logger) error {
	return listener.restartMock(logger)
}

func (listener listenerMock) lock(ctx context.Context, logger *zap.Logger) error {
	return listener.lockMock(ctx, logger)
}

func (listener listenerMock) unlock(logger *zap.Logger) error {
	return listener.unlockMock(logger)
}

func (listener listenerMock) port() int {
	return listener.portMock()
}

func (listener listenerMock) queue() int {
	return listener.queueMock()
}

func (listener listenerMock) healthy() bool {
	return listener.healthyMock()
}

// Interface guards.
var (
	_ listener = (*listenerMock)(nil)
)
