package logging

import (
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap/zapcore"
)

type ProtoModule struct {
	descriptor func() gotenberg.ModuleDescriptor
}

func (mod ProtoModule) Descriptor() gotenberg.ModuleDescriptor {
	return mod.descriptor()
}

func TestLogging_Descriptor(t *testing.T) {
	descriptor := Logging{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Logging))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestLogging_Provision(t *testing.T) {
	logging := new(Logging)
	fs := logging.Descriptor().FlagSet

	err := fs.Parse([]string{""})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	ctx := gotenberg.NewContext(gotenberg.ParsedFlags{FlagSet: fs}, nil)

	err = logging.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestLogging_Validate(t *testing.T) {
	for i, tc := range []struct {
		level, format string
		expectErr     bool
	}{
		{
			level:     "foo",
			expectErr: true,
		},
		{
			level:     debugLoggingLevel,
			format:    "foo",
			expectErr: true,
		},
		{
			level:  debugLoggingLevel,
			format: autoLoggingFormat,
		},
	} {
		mod := new(Logging)
		mod.level = tc.level
		mod.format = tc.format

		err := mod.Validate()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestLogging_Logger(t *testing.T) {
	for i, tc := range []struct {
		level, format string
		expectErr     bool
	}{
		{
			level:     "foo",
			expectErr: true,
		},
		{
			level:     debugLoggingLevel,
			format:    "foo",
			expectErr: true,
		},
		{
			level:  debugLoggingLevel,
			format: autoLoggingFormat,
		},
	} {
		mod := new(Logging)
		mod.level = tc.level
		mod.format = tc.format

		_, err := mod.Logger(ProtoModule{
			descriptor: func() gotenberg.ModuleDescriptor {
				return gotenberg.ModuleDescriptor{ID: "foo", New: nil}
			},
		})

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestNewLogLevel(t *testing.T) {
	for i, tc := range []struct {
		level          string
		expectZapLevel zapcore.Level
		expectErr      bool
	}{
		{
			level:          errorLoggingLevel,
			expectZapLevel: zapcore.ErrorLevel,
		},
		{
			level:          warnLoggingLevel,
			expectZapLevel: zapcore.WarnLevel,
		},
		{
			level:          infoLoggingLevel,
			expectZapLevel: zapcore.InfoLevel,
		},
		{
			level:          debugLoggingLevel,
			expectZapLevel: zapcore.DebugLevel,
		},
		{
			level:          "foo",
			expectZapLevel: -2,
			expectErr:      true,
		},
	} {
		actual, err := newLogLevel(tc.level)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		if tc.expectZapLevel != actual {
			t.Errorf("test %d: expected %d level but got %d", i, tc.expectZapLevel, actual)
		}
	}
}

func TestNewLogEncoder(t *testing.T) {
	for i, tc := range []struct {
		format    string
		expectErr bool
	}{
		{
			format: autoLoggingFormat,
		},
		{
			format: textLoggingFormat,
		},
		{
			format: jsonLoggingFormat,
		},
		{
			format:    "foo",
			expectErr: true,
		},
	} {
		_, err := newLogEncoder(tc.format)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

// Interface guards.
var (
	_ gotenberg.Module = (*ProtoModule)(nil)
)
