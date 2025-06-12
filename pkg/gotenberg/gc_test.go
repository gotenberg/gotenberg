package gotenberg

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

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
				p := fmt.Sprintf("%s/a_directory", os.TempDir())

				err := os.MkdirAll(p, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(fmt.Sprintf("%s/a_foo_file", p), []byte{1}, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(fmt.Sprintf("%s/a_bar_file", p), []byte{1}, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(fmt.Sprintf("%s/a_baz_file", p), []byte{1}, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return p
			}(),
			includeSubstr:   []string{"foo", path.Join(os.TempDir(), "/a_directory/a_bar_file")},
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

			err := GarbageCollect(zap.NewNop(), tc.rootPath, tc.includeSubstr, time.Now())

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
				p := fmt.Sprintf("%s/%s", tc.rootPath, name)
				_, err = os.Stat(p)
				if !os.IsNotExist(err) {
					t.Errorf("expected '%s' not to exist but it does: %v", p, err)
				}
			}

			for _, name := range tc.expectExists {
				p := fmt.Sprintf("%s/%s", tc.rootPath, name)
				_, err = os.Stat(p)
				if os.IsNotExist(err) {
					t.Errorf("expected '%s' to exist but it does not: %v", p, err)
				}
			}
		})
	}
}
