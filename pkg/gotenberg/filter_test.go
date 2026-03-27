package gotenberg

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
)

func TestFilterDeadline(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		allowed       []*regexp2.Regexp
		denied        []*regexp2.Regexp
		s             string
		deadline      time.Time
		expectError   bool
		expectedError error
	}{
		{
			scenario:      "DeadlineExceeded (allowed)",
			allowed:       []*regexp2.Regexp{regexp2.MustCompile("foo", 0)},
			denied:        nil,
			s:             "foo",
			deadline:      time.Now().Add(time.Duration(-1) * time.Hour),
			expectError:   true,
			expectedError: context.DeadlineExceeded,
		},
		{
			scenario:      "ErrFiltered (allowed, no match)",
			allowed:       []*regexp2.Regexp{regexp2.MustCompile("foo", 0)},
			denied:        nil,
			s:             "bar",
			deadline:      time.Now().Add(time.Duration(5) * time.Second),
			expectError:   true,
			expectedError: ErrFiltered,
		},
		{
			scenario:      "DeadlineExceeded (denied)",
			allowed:       nil,
			denied:        []*regexp2.Regexp{regexp2.MustCompile("foo", 0)},
			s:             "foo",
			deadline:      time.Now().Add(time.Duration(-1) * time.Hour),
			expectError:   true,
			expectedError: context.DeadlineExceeded,
		},
		{
			scenario:      "ErrFiltered (denied)",
			allowed:       nil,
			denied:        []*regexp2.Regexp{regexp2.MustCompile("foo", 0)},
			s:             "foo",
			deadline:      time.Now().Add(time.Duration(5) * time.Second),
			expectError:   true,
			expectedError: ErrFiltered,
		},
		{
			scenario:    "success (empty lists)",
			allowed:     nil,
			denied:      nil,
			s:           "foo",
			deadline:    time.Now().Add(time.Duration(5) * time.Second),
			expectError: false,
		},
		{
			scenario:    "multi-pattern allow list, second matches",
			allowed:     []*regexp2.Regexp{regexp2.MustCompile("^https://", 0), regexp2.MustCompile("^file:///tmp/", 0)},
			denied:      nil,
			s:           "file:///tmp/abc/index.html",
			deadline:    time.Now().Add(time.Duration(5) * time.Second),
			expectError: false,
		},
		{
			scenario:      "multi-pattern allow list, none matches",
			allowed:       []*regexp2.Regexp{regexp2.MustCompile("^https://", 0), regexp2.MustCompile("^ftp://", 0)},
			denied:        nil,
			s:             "file:///tmp/abc/index.html",
			deadline:      time.Now().Add(time.Duration(5) * time.Second),
			expectError:   true,
			expectedError: ErrFiltered,
		},
		{
			scenario:      "multi-pattern deny list, second matches",
			allowed:       nil,
			denied:        []*regexp2.Regexp{regexp2.MustCompile("^ftp://", 0), regexp2.MustCompile("^file:.*", 0)},
			s:             "file:///etc/passwd",
			deadline:      time.Now().Add(time.Duration(5) * time.Second),
			expectError:   true,
			expectedError: ErrFiltered,
		},
		{
			scenario:    "https URL passes deny list targeting file://",
			allowed:     nil,
			denied:      []*regexp2.Regexp{regexp2.MustCompile("^file:.*", 0)},
			s:           "https://example.com",
			deadline:    time.Now().Add(time.Duration(5) * time.Second),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			err := FilterDeadline(tc.allowed, tc.denied, tc.s, tc.deadline)

			if tc.expectError && err == nil {
				t.Fatal("expected an error but got none")
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v but got: %v", tc.expectedError, err)
			}
		})
	}
}

func TestRegexpToSlice(t *testing.T) {
	for _, tc := range []struct {
		scenario  string
		input     *regexp2.Regexp
		expectNil bool
		expectLen int
	}{
		{
			scenario:  "nil regexp",
			input:     nil,
			expectNil: true,
		},
		{
			scenario:  "empty regexp",
			input:     regexp2.MustCompile("", 0),
			expectNil: true,
		},
		{
			scenario:  "non-empty regexp",
			input:     regexp2.MustCompile("^file:.*", 0),
			expectNil: false,
			expectLen: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			result := RegexpToSlice(tc.input)

			if tc.expectNil && result != nil {
				t.Fatalf("expected nil but got: %v", result)
			}

			if !tc.expectNil && result == nil {
				t.Fatal("expected non-nil but got nil")
			}

			if !tc.expectNil && len(result) != tc.expectLen {
				t.Fatalf("expected length %d but got %d", tc.expectLen, len(result))
			}
		})
	}
}
