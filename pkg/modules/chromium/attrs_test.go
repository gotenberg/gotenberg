package chromium

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestConversionInputAttrs(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "index.html")
	content := []byte("<html></html>")
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	t.Run("file URL with non-api context", func(t *testing.T) {
		got := map[string]int64{}
		for _, kv := range conversionInputAttrs(context.Background(), "file://"+tmp) {
			got[string(kv.Key)] = kv.Value.AsInt64()
		}

		if _, ok := got["gotenberg.conversion.input.files.count"]; ok {
			t.Error("did not expect files.count for a non-api context")
		}
		if got["gotenberg.conversion.input.html.bytes"] != int64(len(content)) {
			t.Errorf("expected html.bytes=%d, got %d", len(content), got["gotenberg.conversion.input.html.bytes"])
		}
	})

	t.Run("remote URL yields no html.bytes", func(t *testing.T) {
		for _, kv := range conversionInputAttrs(context.Background(), "https://example.com") {
			if string(kv.Key) == "gotenberg.conversion.input.html.bytes" {
				t.Error("did not expect html.bytes for a remote URL")
			}
		}
	})

	t.Run("api context yields files.count", func(t *testing.T) {
		var found bool
		for _, kv := range conversionInputAttrs(&api.Context{}, "https://example.com") {
			if string(kv.Key) == "gotenberg.conversion.input.files.count" {
				found = true
				if kv.Value.AsInt64() != 0 {
					t.Errorf("expected files.count=0, got %d", kv.Value.AsInt64())
				}
			}
		}
		if !found {
			t.Error("expected files.count for an api context")
		}
	})
}
