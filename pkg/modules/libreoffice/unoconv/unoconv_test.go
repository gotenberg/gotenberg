package unoconv

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

type ProtoModule struct {
	descriptor func() gotenberg.ModuleDescriptor
}

func (mod ProtoModule) Descriptor() gotenberg.ModuleDescriptor {
	return mod.descriptor()
}

type ProtoValidator struct {
	ProtoModule
	validate func() error
}

func (mod ProtoValidator) Validate() error {
	return mod.validate()
}

type ProtoLoggerProvider struct {
	ProtoModule
	logger func(mod gotenberg.Module) (*zap.Logger, error)
}

func (factory ProtoLoggerProvider) Logger(mod gotenberg.Module) (*zap.Logger, error) {
	return factory.logger(mod)
}

func TestUnoconv_Descriptor(t *testing.T) {
	descriptor := Unoconv{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Unoconv))

	if actual != expect {
		t.Errorf("expected '%'s' but got '%s'", expect, actual)
	}
}

func TestUnoconv_Provision(t *testing.T) {
	for i, tc := range []struct {
		ctx       *gotenberg.Context
		expectErr bool
	}{
		{
			ctx: gotenberg.NewContext(
				gotenberg.ParsedFlags{FlagSet: new(Unoconv).Descriptor().FlagSet},
				make([]gotenberg.ModuleDescriptor, 0),
			),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct {
					ProtoLoggerProvider
				}{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.logger = func(mod gotenberg.Module) (*zap.Logger, error) { return nil, errors.New("foo") }

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Unoconv).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct {
					ProtoLoggerProvider
				}{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.logger = func(mod gotenberg.Module) (*zap.Logger, error) { return zap.NewNop(), nil }

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Unoconv).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
		},
	} {
		mod := new(Unoconv)
		err := mod.Provision(tc.ctx)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestUnoconv_Validate(t *testing.T) {
	for i, tc := range []struct {
		binPath   string
		expectErr bool
	}{
		{
			expectErr: true,
		},
		{
			binPath:   "/foo",
			expectErr: true,
		},
		{
			binPath: os.Getenv("UNOCONV_BIN_PATH"),
		},
	} {
		mod := Unoconv{
			binPath: tc.binPath,
		}

		err := mod.Validate()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestUnoconv_Start(t *testing.T) {
	for i, tc := range []struct {
		mod       *Unoconv
		expectErr bool
	}{
		{
			mod: &Unoconv{
				disableListener: true,
				logger:          zap.NewNop(),
			},
		},
		{
			mod: &Unoconv{
				binPath:         os.Getenv("UNOCONV_BIN_PATH"),
				disableListener: false,
				logger:          zap.NewExample(),
			},
		},
	} {
		func() {
			err := tc.mod.Start()

			if tc.expectErr && err == nil {
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1)*time.Nanosecond)
			defer cancel()

			err = tc.mod.Stop(ctx)
			if err != nil {
				t.Errorf("test %d: expected not error but got: %v", i, err)
			}
		}()
	}
}

func TestUnoconv_StartupMessage(t *testing.T) {
	for i, tc := range []struct {
		disableListener bool
		expectMessage   string
	}{
		{
			disableListener: true,
			expectMessage:   "listener disabled",
		},
		{
			expectMessage: "listener started on port 0",
		},
	} {
		mod := Unoconv{
			disableListener: tc.disableListener,
		}

		actual := mod.StartupMessage()
		if actual != tc.expectMessage {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expectMessage, actual)
		}
	}
}

func TestUnoconv_Stop(t *testing.T) {
	for i, tc := range []struct {
		start           bool
		disableListener bool
		timeout         time.Duration
		expectErr       bool
	}{
		{
			disableListener: true,
		},
		{
			expectErr: true,
		},
		{
			start:   true,
			timeout: time.Duration(1) * time.Nanosecond,
		},
	} {
		func() {
			mod := &Unoconv{
				binPath:         os.Getenv("UNOCONV_BIN_PATH"),
				disableListener: tc.disableListener,
				logger:          zap.NewNop(),
			}

			if tc.start {
				err := mod.Start()
				if err != nil {
					t.Fatalf("test %d: expected no error but got: %v", i, err)
				}
			}

			var err error

			if tc.timeout == 0 {
				err = mod.Stop(context.TODO())
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
				defer cancel()

				err = mod.Stop(ctx)
			}

			if tc.expectErr && err == nil {
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}
		}()
	}
}

func TestUnoconv_Metrics(t *testing.T) {
	metrics, err := new(Unoconv).Metrics()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if len(metrics) != 3 {
		t.Fatalf("expected %d metrics, but got %d", 1, len(metrics))
	}

	actual := metrics[0].Read()
	if actual != 0 {
		t.Errorf("expected %d unoconv instances, but got %f", 0, actual)
	}

	actual = metrics[1].Read()
	if actual != 0 {
		t.Errorf("expected %d unoconv listener instances, but got %f", 0, actual)
	}

	actual = metrics[2].Read()
	if actual != 0 {
		t.Errorf("expected %d processes in the queue, but got %f", 0, actual)
	}
}

func TestUnoconv_Unoconv(t *testing.T) {
	mod := new(Unoconv)

	_, err := mod.Unoconv()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestUnoconv_PDF(t *testing.T) {
	for i, tc := range []struct {
		ctx       context.Context
		mod       Unoconv
		logger    *zap.Logger
		inputPath string
		options   Options
		expectErr bool
	}{
		{
			mod: Unoconv{
				binPath:         os.Getenv("UNOCONV_BIN_PATH"),
				disableListener: true,
			},
			logger:    zap.NewNop(),
			expectErr: true,
		},
		{
			ctx: context.Background(),
			mod: Unoconv{
				binPath:         os.Getenv("UNOCONV_BIN_PATH"),
				disableListener: true,
			},
			logger:    zap.NewExample(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				Landscape:  true,
				PageRanges: "1-2",
				PDFArchive: true,
			},
		},
		{
			ctx: context.Background(),
			mod: Unoconv{
				binPath:         os.Getenv("UNOCONV_BIN_PATH"),
				disableListener: true,
			},
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				PageRanges: "foo",
			},
			expectErr: true,
		},
		{
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.TODO())
				defer cancel()

				return ctx
			}(),
			mod: Unoconv{
				binPath:         os.Getenv("UNOCONV_BIN_PATH"),
				disableListener: true,
			},
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			expectErr: true,
		},
		{
			ctx: context.Background(),
			mod: Unoconv{
				binPath: os.Getenv("UNOCONV_BIN_PATH"),
				logger:  zap.NewNop(),
			},
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
		},
		{
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.TODO())
				defer cancel()

				return ctx
			}(),
			mod: Unoconv{
				binPath: os.Getenv("UNOCONV_BIN_PATH"),
				logger:  zap.NewNop(),
			},
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			expectErr: true,
		},
	} {
		func() {
			outputDir, err := gotenberg.MkdirAll()
			if err != nil {
				t.Fatalf("test %d: expected error but got: %v", i, err)
			}

			defer func() {
				err := os.RemoveAll(outputDir)
				if err != nil {
					t.Fatalf("test %d: expected no error but got: %v", i, err)
				}
			}()

			if !tc.mod.disableListener {
				err = tc.mod.Start()
				if err != nil {
					t.Fatalf("test %d: expected no error but got: %v", i, err)
				}

				// Let's give it some room to start.
				time.Sleep(time.Duration(1) * time.Second)
			}

			err = tc.mod.PDF(tc.ctx, tc.logger, tc.inputPath, outputDir+"/foo.pdf", tc.options)

			if tc.expectErr && err == nil {
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}

			if !tc.mod.disableListener {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1)*time.Nanosecond)
				defer cancel()

				err = tc.mod.Stop(ctx)
				if err != nil {
					t.Fatalf("test %d: expected no error but got: %v", i, err)
				}
			}
		}()
	}
}

func TestUnoconv_Extensions(t *testing.T) {
	mod := new(Unoconv)
	extensions := mod.Extensions()

	actual := len(extensions)
	expect := 76

	if actual != expect {
		t.Errorf("expected %d extensions but got %d", expect, actual)
	}
}

// Interface guards.
var (
	_ gotenberg.Module         = (*ProtoModule)(nil)
	_ gotenberg.Validator      = (*ProtoValidator)(nil)
	_ gotenberg.LoggerProvider = (*ProtoLoggerProvider)(nil)
	_ gotenberg.Module         = (*ProtoLoggerProvider)(nil)
)
