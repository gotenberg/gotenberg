package api

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestApi_ExecPaths(t *testing.T) {
	binDir := t.TempDir()
	for _, helper := range []string{"dirname", "grep"} {
		err := os.WriteFile(filepath.Join(binDir, helper), []byte("#!/bin/sh\n"), 0o755) //nolint:gosec // G306: must be executable
		if err != nil {
			t.Fatalf("write helper %q: %v", helper, err)
		}
	}
	t.Setenv("PATH", binDir)

	withScript := t.TempDir()
	err := os.WriteFile(filepath.Join(withScript, "soffice"), []byte("#!/bin/sh\n"), 0o755) //nolint:gosec // G306: must be executable
	if err != nil {
		t.Fatalf("write soffice: %v", err)
	}

	withoutScript := t.TempDir()

	for _, tc := range []struct {
		scenario       string
		programDirPath string
		expectPaths    []string
	}{
		{
			scenario:       "soffice script present",
			programDirPath: withScript,
			expectPaths: []string{
				"/usr/bin/lok",
				withScript,
				filepath.Join(withScript, "soffice"),
				filepath.Join(binDir, "dirname"),
				filepath.Join(binDir, "grep"),
			},
		},
		{
			scenario:       "soffice script missing",
			programDirPath: withoutScript,
			expectPaths: []string{
				"/usr/bin/lok",
				withoutScript,
				filepath.Join(binDir, "dirname"),
				filepath.Join(binDir, "grep"),
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			a := &Api{args: libreOfficeArguments{
				binPath:    filepath.Join(tc.programDirPath, "soffice.bin"),
				lokBinPath: "/usr/bin/lok",
			}}

			paths := a.ExecPaths()

			if !reflect.DeepEqual(paths, tc.expectPaths) {
				t.Errorf("expected %v but got %v", tc.expectPaths, paths)
			}
		})
	}
}
