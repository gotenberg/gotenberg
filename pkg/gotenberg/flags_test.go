package gotenberg

import (
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
