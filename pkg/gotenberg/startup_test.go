package gotenberg

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func testStartup() Startup {
	return Startup{
		version:  "9.0.0",
		duration: 1834 * time.Millisecond,
		reports: []moduleReport{
			{id: "api", Name: "API", Fields: []ReportField{{Key: "address", Value: "http://[::]:3000/"}}},
			{id: "chromium", Name: "Chromium", Fields: []ReportField{
				{Key: "version", Value: "Chromium 148.0.7778.96", Text: "148.0.7778.96"},
				{Key: "max_concurrency", Value: 6, Text: "max concurrency 6"},
			}},
			{id: "libreoffice-api", Name: "LibreOffice", Fields: []ReportField{{Key: "start", Value: "auto", Text: "started"}}},
		},
	}
}

func TestStartup_Write(t *testing.T) {
	for _, tc := range []struct {
		scenario string
		banner   bool
		color    bool
	}{
		{scenario: "plain", banner: true},
		{scenario: "colored", banner: true, color: true},
		{scenario: "without banner"},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var buf bytes.Buffer
			err := testStartup().Write(&buf, tc.banner, tc.color)
			if err != nil {
				t.Fatalf("write: %v", err)
			}

			out := stripANSI(buf.String())
			if !tc.color && out != buf.String() {
				t.Fatal("got ANSI escape sequences without color")
			}
			if tc.color && out == buf.String() {
				t.Fatal("got no ANSI escape sequences with color")
			}

			// Each value starts in the column after the longest name.
			column := len("  LibreOffice  ")
			for name, value := range map[string]string{
				"API":         "http://[::]:3000/",
				"Chromium":    "148.0.7778.96 · max concurrency 6",
				"LibreOffice": "started",
			} {
				line := findLine(t, out, "  "+name+" ")
				if line[column:] != value || strings.TrimSpace(line[:column]) != name {
					t.Errorf("line %q: want %q from column %d", line, value, column)
				}
			}

			if tc.banner != strings.Contains(out, bannerArt[1]) {
				t.Errorf("banner shown = %t, want %t", !tc.banner, tc.banner)
			}
			if !tc.banner {
				findLine(t, out, "  Gotenberg 9.0.0")
			}
			findLine(t, out, "  Ready in 1.834s. Press Ctrl+C to stop.")
		})
	}
}

func TestStartup_Log(t *testing.T) {
	var buf bytes.Buffer
	testStartup().Log(context.Background(), slog.New(slog.NewJSONHandler(&buf, nil)))

	var record struct {
		Msg      string `json:"msg"`
		Version  string `json:"version"`
		Duration string `json:"startup_duration"`
		Chromium struct {
			Version        string `json:"version"`
			MaxConcurrency int    `json:"max_concurrency"`
		} `json:"chromium"`
		LibreOffice struct {
			Start string `json:"start"`
		} `json:"libreoffice-api"`
	}
	err := json.Unmarshal(buf.Bytes(), &record)
	if err != nil {
		t.Fatalf("unmarshal %q: %v", buf.String(), err)
	}

	if record.Msg != "Gotenberg started" || record.Version != "9.0.0" || record.Duration != "1.834s" {
		t.Errorf("record = %+v", record)
	}
	if record.Chromium.Version != "Chromium 148.0.7778.96" || record.Chromium.MaxConcurrency != 6 || record.LibreOffice.Start != "auto" {
		t.Errorf("record = %+v, want the machine values, not the text", record)
	}
}

func TestVersionNumber(t *testing.T) {
	for in, want := range map[string]string{
		"Chromium 148.0.7778.96 built on Debian": "148.0.7778.96",
		"LibreOffice 26.2.1.2 abc123":            "26.2.1.2",
		"exec: not found":                        "exec: not found",
		"":                                       "",
	} {
		got := VersionNumber(in)
		if got != want {
			t.Errorf("VersionNumber(%q) = %q, want %q", in, got, want)
		}
	}
}

func findLine(t *testing.T, out, prefix string) string {
	t.Helper()

	for line := range strings.Lines(out) {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSuffix(line, "\n")
		}
	}

	t.Fatalf("no line starts with %q in:\n%s", prefix, out)

	return ""
}

func stripANSI(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "\x1b[")
		if i < 0 {
			return b.String() + s
		}
		b.WriteString(s[:i])
		j := strings.IndexByte(s[i:], 'm')
		s = s[i+j+1:]
	}
}
