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
		expectErr       bool
		expectNotExists []string
		expectExists    []string
	}{
		{
			scenario:  "root path does not exist",
			rootPath:  uuid.NewString(),
			expectErr: true,
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
			expectExists:    []string{"a_baz_file"},
			expectNotExists: []string{"a_foo_file", "a_bar_file"},
		},
	} {
		func() {
			defer func() {
				err := os.RemoveAll(tc.rootPath)
				if err != nil {
					t.Fatalf("%s: expected no error while cleaning up but got: %v", tc.scenario, err)
				}
			}()

			err := GarbageCollect(zap.NewNop(), tc.rootPath, tc.includeSubstr)

			if !tc.expectErr && err != nil {
				t.Fatalf("%s: expected no error but got: %v", tc.scenario, err)
			}

			if tc.expectErr && err == nil {
				t.Fatalf("%s: expected error but got: %v", tc.scenario, err)
			}

			if tc.expectErr && err != nil {
				return
			}

			for _, name := range tc.expectNotExists {
				path := fmt.Sprintf("%s/%s", tc.rootPath, name)
				_, err = os.Stat(path)
				if !os.IsNotExist(err) {
					t.Errorf("%s: expected '%s' not to exist but it does: %v", tc.scenario, path, err)
				}
			}

			for _, name := range tc.expectExists {
				path := fmt.Sprintf("%s/%s", tc.rootPath, name)
				_, err = os.Stat(path)
				if os.IsNotExist(err) {
					t.Errorf("%s: expected '%s' to exist but it does not: %v", tc.scenario, path, err)
				}
			}
		}()
	}
}
