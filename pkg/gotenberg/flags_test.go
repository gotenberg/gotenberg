package gotenberg

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	flag "github.com/spf13/pflag"
)

func TestParsedFlags_MustString(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.String("foo", "", "")

	err := fs.Parse([]string{"--foo=foo"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustString(tc.name)
		})
	}
}

func TestParsedFlags_MustDeprecatedString(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		rawFlags    []string
		expectValue string
	}{
		{
			scenario:    "deprecated flag value",
			rawFlags:    []string{"--foo=foo"},
			expectValue: "foo",
		},
		{
			scenario:    "non-deprecated flag value",
			rawFlags:    []string{"--bar=bar"},
			expectValue: "bar",
		},
		{
			scenario:    "deprecated flag value > non-deprecated flag value",
			rawFlags:    []string{"--foo=foo", "--bar=bar"},
			expectValue: "foo",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := flag.NewFlagSet("tests", flag.ContinueOnError)
			fs.String("foo", "", "")
			fs.String("bar", "", "")

			parsedFlags := ParsedFlags{FlagSet: fs}

			err := parsedFlags.Parse(tc.rawFlags)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			actual := parsedFlags.MustDeprecatedString("foo", "bar")
			if actual != tc.expectValue {
				t.Errorf("expected '%s' but got '%s'", tc.expectValue, actual)
			}
		})
	}
}

func TestParsedFlags_MustStringSlice(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.StringSlice("foo", make([]string, 0), "")

	err := fs.Parse([]string{"--foo=foo", "--foo=bar", "--foo=baz"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustStringSlice(tc.name)
		})
	}
}

func TestParsedFlags_MustDeprecatedStringSlice(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		rawFlags    []string
		expectValue []string
	}{
		{
			scenario:    "deprecated flag value",
			rawFlags:    []string{"--foo=foo"},
			expectValue: []string{"foo"},
		},
		{
			scenario:    "non-deprecated flag value",
			rawFlags:    []string{"--bar=bar"},
			expectValue: []string{"bar"},
		},
		{
			scenario:    "deprecated flag value > non-deprecated flag value",
			rawFlags:    []string{"--foo=foo", "--bar=bar"},
			expectValue: []string{"foo"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := flag.NewFlagSet("tests", flag.ContinueOnError)
			fs.StringSlice("foo", make([]string, 0), "")
			fs.StringSlice("bar", make([]string, 0), "")

			parsedFlags := ParsedFlags{FlagSet: fs}

			err := parsedFlags.Parse(tc.rawFlags)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			actual := parsedFlags.MustDeprecatedStringSlice("foo", "bar")
			if !reflect.DeepEqual(actual, tc.expectValue) {
				t.Errorf("expected %+v but got %+v", tc.expectValue, actual)
			}
		})
	}
}

func TestParsedFlags_MustBool(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Bool("foo", false, "")

	err := fs.Parse([]string{"--foo=true"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustBool(tc.name)
		})
	}
}

func TestParsedFlags_MustDeprecatedBool(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		rawFlags    []string
		expectValue bool
	}{
		{
			scenario:    "deprecated flag value",
			rawFlags:    []string{"--foo=true"},
			expectValue: true,
		},
		{
			scenario:    "non-deprecated flag value",
			rawFlags:    []string{"--bar=false"},
			expectValue: false,
		},
		{
			scenario:    "deprecated flag value > non-deprecated flag value",
			rawFlags:    []string{"--foo=true", "--bar=false"},
			expectValue: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := flag.NewFlagSet("tests", flag.ContinueOnError)
			fs.Bool("foo", false, "")
			fs.Bool("bar", true, "")

			parsedFlags := ParsedFlags{FlagSet: fs}

			err := parsedFlags.Parse(tc.rawFlags)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			actual := parsedFlags.MustDeprecatedBool("foo", "bar")
			if actual != tc.expectValue {
				t.Errorf("expected %v but got %v", tc.expectValue, actual)
			}
		})
	}
}

func TestParsedFlags_MustInt64(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Int64("foo", 0, "")

	err := fs.Parse([]string{"--foo=1"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustInt64(tc.name)
		})
	}
}

func TestParsedFlags_MustDeprecatedInt64(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		rawFlags    []string
		expectValue int64
	}{
		{
			scenario:    "deprecated flag value",
			rawFlags:    []string{"--foo=1"},
			expectValue: 1,
		},
		{
			scenario:    "non-deprecated flag value",
			rawFlags:    []string{"--bar=2"},
			expectValue: 2,
		},
		{
			scenario:    "deprecated flag value > non-deprecated flag value",
			rawFlags:    []string{"--foo=1", "--bar=2"},
			expectValue: 1,
		},
	} {
		fs := flag.NewFlagSet("tests", flag.ContinueOnError)
		fs.Int64("foo", 0, "")
		fs.Int64("bar", 0, "")

		parsedFlags := ParsedFlags{FlagSet: fs}

		err := parsedFlags.Parse(tc.rawFlags)
		if err != nil {
			t.Fatalf("expected no error but got: %v", err)
		}

		actual := parsedFlags.MustDeprecatedInt64("foo", "bar")
		if actual != tc.expectValue {
			t.Errorf("expected %d but got %d", tc.expectValue, actual)
		}
	}
}

func TestParsedFlags_MustInt(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Int("foo", 0, "")

	err := fs.Parse([]string{"--foo=1"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustInt(tc.name)
		})
	}
}

func TestParsedFlags_MustDeprecatedInt(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		rawFlags    []string
		expectValue int
	}{
		{
			scenario:    "deprecated flag value",
			rawFlags:    []string{"--foo=1"},
			expectValue: 1,
		},
		{
			scenario:    "non-deprecated flag value",
			rawFlags:    []string{"--bar=2"},
			expectValue: 2,
		},
		{
			scenario:    "deprecated flag value > non-deprecated flag value",
			rawFlags:    []string{"--foo=1", "--bar=2"},
			expectValue: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := flag.NewFlagSet("tests", flag.ContinueOnError)
			fs.Int("foo", 0, "")
			fs.Int("bar", 0, "")

			parsedFlags := ParsedFlags{FlagSet: fs}

			err := parsedFlags.Parse(tc.rawFlags)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			actual := parsedFlags.MustDeprecatedInt("foo", "bar")
			if actual != tc.expectValue {
				t.Errorf("expected %d but got %d", tc.expectValue, actual)
			}
		})
	}
}

func TestParsedFlags_MustFloat64(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Float64("foo", 1.0, "")

	err := fs.Parse([]string{"--foo=2.0"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustFloat64(tc.name)
		})
	}
}

func TestParsedFlags_MustDeprecatedFloat64(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		rawFlags    []string
		expectValue float64
	}{
		{
			scenario:    "deprecated flag value",
			rawFlags:    []string{"--foo=1.0"},
			expectValue: 1.0,
		},
		{
			scenario:    "non-deprecated flag value",
			rawFlags:    []string{"--bar=2.0"},
			expectValue: 2.0,
		},
		{
			scenario:    "deprecated flag value > non-deprecated flag value",
			rawFlags:    []string{"--foo=1.0", "--bar=2.0"},
			expectValue: 1.0,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := flag.NewFlagSet("tests", flag.ContinueOnError)
			fs.Float64("foo", 0, "")
			fs.Float64("bar", 0, "")

			parsedFlags := ParsedFlags{FlagSet: fs}

			err := parsedFlags.Parse(tc.rawFlags)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			actual := parsedFlags.MustDeprecatedFloat64("foo", "bar")
			if actual != tc.expectValue {
				t.Errorf("expected %f but got %f", tc.expectValue, actual)
			}
		})
	}
}

func TestParsedFlags_MustDuration(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Duration("foo", time.Duration(1)*time.Second, "")

	err := fs.Parse([]string{"--foo=2m"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustDuration(tc.name)
		})
	}
}

func TestParsedFlags_MustDeprecatedDuration(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		rawFlags    []string
		expectValue time.Duration
	}{
		{
			scenario:    "deprecated flag value",
			rawFlags:    []string{"--foo=1s"},
			expectValue: time.Duration(1) * time.Second,
		},
		{
			scenario:    "non-deprecated flag value",
			rawFlags:    []string{"--bar=2s"},
			expectValue: time.Duration(2) * time.Second,
		},
		{
			scenario:    "deprecated flag value > non-deprecated flag value",
			rawFlags:    []string{"--foo=1s", "--bar=2s"},
			expectValue: time.Duration(1) * time.Second,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := flag.NewFlagSet("tests", flag.ContinueOnError)
			fs.Duration("foo", 0, "")
			fs.Duration("bar", 0, "")

			parsedFlags := ParsedFlags{FlagSet: fs}

			err := parsedFlags.Parse(tc.rawFlags)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			actual := parsedFlags.MustDeprecatedDuration("foo", "bar")
			if actual != tc.expectValue {
				t.Errorf("expected '%s' but got '%s'", tc.expectValue, actual)
			}
		})
	}
}

func TestParsedFlags_MustHumanReadableBytesString(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.String("foo", "1MB", "")
	fs.String("bar", "1MB", "")

	err := fs.Parse([]string{"--foo=1GB", "--bar=foo"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustHumanReadableBytesString(tc.name)
		})
	}
}

func TestParsedFlags_MustDeprecatedHumanReadableBytesString(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		rawFlags    []string
		expectValue string
	}{
		{
			scenario:    "deprecated flag value",
			rawFlags:    []string{"--foo=1MB"},
			expectValue: "1MB",
		},
		{
			scenario:    "non-deprecated flag value",
			rawFlags:    []string{"--bar=2MB"},
			expectValue: "2MB",
		},
		{
			scenario:    "deprecated flag value > non-deprecated flag value",
			rawFlags:    []string{"--foo=1MB", "--bar=2MB"},
			expectValue: "1MB",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := flag.NewFlagSet("tests", flag.ContinueOnError)
			fs.String("foo", "", "")
			fs.String("bar", "", "")

			parsedFlags := ParsedFlags{FlagSet: fs}

			err := parsedFlags.Parse(tc.rawFlags)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			actual := parsedFlags.MustDeprecatedHumanReadableBytesString("foo", "bar")
			if actual != tc.expectValue {
				t.Errorf("expected '%s' but got '%s'", tc.expectValue, actual)
			}
		})
	}
}

func TestParsedFlags_MustRegexp(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.String("foo", "", "")
	fs.String("bar", "", "")

	err := fs.Parse([]string{"--foo=", "--bar=*"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustRegexp(tc.name)
		})
	}
}

func TestParsedFlags_MustDeprecatedRegexp(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		rawFlags    []string
		expectValue *regexp.Regexp
	}{
		{
			scenario:    "deprecated flag value",
			rawFlags:    []string{"--foo=foo"},
			expectValue: regexp.MustCompile("foo"),
		},
		{
			scenario:    "non-deprecated flag value",
			rawFlags:    []string{"--bar=bar"},
			expectValue: regexp.MustCompile("bar"),
		},
		{
			scenario:    "deprecated flag value > non-deprecated flag value",
			rawFlags:    []string{"--foo=foo", "--bar=bar"},
			expectValue: regexp.MustCompile("foo"),
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := flag.NewFlagSet("tests", flag.ContinueOnError)
			fs.String("foo", "", "")
			fs.String("bar", "", "")

			parsedFlags := ParsedFlags{FlagSet: fs}

			err := parsedFlags.Parse(tc.rawFlags)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			actual := parsedFlags.MustDeprecatedRegexp("foo", "bar")
			if actual.String() != tc.expectValue.String() {
				t.Errorf("expected '%s' but got '%s'", tc.expectValue.String(), actual.String())
			}
		})
	}
}
