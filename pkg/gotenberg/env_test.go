package gotenberg

import (
	"os"
	"testing"
)

func TestStringEnv(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		key         string
		setEnv      func()
		expectVal   string
		expectError bool
	}{
		{
			scenario:    "non-existing environment variable",
			key:         "NON_EXISTING",
			expectVal:   "",
			expectError: true,
		},
		{
			scenario: "empty environment variable",
			key:      "EMPTY_STRING",
			setEnv: func() {
				err := os.Setenv("EMPTY_STRING", "")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			},
			expectVal:   "",
			expectError: true,
		},
		{
			scenario: "success",
			key:      "EXISTING_STRING_VALUE",
			setEnv: func() {
				err := os.Setenv("EXISTING_STRING_VALUE", "foo")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			},
			expectVal:   "foo",
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.setEnv != nil {
				tc.setEnv()
			}

			val, err := StringEnv(tc.key)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectVal != val {
				t.Errorf("expected value '%s' but got '%s'", tc.expectVal, val)
			}
		})
	}
}

func TestIntEnv(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		key         string
		setEnv      func()
		expectVal   int
		expectError bool
	}{
		{
			scenario: "empty environment variable",
			key:      "EMPTY_INT",
			setEnv: func() {
				err := os.Setenv("EMPTY_INT", "")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			},
			expectVal:   0,
			expectError: true,
		},
		{
			scenario: "non-integer value",
			key:      "NON_INTEGER",
			setEnv: func() {
				err := os.Setenv("NON_INTEGER", "foo")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			},
			expectVal:   0,
			expectError: true,
		},
		{
			scenario: "success",
			key:      "EXISTING_INT_VALUE",
			setEnv: func() {
				err := os.Setenv("EXISTING_INT_VALUE", "123")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			},
			expectVal:   123,
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.setEnv != nil {
				tc.setEnv()
			}

			val, err := IntEnv(tc.key)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectVal != val {
				t.Errorf("expected value %d but got %d", tc.expectVal, val)
			}
		})
	}
}
