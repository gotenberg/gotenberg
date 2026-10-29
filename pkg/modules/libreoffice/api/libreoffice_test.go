package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeLokEnv makes the test binary act as lok's long-running mode, so the
// process code runs against the real pipe protocol without LibreOffice.
const fakeLokEnv = "GOTENBERG_TEST_FAKE_LOK"

func TestMain(m *testing.M) {
	if os.Getenv(fakeLokEnv) != "" {
		fakeLok()
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// fakeLok answers each request according to the input filename: "ok" writes
// the output, "badpass" and "unreadable" fail with lok's codes, "hang" never
// answers, and "crash" exits. It records its arguments next to its profile.
func fakeLok() {
	for i, arg := range os.Args {
		if arg == "--user-profile-path" && i+1 < len(os.Args) {
			_ = os.WriteFile(filepath.Join(os.Args[i+1], "args"), []byte(strings.Join(os.Args[1:], " ")), 0o600)
		}
	}

	in := bufio.NewReader(os.Stdin)
	out := json.NewEncoder(os.Stdout)

	for {
		line, err := in.ReadBytes('\n')
		if err != nil {
			return
		}

		var req lokRequest
		_ = json.Unmarshal(line, &req)

		name := strings.TrimSuffix(filepath.Base(req.InputPath), filepath.Ext(req.InputPath))
		if req.InputPath == "" {
			name = ""
		}

		switch name {
		case "":
			_ = out.Encode(lokResponse{Code: lokCodeInvalidRequest, Error: "inputPath and outputPath are required"})
		case "ok":
			_ = os.WriteFile(req.OutputPath, line, 0o600)
			_ = out.Encode(lokResponse{Success: true})
		case "badpass":
			_ = out.Encode(lokResponse{Code: lokCodeInvalidPassword, Error: "failed to load document: wrong document password"})
		case "unreadable":
			_ = out.Encode(lokResponse{Code: lokCodeLoadFailed, Error: "failed to load document"})
		case "hang":
			time.Sleep(time.Hour)
		case "crash":
			os.Exit(3)
		}
	}
}

func startFakeLok(t *testing.T) *libreOfficeProcess {
	t.Helper()
	t.Setenv(fakeLokEnv, "1")

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	p := newLibreOfficeProcess(libreOfficeArguments{
		binPath:      "/opt/libreoffice/program/soffice.bin",
		lokBinPath:   exe,
		startTimeout: 10 * time.Second,
	}).(*libreOfficeProcess)

	logger := slog.New(slog.DiscardHandler)

	err = p.Start(logger)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	t.Cleanup(func() { _ = p.Stop(logger) })

	return p
}

func TestLibreOfficeProcess(t *testing.T) {
	p := startFakeLok(t)
	logger := slog.New(slog.DiscardHandler)
	dir := t.TempDir()

	args, err := os.ReadFile(filepath.Join(p.userProfileDirPath, "args"))
	if err != nil {
		t.Fatalf("read the fake lok arguments: %v", err)
	}

	want := "--long-running --libreoffice-program-path /opt/libreoffice/program --user-profile-path " + p.userProfileDirPath
	if string(args) != want {
		t.Errorf("lok arguments = %q, want %q", args, want)
	}

	xcu, err := os.ReadFile(filepath.Join(p.userProfileDirPath, "user", "registrymodifications.xcu"))
	if err != nil || !strings.Contains(string(xcu), "BlockUntrustedRefererLinks") {
		t.Errorf("the profile was not seeded before lok started: %v", err)
	}

	output := filepath.Join(dir, "out.pdf")
	options := DefaultOptions()
	options.PdfFormats = PdfFormats{PdfA: PdfA2b}

	err = p.pdf(context.Background(), logger, filepath.Join(dir, "ok.csv"), output, options)
	if err != nil {
		t.Fatalf("pdf: %v", err)
	}

	var sent lokRequest
	data, err := os.ReadFile(output)
	if err != nil || json.Unmarshal(data, &sent) != nil {
		t.Fatalf("the fake lok wrote no request: %v", err)
	}

	if sent.PdfVersion != 2 || !sent.EmbedStandardFonts || !sent.DisableCalcHeader || !sent.UpdateIndexes {
		t.Errorf("request = %+v, want PDF/A-2b with embedded fonts, the CSV header off and indexes updated", sent)
	}

	for _, tc := range []struct {
		input string
		want  error
	}{
		{"badpass.docx", ErrInvalidPassword},
		{"unreadable.docx", ErrLoadFailed},
	} {
		err = p.pdf(context.Background(), logger, filepath.Join(dir, tc.input), output, DefaultOptions())
		if !errors.Is(err, tc.want) {
			t.Errorf("pdf(%s) = %v, want %v", tc.input, err, tc.want)
		}
	}

	if !p.Healthy(logger) {
		t.Fatal("failed conversions left lok unhealthy")
	}
}

func TestNewLokRequest_Paper(t *testing.T) {
	options := DefaultOptions()

	req, err := newLokRequest("in.docx", "out.pdf", options)
	if err != nil || req.PaperFormat != lokPaperFormatUnset {
		t.Fatalf("no paper size: %+v, %v, want the document's own size", req, err)
	}

	options.PaperWidth, options.PaperHeight = 210/25.4, 297/25.4

	req, err = newLokRequest("in.docx", "out.pdf", options)
	if err != nil || req.PaperFormat != lokPaperFormatUser || req.PaperWidth != 21000 || req.PaperHeight != 29700 {
		t.Fatalf("A4: %+v, %v, want a custom 21000x29700 size", req, err)
	}
}

func TestLibreOfficeProcess_InvalidPdfFormats(t *testing.T) {
	p := startFakeLok(t)

	options := DefaultOptions()
	options.PdfFormats = PdfFormats{PdfA: "PDF/A-1a"}

	err := p.pdf(context.Background(), slog.New(slog.DiscardHandler), "ok.docx", "out.pdf", options)
	if !errors.Is(err, ErrInvalidPdfFormats) {
		t.Fatalf("pdf = %v, want ErrInvalidPdfFormats", err)
	}
}

// TestLibreOfficeProcess_TimeoutKills pins that a cancelled conversion kills
// lok, so that its late answer can never reach the next request.
func TestLibreOfficeProcess_TimeoutKills(t *testing.T) {
	p := startFakeLok(t)
	logger := slog.New(slog.DiscardHandler)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := p.pdf(ctx, logger, "hang.docx", "out.pdf", DefaultOptions())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("pdf = %v, want DeadlineExceeded", err)
	}

	select {
	case <-p.exited:
	case <-time.After(5 * time.Second):
		t.Fatal("lok still runs after a cancelled conversion")
	}

	if p.Healthy(logger) {
		t.Fatal("a killed lok reports healthy")
	}
}

func TestLibreOfficeProcess_Crash(t *testing.T) {
	p := startFakeLok(t)
	logger := slog.New(slog.DiscardHandler)

	err := p.pdf(context.Background(), logger, "crash.docx", "out.pdf", DefaultOptions())
	if !errors.Is(err, ErrCrashed) {
		t.Fatalf("pdf = %v, want ErrCrashed", err)
	}

	if p.Healthy(logger) {
		t.Fatal("a crashed lok reports healthy")
	}
}
