package gotenberg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildVersion(t *testing.T) {
	for _, tc := range []struct {
		scenario  string
		fileBody  string
		writeFile bool
		setEnv    bool
		moduleID  string
		wantValue string
		wantOk    bool
	}{
		{
			scenario:  "version present",
			fileBody:  "Chromium 146.0",
			writeFile: true,
			setEnv:    true,
			moduleID:  "chromium",
			wantValue: "Chromium 146.0",
			wantOk:    true,
		},
		{
			scenario:  "value trimmed",
			fileBody:  "  qpdf version 11.9.0  \n",
			writeFile: true,
			setEnv:    true,
			moduleID:  "qpdf",
			wantValue: "qpdf version 11.9.0",
			wantOk:    true,
		},
		{
			scenario:  "empty file falls back",
			fileBody:  "   \n",
			writeFile: true,
			setEnv:    true,
			moduleID:  "pdftk",
			wantValue: "",
			wantOk:    false,
		},
		{
			scenario:  "missing file falls back",
			writeFile: false,
			setEnv:    true,
			moduleID:  "exiftool",
			wantValue: "",
			wantOk:    false,
		},
		{
			scenario:  "env unset falls back",
			setEnv:    false,
			moduleID:  "chromium",
			wantValue: "",
			wantOk:    false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.setEnv {
				dir := t.TempDir()
				if tc.writeFile {
					if err := os.WriteFile(filepath.Join(dir, tc.moduleID), []byte(tc.fileBody), 0o600); err != nil {
						t.Fatalf("write version file: %v", err)
					}
				}
				t.Setenv(BuildVersionsDirPathEnvVar, dir)
			} else {
				t.Setenv(BuildVersionsDirPathEnvVar, "")
			}

			value, ok := BuildVersion(tc.moduleID)
			if value != tc.wantValue || ok != tc.wantOk {
				t.Errorf("BuildVersion(%q) = (%q, %t), want (%q, %t)", tc.moduleID, value, ok, tc.wantValue, tc.wantOk)
			}
		})
	}
}
