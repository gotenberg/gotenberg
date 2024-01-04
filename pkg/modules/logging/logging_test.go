package logging

import (
	"fmt"
	"reflect"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestLogging_Descriptor(t *testing.T) {
	descriptor := new(Logging).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Logging))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestLogging_Provision(t *testing.T) {
	for _, tc := range []struct {
		scenario           string
		level              string
		format             string
		fieldsPrefix       string
		expectLevel        string
		expectFormat       string
		expectFieldsPrefix string
	}{
		{
			scenario:           "default values",
			expectLevel:        infoLoggingLevel,
			expectFormat:       autoLoggingFormat,
			expectFieldsPrefix: "",
		},
		{
			scenario:           "explicit values",
			level:              "debug",
			format:             "json",
			fieldsPrefix:       "gotenberg",
			expectLevel:        debugLoggingLevel,
			expectFormat:       jsonLoggingFormat,
			expectFieldsPrefix: "gotenberg",
		},
		{
			scenario:           "wrong values", // no validation at this point.
			level:              "foo",
			format:             "foo",
			expectLevel:        "foo",
			expectFormat:       "foo",
			expectFieldsPrefix: "",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var flags []string

			if tc.level != "" {
				flags = append(flags, "--log-level", tc.level)
			}

			if tc.format != "" {
				flags = append(flags, "--log-format", tc.format)
			}

			if tc.fieldsPrefix != "" {
				flags = append(flags, "--log-fields-prefix", tc.fieldsPrefix)
			}

			logging := new(Logging)
			fs := logging.Descriptor().FlagSet

			err := fs.Parse(flags)
			if err != nil {
				t.Fatalf("expected no error while parsing flags but got: %v", err)
			}

			ctx := gotenberg.NewContext(gotenberg.ParsedFlags{FlagSet: fs}, nil)

			err = logging.Provision(ctx)
			if err != nil {
				t.Fatalf("expected no error while provisioning but got: %v", err)
			}

			if logging.level != tc.expectLevel {
				t.Errorf("expected logging level '%s' but got '%s'", tc.expectLevel, logging.level)
			}

			if logging.format != tc.expectFormat {
				t.Errorf("expected logging format '%s' but got '%s'", tc.expectFormat, logging.format)
			}

			if logging.fieldsPrefix != tc.expectFieldsPrefix {
				t.Errorf("expected logging fields prefix '%s' but got '%s'", tc.expectFieldsPrefix, logging.fieldsPrefix)
			}
		})
	}
}

func TestLogging_Validate(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		level       string
		format      string
		expectError bool
	}{
		{
			scenario:    "invalid level",
			level:       "foo",
			expectError: true,
		},
		{
			scenario:    "invalid format",
			level:       debugLoggingLevel,
			format:      "foo",
			expectError: true,
		},
		{
			scenario: "valid level and format",
			level:    debugLoggingLevel,
			format:   autoLoggingFormat,
		},
	} {
		logging := new(Logging)
		logging.level = tc.level
		logging.format = tc.format

		err := logging.Validate()

		if tc.expectError && err == nil {
			t.Errorf("%s: expected error but got: %v", tc.scenario, err)
		}

		if !tc.expectError && err != nil {
			t.Errorf("%s: expected no error but got: %v", tc.scenario, err)
		}
	}
}

func TestLogging_Logger(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		level        string
		format       string
		fieldsPrefix string
		expectError  bool
	}{
		{
			scenario:    "invalid level",
			level:       "foo",
			expectError: true,
		},
		{
			scenario:    "invalid format",
			level:       debugLoggingLevel,
			format:      "foo",
			expectError: true,
		},
		{
			scenario: "valid level and format",
			level:    debugLoggingLevel,
			format:   autoLoggingFormat,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logging := new(Logging)
			logging.level = tc.level
			logging.format = tc.format
			logging.fieldsPrefix = tc.fieldsPrefix

			_, err := logging.Logger(&gotenberg.ModuleMock{
				DescriptorMock: func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "mock", New: nil}
				},
			})

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

func TestCustomCore(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		level        zapcore.Level
		fieldsPrefix string
		expectEntry  bool
	}{
		{
			scenario:     "level enabled",
			level:        zapcore.DebugLevel,
			fieldsPrefix: "gotenberg",
			expectEntry:  true,
		},
		{
			scenario:    "no fields prefix",
			level:       zapcore.DebugLevel,
			expectEntry: true,
		},
		{
			scenario: "level disabled",
			level:    zapcore.ErrorLevel,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			core, obsvr := observer.New(tc.level)
			lgr := zap.New(customCore{
				Core:         core,
				fieldsPrefix: tc.fieldsPrefix,
			}).With(zap.String("a_field", "a value"))

			lgr.Debug("a debug message", zap.String("another_field", "another value"))

			entries := obsvr.TakeAll()

			if tc.expectEntry && len(entries) == 0 {
				t.Fatal("expected an entry")
			}

			if !tc.expectEntry && len(entries) != 0 {
				t.Fatal("expected no entry")
			}

			var prefix string
			if tc.fieldsPrefix != "" {
				prefix = tc.fieldsPrefix + "_"
			}

			for _, entry := range entries {
				fields := entry.Context

				if len(fields) != 2 {
					t.Fatalf("expected 2 fields but got %d", len(fields))
				}

				if fields[0].Key != fmt.Sprintf("%sa_field", prefix) {
					t.Errorf("expected 'gotenberg_a_field' but got '%s'", fields[0].Key)
				}

				if fields[1].Key != fmt.Sprintf("%sanother_field", prefix) {
					t.Errorf("expected 'gotenberg_another_field' but got '%s'", fields[1].Key)
				}
			}
		})
	}
}

func Test_newLogLevel(t *testing.T) {
	for _, tc := range []struct {
		scenario       string
		level          string
		expectZapLevel zapcore.Level
		expectError    bool
	}{
		{
			scenario:       "error level",
			level:          errorLoggingLevel,
			expectZapLevel: zapcore.ErrorLevel,
		},
		{
			scenario:       "warning level",
			level:          warnLoggingLevel,
			expectZapLevel: zapcore.WarnLevel,
		},
		{
			scenario:       "info level",
			level:          infoLoggingLevel,
			expectZapLevel: zapcore.InfoLevel,
		},
		{
			scenario:       "debug level",
			level:          debugLoggingLevel,
			expectZapLevel: zapcore.DebugLevel,
		},
		{
			scenario:       "invalid level",
			level:          "foo",
			expectZapLevel: zapcore.InvalidLevel,
			expectError:    true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			actual, err := newLogLevel(tc.level)

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectZapLevel != actual {
				t.Errorf("expected %d level but got %d", tc.expectZapLevel, actual)
			}
		})
	}
}

func Test_newLogEncoder(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		format      string
		expectError bool
	}{
		{
			scenario: "auto format",
			format:   autoLoggingFormat,
		},
		{
			scenario: "text format",
			format:   textLoggingFormat,
		},
		{
			scenario: "json format",
			format:   jsonLoggingFormat,
		},
		{
			scenario:    "invalid format",
			format:      "foo",
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			_, err := newLogEncoder(tc.format)

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}
