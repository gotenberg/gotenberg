package gotenberg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg/internal/log"
	internalotel "github.com/gotenberg/gotenberg/v9/pkg/gotenberg/internal/otel"
)

// bannerArt is "Gotenberg" in the Small Slant figlet font. See
// https://patorjk.com/software/taag/#p=display&f=Small%20Slant&t=Gotenberg.
var bannerArt = []string{
	`  _____     __           __`,
	` / ___/__  / /____ ___  / /  ___ _______ _`,
	`/ (_ / _ \/ __/ -_) _ \/ _ \/ -_) __/ _ ` + "`" + `/`,
	`\___/\___/\__/\__/_//_/_.__/\__/_/  \_, /`,
	`                                   /___/`,
}

const tagline = "Convert documents to PDF, then edit, sign, validate, and render them."

// Startup describes a started Gotenberg process. [Startup.Write] renders it
// for a terminal, [Startup.Log] as a log record.
type Startup struct {
	version  string
	duration time.Duration
	reports  []moduleReport
}

type moduleReport struct {
	id string
	Report
}

// NewStartup gathers the reports of the modules of ctx that implement
// [Reporter], in module ID order, followed by the telemetry's. Call it once
// every [App] has started, duration after the process did.
func NewStartup(ctx *Context, version string, duration time.Duration) (Startup, error) {
	reporters, err := ctx.Modules(new(Reporter))
	if err != nil {
		return Startup{}, fmt.Errorf("get reporters: %w", err)
	}

	s := Startup{version: version, duration: duration}
	for _, r := range reporters {
		s.reports = append(s.reports, moduleReport{
			id:     r.(Module).Descriptor().ID,
			Report: r.(Reporter).Report(),
		})
	}

	s.reports = append(s.reports, moduleReport{id: "telemetry", Report: telemetryReport()})

	return s, nil
}

// telemetryReport names the exporter of each OpenTelemetry signal.
func telemetryReport() Report {
	traces, metrics, logs := internalotel.Exporters()

	field := func(signal, exporter string) ReportField {
		text := exporter
		// "none" is the OpenTelemetry name for no exporter.
		if exporter == "" || exporter == "none" {
			text = "off"
		}

		return ReportField{Key: signal + "_exporter", Value: exporter, Text: signal + " " + text}
	}

	return Report{
		Name: "Telemetry",
		Fields: []ReportField{
			field("traces", traces),
			field("metrics", metrics),
			field("logs", logs),
		},
	}
}

// Write renders the startup summary for a terminal: the banner when banner is
// true, then one line per report, their names padded to a common width so
// that the values align. color adds ANSI styles.
func (s Startup) Write(w io.Writer, banner, color bool) error {
	bold, dim := style(color, "1"), style(color, "2")

	var b strings.Builder
	b.WriteString("\n")

	if banner {
		last := len(bannerArt) - 1
		for _, line := range bannerArt[:last] {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("  " + bannerArt[last] + "   " + bold(s.version) + "\n\n")
		b.WriteString("  " + dim(tagline) + "\n\n")
	} else {
		b.WriteString("  " + bold("Gotenberg "+s.version) + "\n\n")
	}

	width := 0
	for _, r := range s.reports {
		width = max(width, utf8.RuneCountInString(r.Name))
	}

	separator := dim(" · ")
	for _, r := range s.reports {
		texts := make([]string, 0, len(r.Fields))
		for _, f := range r.Fields {
			texts = append(texts, f.text())
		}

		// Pad before styling: the escape sequences take no columns.
		name := r.Name + strings.Repeat(" ", width-utf8.RuneCountInString(r.Name))
		b.WriteString("  " + bold(name) + "  " + strings.Join(texts, separator) + "\n")
	}

	b.WriteString("\n  " + dim(fmt.Sprintf("Ready in %s. Press Ctrl+C to stop.", s.duration.Round(time.Millisecond))) + "\n\n")

	_, err := io.WriteString(w, b.String())
	if err != nil {
		return fmt.Errorf("write startup summary: %w", err)
	}

	return nil
}

// Log records the startup summary as one log record, each report a group
// named after its module ID.
func (s Startup) Log(ctx context.Context, logger *slog.Logger) {
	attrs := make([]any, 0, 2+len(s.reports))
	attrs = append(attrs,
		slog.String("version", s.version),
		slog.String("startup_duration", s.duration.Round(time.Millisecond).String()),
	)

	for _, r := range s.reports {
		fields := make([]any, 0, len(r.Fields))
		for _, f := range r.Fields {
			fields = append(fields, slog.Any(f.Key, f.Value))
		}
		attrs = append(attrs, slog.Group(r.id, fields...))
	}

	logger.InfoContext(ctx, "Gotenberg started", attrs...)
}

func (f ReportField) text() string {
	if f.Text != "" {
		return f.Text
	}

	return fmt.Sprint(f.Value)
}

// style returns a function wrapping a string in the ANSI SGR code, or leaving
// it as is when color is false.
func style(color bool, code string) func(string) string {
	return func(s string) string {
		if !color || s == "" {
			return s
		}

		return "\x1b[" + code + "m" + s + "\x1b[0m"
	}
}

// ProcessLogger returns the global logger for the process itself, as opposed
// to a module's (see [Logger]).
func ProcessLogger() *slog.Logger {
	return log.Logger().With(slog.String("logger", "gotenberg"))
}
