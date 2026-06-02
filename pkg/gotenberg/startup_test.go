package gotenberg

import (
	"context"
	"testing"
)

func TestDebugModuleVersion(t *testing.T) {
	info := DebugInfo{
		ModulesAdditionalData: map[string]map[string]any{
			"chromium": {"version": "Chromium 145.0"},
			"broken":   {"version": 42},
		},
	}

	if got := debugModuleVersion(info, "chromium"); got != "Chromium 145.0" {
		t.Errorf("expected chromium version, got %q", got)
	}
	if got := debugModuleVersion(info, "missing"); got != "" {
		t.Errorf("expected empty for a missing module, got %q", got)
	}
	if got := debugModuleVersion(info, "broken"); got != "" {
		t.Errorf("expected empty for a non-string version, got %q", got)
	}
}

func TestEmitStartupSpan(t *testing.T) {
	recorder := newTestSpanRecorder(t)

	debugMu.Lock()
	previous := debug
	debug = &DebugInfo{
		Version: "v8.0.0",
		ModulesAdditionalData: map[string]map[string]any{
			"chromium":        {"version": "Chromium 145.0"},
			"libreoffice-api": {"version": "LibreOffice 24.8"},
		},
	}
	debugMu.Unlock()
	t.Cleanup(func() {
		debugMu.Lock()
		debug = previous
		debugMu.Unlock()
	})

	EmitStartupSpan(context.Background())

	span := findSpan(recorder, "gotenberg.startup")
	if span == nil {
		t.Fatal("expected a gotenberg.startup span to be recorded")
	}

	if v, ok := spanAttr(span, "gotenberg.chromium.version"); !ok || v.AsString() != "Chromium 145.0" {
		t.Errorf("expected gotenberg.chromium.version=Chromium 145.0, got %q (present=%t)", v.AsString(), ok)
	}
	if v, ok := spanAttr(span, "gotenberg.libreoffice.version"); !ok || v.AsString() != "LibreOffice 24.8" {
		t.Errorf("expected gotenberg.libreoffice.version=LibreOffice 24.8, got %q (present=%t)", v.AsString(), ok)
	}
}
