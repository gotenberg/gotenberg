package qpdf

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"
)

func TestStripQpdfStringPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"unicode prefix", "u:factur-x.xml", "factur-x.xml"},
		{"binary prefix", "b:binary.bin", "binary.bin"},
		{"encoded prefix", "e:encoded.txt", "encoded.txt"},
		{"no prefix", "plain.xml", "plain.xml"},
		{"empty string", "", ""},
		{"prefix only", "u:", ""},
		{"colon in value", "u:file:name.xml", "file:name.xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripQpdfStringPrefix(tt.input)
			if got != tt.expected {
				t.Errorf("stripQpdfStringPrefix(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParsePdfObjects(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKeys  []string
		wantError bool
	}{
		{
			name:     "valid QPDF JSON v2",
			input:    `{"qpdf":[{"jsonversion":2},{"obj:1 0 R":{"value":{"/Type":"/Catalog"}}}]}`,
			wantKeys: []string{"obj:1 0 R"},
		},
		{
			name:      "invalid JSON",
			input:     `not json`,
			wantError: true,
		},
		{
			name:      "empty qpdf array",
			input:     `{"qpdf":[]}`,
			wantError: true,
		},
		{
			name:      "only header element",
			input:     `{"qpdf":[{"jsonversion":2}]}`,
			wantError: true,
		},
		{
			name:     "multiple objects",
			input:    `{"qpdf":[{},{"obj:1 0 R":{"value":{}},"obj:2 0 R":{"value":{}}}]}`,
			wantKeys: []string{"obj:1 0 R", "obj:2 0 R"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects, err := parsePdfObjects([]byte(tt.input))
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, key := range tt.wantKeys {
				if _, ok := objects[key]; !ok {
					t.Errorf("expected key %q in objects", key)
				}
			}
		})
	}
}

func TestPatchFilespecMetadata(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("sets AFRelationship on matching Filespec", func(t *testing.T) {
		objects := map[string]json.RawMessage{
			"obj:1 0 R": json.RawMessage(`{"value":{"/Type":"/Catalog"}}`),
			"obj:2 0 R": json.RawMessage(`{"value":{"/Type":"/Filespec","/UF":"u:factur-x.xml"}}`),
		}
		metadata := map[string]map[string]string{
			"factur-x.xml": {"relationship": "Data"},
		}

		catalogRef, _, filespecRefs, updateObjects := patchFilespecMetadata(logger, objects, metadata)

		if catalogRef != "obj:1 0 R" {
			t.Errorf("catalogRef = %q, want %q", catalogRef, "obj:1 0 R")
		}
		if len(filespecRefs) != 1 || filespecRefs[0] != "obj:2 0 R" {
			t.Errorf("filespecRefs = %v, want [obj:2 0 R]", filespecRefs)
		}
		updated, ok := updateObjects["obj:2 0 R"]
		if !ok {
			t.Fatal("expected obj:2 0 R in updateObjects")
		}
		value := updated.(map[string]any)["value"].(map[string]any)
		if value["/AFRelationship"] != "/Data" {
			t.Errorf("/AFRelationship = %v, want /Data", value["/AFRelationship"])
		}
	})

	t.Run("skips Filespec with no matching metadata", func(t *testing.T) {
		objects := map[string]json.RawMessage{
			"obj:1 0 R": json.RawMessage(`{"value":{"/Type":"/Filespec","/UF":"u:other.xml"}}`),
		}
		metadata := map[string]map[string]string{
			"factur-x.xml": {"relationship": "Data"},
		}

		_, _, filespecRefs, _ := patchFilespecMetadata(logger, objects, metadata)
		if len(filespecRefs) != 0 {
			t.Errorf("filespecRefs = %v, want empty", filespecRefs)
		}
	})

	t.Run("falls back to /F when /UF is absent", func(t *testing.T) {
		objects := map[string]json.RawMessage{
			"obj:1 0 R": json.RawMessage(`{"value":{"/Type":"/Filespec","/F":"u:factur-x.xml"}}`),
		}
		metadata := map[string]map[string]string{
			"factur-x.xml": {"relationship": "Alternative"},
		}

		_, _, filespecRefs, updateObjects := patchFilespecMetadata(logger, objects, metadata)
		if len(filespecRefs) != 1 {
			t.Fatalf("filespecRefs = %v, want 1 entry", filespecRefs)
		}
		value := updateObjects["obj:1 0 R"].(map[string]any)["value"].(map[string]any)
		if value["/AFRelationship"] != "/Alternative" {
			t.Errorf("/AFRelationship = %v, want /Alternative", value["/AFRelationship"])
		}
	})

	t.Run("sets stream Subtype via EF reference", func(t *testing.T) {
		objects := map[string]json.RawMessage{
			"obj:2 0 R": json.RawMessage(`{"value":{"/Type":"/Filespec","/UF":"u:factur-x.xml","/EF":{"/F":"3 0 R"}}}`),
			"obj:3 0 R": json.RawMessage(`{"stream":{"dict":{"/Type":"/EmbeddedFile"}}}`),
		}
		metadata := map[string]map[string]string{
			"factur-x.xml": {"mimeType": "text/xml"},
		}

		_, _, _, updateObjects := patchFilespecMetadata(logger, objects, metadata)
		streamObj, ok := updateObjects["obj:3 0 R"]
		if !ok {
			t.Fatal("expected obj:3 0 R in updateObjects")
		}
		stream := streamObj.(map[string]any)["stream"].(map[string]any)
		dict := stream["dict"].(map[string]any)
		if dict["/Subtype"] != "/text/xml" {
			t.Errorf("/Subtype = %v, want /text/xml", dict["/Subtype"])
		}
	})
}

func TestPatchCatalogAF(t *testing.T) {
	t.Run("adds filespec refs to AF array", func(t *testing.T) {
		catalogValue := map[string]any{"/Type": "/Catalog"}
		updateObjects := make(map[string]any)

		patchCatalogAF("obj:1 0 R", catalogValue, []string{"obj:2 0 R", "obj:3 0 R"}, updateObjects)

		af, ok := catalogValue["/AF"].([]any)
		if !ok {
			t.Fatal("expected /AF to be []any")
		}
		if len(af) != 2 {
			t.Fatalf("/AF has %d entries, want 2", len(af))
		}
		if af[0] != "2 0 R" || af[1] != "3 0 R" {
			t.Errorf("/AF = %v, want [2 0 R, 3 0 R]", af)
		}
	})

	t.Run("does not duplicate existing refs", func(t *testing.T) {
		catalogValue := map[string]any{
			"/Type": "/Catalog",
			"/AF":   []any{"2 0 R"},
		}
		updateObjects := make(map[string]any)

		patchCatalogAF("obj:1 0 R", catalogValue, []string{"obj:2 0 R", "obj:3 0 R"}, updateObjects)

		af := catalogValue["/AF"].([]any)
		if len(af) != 2 {
			t.Fatalf("/AF has %d entries, want 2", len(af))
		}
	})

	t.Run("no-op when catalogRef is empty", func(t *testing.T) {
		updateObjects := make(map[string]any)
		patchCatalogAF("", nil, []string{"obj:2 0 R"}, updateObjects)
		if len(updateObjects) != 0 {
			t.Error("expected no updates for empty catalogRef")
		}
	})
}

func TestSetStreamSubtype(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("sets Subtype in stream dict", func(t *testing.T) {
		objects := map[string]json.RawMessage{
			"obj:3 0 R": json.RawMessage(`{"stream":{"dict":{"/Type":"/EmbeddedFile"}}}`),
		}
		updateObjects := make(map[string]any)

		setStreamSubtype(logger, objects, updateObjects, "obj:3 0 R", "text/xml")

		streamObj := updateObjects["obj:3 0 R"].(map[string]any)["stream"].(map[string]any)
		dict := streamObj["dict"].(map[string]any)
		if dict["/Subtype"] != "/text/xml" {
			t.Errorf("/Subtype = %v, want /text/xml", dict["/Subtype"])
		}
	})

	t.Run("auto-adds obj: prefix to ref", func(t *testing.T) {
		objects := map[string]json.RawMessage{
			"obj:5 0 R": json.RawMessage(`{"stream":{"dict":{}}}`),
		}
		updateObjects := make(map[string]any)

		setStreamSubtype(logger, objects, updateObjects, "5 0 R", "application/pdf")

		if _, ok := updateObjects["obj:5 0 R"]; !ok {
			t.Error("expected obj:5 0 R in updateObjects")
		}
	})

	t.Run("warns on missing object", func(t *testing.T) {
		objects := map[string]json.RawMessage{}
		updateObjects := make(map[string]any)

		setStreamSubtype(logger, objects, updateObjects, "obj:99 0 R", "text/xml")

		if len(updateObjects) != 0 {
			t.Error("expected no updates for missing object")
		}
	})

	t.Run("warns on object without stream key", func(t *testing.T) {
		objects := map[string]json.RawMessage{
			"obj:3 0 R": json.RawMessage(`{"value":{"/Type":"/Page"}}`),
		}
		updateObjects := make(map[string]any)

		setStreamSubtype(logger, objects, updateObjects, "obj:3 0 R", "text/xml")

		if len(updateObjects) != 0 {
			t.Error("expected no updates for non-stream object")
		}
	})
}
