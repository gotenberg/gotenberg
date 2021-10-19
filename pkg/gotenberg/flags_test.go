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

	for i, tc := range []struct {
		name        string
		expectPanic bool
	}{
		{
			name: "foo",
		},
		{
			name:        "bar",
			expectPanic: true,
		},
	} {
		func() {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			parsedFlags.MustString(tc.name)
		}()
	}
}

func TestParsedFlags_MustDeprecatedString(t *testing.T) {
	for i, tc := range []struct {
		rawFlags    []string
		expectValue string
	}{
		{
			rawFlags:    []string{"--foo=foo"},
			expectValue: "foo",
		},
		{
			rawFlags:    []string{"--bar=bar"},
			expectValue: "bar",
		},
		{
			rawFlags:    []string{"--foo=foo", "--bar=bar"},
			expectValue: "foo",
		},
	} {
		fs := flag.NewFlagSet("tests", flag.ContinueOnError)
		fs.String("foo", "", "")
		fs.String("bar", "", "")

		parsedFlags := ParsedFlags{FlagSet: fs}

		err := parsedFlags.Parse(tc.rawFlags)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		actual := parsedFlags.MustDeprecatedString("foo", "bar")
		if actual != tc.expectValue {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expectValue, actual)
		}
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

	for i, tc := range []struct {
		name        string
		expectPanic bool
	}{
		{
			name: "foo",
		},
		{
			name:        "bar",
			expectPanic: true,
		},
	} {
		func() {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			parsedFlags.MustStringSlice(tc.name)
		}()
	}
}

func TestParsedFlags_MustDeprecatedStringSlice(t *testing.T) {
	for i, tc := range []struct {
		rawFlags    []string
		expectValue []string
	}{
		{
			rawFlags:    []string{"--foo=foo"},
			expectValue: []string{"foo"},
		},
		{
			rawFlags:    []string{"--bar=bar"},
			expectValue: []string{"bar"},
		},
		{
			rawFlags:    []string{"--foo=foo", "--bar=bar"},
			expectValue: []string{"foo"},
		},
	} {
		fs := flag.NewFlagSet("tests", flag.ContinueOnError)
		fs.StringSlice("foo", make([]string, 0), "")
		fs.StringSlice("bar", make([]string, 0), "")

		parsedFlags := ParsedFlags{FlagSet: fs}

		err := parsedFlags.Parse(tc.rawFlags)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		actual := parsedFlags.MustDeprecatedStringSlice("foo", "bar")
		if !reflect.DeepEqual(actual, tc.expectValue) {
			t.Errorf("test %d: expected %+v but got %+v", i, tc.expectValue, actual)
		}
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

	for i, tc := range []struct {
		name        string
		expectPanic bool
	}{
		{
			name: "foo",
		},
		{
			name:        "bar",
			expectPanic: true,
		},
	} {
		func() {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			parsedFlags.MustBool(tc.name)
		}()
	}
}

func TestParsedFlags_MustDeprecatedBool(t *testing.T) {
	for i, tc := range []struct {
		rawFlags    []string
		expectValue bool
	}{
		{
			rawFlags:    []string{"--foo=true"},
			expectValue: true,
		},
		{
			rawFlags:    []string{"--bar=false"},
			expectValue: false,
		},
		{
			rawFlags:    []string{"--foo=true", "--bar=false"},
			expectValue: true,
		},
	} {
		fs := flag.NewFlagSet("tests", flag.ContinueOnError)
		fs.Bool("foo", false, "")
		fs.Bool("bar", true, "")

		parsedFlags := ParsedFlags{FlagSet: fs}

		err := parsedFlags.Parse(tc.rawFlags)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		actual := parsedFlags.MustDeprecatedBool("foo", "bar")
		if actual != tc.expectValue {
			t.Errorf("test %d: expected %v but got %v", i, tc.expectValue, actual)
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

	for i, tc := range []struct {
		name        string
		expectPanic bool
	}{
		{
			name: "foo",
		},
		{
			name:        "bar",
			expectPanic: true,
		},
	} {
		func() {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			parsedFlags.MustInt(tc.name)
		}()
	}
}

func TestParsedFlags_MustDeprecatedInt(t *testing.T) {
	for i, tc := range []struct {
		rawFlags    []string
		expectValue int
	}{
		{
			rawFlags:    []string{"--foo=1"},
			expectValue: 1,
		},
		{
			rawFlags:    []string{"--bar=2"},
			expectValue: 2,
		},
		{
			rawFlags:    []string{"--foo=1", "--bar=2"},
			expectValue: 1,
		},
	} {
		fs := flag.NewFlagSet("tests", flag.ContinueOnError)
		fs.Int("foo", 0, "")
		fs.Int("bar", 0, "")

		parsedFlags := ParsedFlags{FlagSet: fs}

		err := parsedFlags.Parse(tc.rawFlags)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		actual := parsedFlags.MustDeprecatedInt("foo", "bar")
		if actual != tc.expectValue {
			t.Errorf("test %d: expected %d but got %d", i, tc.expectValue, actual)
		}
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

	for i, tc := range []struct {
		name        string
		expectPanic bool
	}{
		{
			name: "foo",
		},
		{
			name:        "bar",
			expectPanic: true,
		},
	} {
		func() {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			parsedFlags.MustFloat64(tc.name)
		}()
	}
}

func TestParsedFlags_MustDeprecatedFloat64(t *testing.T) {
	for i, tc := range []struct {
		rawFlags    []string
		expectValue float64
	}{
		{
			rawFlags:    []string{"--foo=1.0"},
			expectValue: 1.0,
		},
		{
			rawFlags:    []string{"--bar=2.0"},
			expectValue: 2.0,
		},
		{
			rawFlags:    []string{"--foo=1.0", "--bar=2.0"},
			expectValue: 1.0,
		},
	} {
		fs := flag.NewFlagSet("tests", flag.ContinueOnError)
		fs.Float64("foo", 0, "")
		fs.Float64("bar", 0, "")

		parsedFlags := ParsedFlags{FlagSet: fs}

		err := parsedFlags.Parse(tc.rawFlags)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		actual := parsedFlags.MustDeprecatedFloat64("foo", "bar")
		if actual != tc.expectValue {
			t.Errorf("test %d: expected %f but got %f", i, tc.expectValue, actual)
		}
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

	for i, tc := range []struct {
		name        string
		expectPanic bool
	}{
		{
			name: "foo",
		},
		{
			name:        "bar",
			expectPanic: true,
		},
	} {
		func() {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			parsedFlags.MustDuration(tc.name)
		}()
	}
}

func TestParsedFlags_MustDeprecatedDuration(t *testing.T) {
	for i, tc := range []struct {
		rawFlags    []string
		expectValue time.Duration
	}{
		{
			rawFlags:    []string{"--foo=1s"},
			expectValue: time.Duration(1) * time.Second,
		},
		{
			rawFlags:    []string{"--bar=2s"},
			expectValue: time.Duration(2) * time.Second,
		},
		{
			rawFlags:    []string{"--foo=1s", "--bar=2s"},
			expectValue: time.Duration(1) * time.Second,
		},
	} {
		fs := flag.NewFlagSet("tests", flag.ContinueOnError)
		fs.Duration("foo", 0, "")
		fs.Duration("bar", 0, "")

		parsedFlags := ParsedFlags{FlagSet: fs}

		err := parsedFlags.Parse(tc.rawFlags)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		actual := parsedFlags.MustDeprecatedDuration("foo", "bar")
		if actual != tc.expectValue {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expectValue, actual)
		}
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

	for i, tc := range []struct {
		name        string
		expectPanic bool
	}{
		{
			name: "foo",
		},
		{
			name:        "bar",
			expectPanic: true,
		},
		{
			name:        "baz",
			expectPanic: true,
		},
	} {
		func() {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			parsedFlags.MustHumanReadableBytesString(tc.name)
		}()
	}
}

func TestParsedFlags_MustDeprecatedHumanReadableBytesString(t *testing.T) {
	for i, tc := range []struct {
		rawFlags    []string
		expectValue string
	}{
		{
			rawFlags:    []string{"--foo=1MB"},
			expectValue: "1MB",
		},
		{
			rawFlags:    []string{"--bar=2MB"},
			expectValue: "2MB",
		},
		{
			rawFlags:    []string{"--foo=1MB", "--bar=2MB"},
			expectValue: "1MB",
		},
	} {
		fs := flag.NewFlagSet("tests", flag.ContinueOnError)
		fs.String("foo", "", "")
		fs.String("bar", "", "")

		parsedFlags := ParsedFlags{FlagSet: fs}

		err := parsedFlags.Parse(tc.rawFlags)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		actual := parsedFlags.MustDeprecatedHumanReadableBytesString("foo", "bar")
		if actual != tc.expectValue {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expectValue, actual)
		}
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

	for i, tc := range []struct {
		name        string
		expectPanic bool
	}{
		{
			name: "foo",
		},
		{
			name:        "bar",
			expectPanic: true,
		},
		{
			name:        "baz",
			expectPanic: true,
		},
	} {
		func() {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			parsedFlags.MustRegexp(tc.name)
		}()
	}
}

func TestParsedFlags_MustDeprecatedRegexp(t *testing.T) {
	for i, tc := range []struct {
		rawFlags    []string
		expectValue *regexp.Regexp
	}{
		{
			rawFlags:    []string{"--foo=foo"},
			expectValue: regexp.MustCompile("foo"),
		},
		{
			rawFlags:    []string{"--bar=bar"},
			expectValue: regexp.MustCompile("bar"),
		},
		{
			rawFlags:    []string{"--foo=foo", "--bar=bar"},
			expectValue: regexp.MustCompile("foo"),
		},
	} {
		fs := flag.NewFlagSet("tests", flag.ContinueOnError)
		fs.String("foo", "", "")
		fs.String("bar", "", "")

		parsedFlags := ParsedFlags{FlagSet: fs}

		err := parsedFlags.Parse(tc.rawFlags)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		actual := parsedFlags.MustDeprecatedRegexp("foo", "bar")
		if actual.String() != tc.expectValue.String() {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expectValue.String(), actual.String())
		}
	}
}
