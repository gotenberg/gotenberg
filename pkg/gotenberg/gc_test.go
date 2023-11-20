package gotenberg

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestGarbageCollect(t *testing.T) {
	for _, tc := range []struct {
		scenario        string
		rootPath        string
		includeSubstr   []string
		expectError     bool
		expectNotExists []string
		expectExists    []string
	}{
		{
			scenario:    "root path does not exist",
			rootPath:    uuid.NewString(),
			expectError: true,
		},
		{
			scenario: "remove include substrings",
			rootPath: func() string {
				path := fmt.Sprintf("%s/a_directory", os.TempDir())

				err := os.MkdirAll(path, 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/a_foo_file", path), []byte{1}, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(fmt.Sprintf("%s/a_bar_file", path), []byte{1}, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(fmt.Sprintf("%s/a_baz_file", path), []byte{1}, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return path
			}(),
			includeSubstr:   []string{"foo", fmt.Sprintf("%s/a_directory/a_bar_file", os.TempDir())},
			expectError:     false,
			expectExists:    []string{"a_baz_file"},
			expectNotExists: []string{"a_foo_file", "a_bar_file"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			defer func() {
				err := os.RemoveAll(tc.rootPath)
				if err != nil {
					t.Fatalf("expected no error while cleaning up but got: %v", err)
				}
			}()

			err := GarbageCollect(zap.NewNop(), tc.rootPath, tc.includeSubstr)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectError && err != nil {
				return
			}

			for _, name := range tc.expectNotExists {
				path := fmt.Sprintf("%s/%s", tc.rootPath, name)
				_, err = os.Stat(path)
				if !os.IsNotExist(err) {
					t.Errorf("expected '%s' not to exist but it does: %v", path, err)
				}
			}

			for _, name := range tc.expectExists {
				path := fmt.Sprintf("%s/%s", tc.rootPath, name)
				_, err = os.Stat(path)
				if os.IsNotExist(err) {
					t.Errorf("expected '%s' to exist but it does not: %v", path, err)
				}
			}
		})
	}
}
