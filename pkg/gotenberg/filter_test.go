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
		allowed       *regexp2.Regexp
		denied        *regexp2.Regexp
		s             string
		deadline      time.Time
		expectError   bool
		expectedError error
	}{
		{
			scenario:      "DeadlineExceeded (allowed)",
			allowed:       regexp2.MustCompile("foo", 0),
			denied:        regexp2.MustCompile("", 0),
			s:             "foo",
			deadline:      time.Now().Add(time.Duration(-1) * time.Hour),
			expectError:   true,
			expectedError: context.DeadlineExceeded,
		},
		{
			scenario:      "ErrFiltered (allowed)",
			allowed:       regexp2.MustCompile("foo", 0),
			denied:        regexp2.MustCompile("", 0),
			s:             "bar",
			deadline:      time.Now().Add(time.Duration(5) * time.Second),
			expectError:   true,
			expectedError: ErrFiltered,
		},
		{
			scenario:      "DeadlineExceeded (denied)",
			allowed:       regexp2.MustCompile("", 0),
			denied:        regexp2.MustCompile("foo", 0),
			s:             "foo",
			deadline:      time.Now().Add(time.Duration(-1) * time.Hour),
			expectError:   true,
			expectedError: context.DeadlineExceeded,
		},
		{
			scenario:      "ErrFiltered (denied)",
			allowed:       regexp2.MustCompile("", 0),
			denied:        regexp2.MustCompile("foo", 0),
			s:             "foo",
			deadline:      time.Now().Add(time.Duration(5) * time.Second),
			expectError:   true,
			expectedError: ErrFiltered,
		},
		{
			scenario:    "success",
			allowed:     regexp2.MustCompile("", 0),
			denied:      regexp2.MustCompile("", 0),
			s:           "foo",
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
