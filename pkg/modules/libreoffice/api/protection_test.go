package api

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes content to a new file named name inside dir and returns its
// path.
func writeFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(dir, name)
	err := os.WriteFile(path, content, 0o600)
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	return path
}

// writeZip builds a ZIP archive from entries and returns its path.
func writeZip(t *testing.T, dir, name string, entries map[string]string) string {
	t.Helper()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for entryName, content := range entries {
		f, err := w.Create(entryName)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", entryName, err)
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			t.Fatalf("write zip entry %s: %v", entryName, err)
		}
	}

	err := w.Close()
	if err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return writeFile(t, dir, name, buf.Bytes())
}

func TestDetectPasswordProtection(t *testing.T) {
	dir := t.TempDir()

	ole2 := func(name string) string {
		return writeFile(t, dir, name, append(ole2Magic, bytes.Repeat([]byte{0x00}, 64)...))
	}

	for _, tc := range []struct {
		name string
		path string
		want PasswordProtection
	}{
		{
			name: "encrypted OOXML is a compound file",
			path: ole2("encrypted.docx"),
			want: PasswordProtectionRequired,
		},
		{
			name: "extension casing is ignored",
			path: ole2("encrypted.DOCX"),
			want: PasswordProtectionRequired,
		},
		{
			name: "encrypted spreadsheet",
			path: ole2("encrypted.xlsx"),
			want: PasswordProtectionRequired,
		},
		{
			name: "legacy binary document is inconclusive",
			path: ole2("legacy.doc"),
			want: PasswordProtectionUnknown,
		},
		{
			name: "plain OOXML package",
			path: writeZip(t, dir, "plain.docx", map[string]string{
				"[Content_Types].xml": "<Types/>",
				"word/document.xml":   "<w:document/>",
			}),
			want: PasswordProtectionNone,
		},
		{
			name: "encrypted ODF declares encryption-data in its manifest",
			path: writeZip(t, dir, "encrypted.odt", map[string]string{
				"mimetype":              "application/vnd.oasis.opendocument.text",
				"META-INF/manifest.xml": `<manifest:manifest><manifest:file-entry><manifest:encryption-data manifest:checksum="x"/></manifest:file-entry></manifest:manifest>`,
				"content.xml":           "<office:document-content/>",
			}),
			want: PasswordProtectionRequired,
		},
		{
			name: "plain ODF has a manifest without encryption-data",
			path: writeZip(t, dir, "plain.odt", map[string]string{
				"mimetype":              "application/vnd.oasis.opendocument.text",
				"META-INF/manifest.xml": `<manifest:manifest><manifest:file-entry manifest:full-path="/"/></manifest:manifest>`,
				"content.xml":           "<office:document-content/>",
			}),
			want: PasswordProtectionNone,
		},
		{
			name: "flat XML carries no encryption",
			path: writeFile(t, dir, "flat.fodt", []byte("<?xml version=\"1.0\"?><office:document/>")),
			want: PasswordProtectionUnknown,
		},
		{
			name: "plain text",
			path: writeFile(t, dir, "notes.txt", []byte("hello")),
			want: PasswordProtectionUnknown,
		},
		{
			name: "file shorter than any magic",
			path: writeFile(t, dir, "tiny.docx", []byte{0x50}),
			want: PasswordProtectionUnknown,
		},
		{
			name: "empty file",
			path: writeFile(t, dir, "empty.docx", nil),
			want: PasswordProtectionUnknown,
		},
		{
			name: "truncated archive",
			path: writeFile(t, dir, "truncated.docx", append(zipMagic, bytes.Repeat([]byte{0x00}, 32)...)),
			want: PasswordProtectionUnknown,
		},
		{
			name: "non-existent path",
			path: filepath.Join(dir, "does-not-exist.docx"),
			want: PasswordProtectionUnknown,
		},
		{
			name: "directory",
			path: dir,
			want: PasswordProtectionUnknown,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectPasswordProtection(tc.path); got != tc.want {
				t.Errorf("DetectPasswordProtection(%s) = %d, want %d", tc.path, got, tc.want)
			}
		})
	}
}

// TestDetectPasswordProtection_Fixtures anchors detection to the same documents
// the integration scenarios upload, so a fixture swap cannot silently flip a
// status code.
func TestDetectPasswordProtection_Fixtures(t *testing.T) {
	for _, tc := range []struct {
		path string
		want PasswordProtection
	}{
		{"../../../../test/integration/testdata/protected_page_1.docx", PasswordProtectionRequired},
		{"../../../../test/integration/testdata/page_1.docx", PasswordProtectionNone},
	} {
		t.Run(filepath.Base(tc.path), func(t *testing.T) {
			if _, err := os.Stat(tc.path); err != nil {
				t.Skipf("fixture unavailable: %v", err)
			}
			if got := DetectPasswordProtection(tc.path); got != tc.want {
				t.Errorf("DetectPasswordProtection(%s) = %d, want %d", tc.path, got, tc.want)
			}
		})
	}
}

// TestDetectPasswordProtection_OversizedManifest verifies that a manifest far
// larger than the cap still yields a verdict through a bounded read.
func TestDetectPasswordProtection_OversizedManifest(t *testing.T) {
	dir := t.TempDir()

	// Well past odfManifestSizeLimit, and highly compressible, so the archive
	// on disk stays small.
	filler := strings.Repeat("<manifest:file-entry manifest:full-path=\"pad\"/>", 200_000)

	path := writeZip(t, dir, "oversized.odt", map[string]string{
		"mimetype":              "application/vnd.oasis.opendocument.text",
		"META-INF/manifest.xml": "<manifest:manifest>" + filler + "</manifest:manifest>",
	})

	if got := DetectPasswordProtection(path); got != PasswordProtectionNone {
		t.Errorf("DetectPasswordProtection(oversized) = %d, want %d", got, PasswordProtectionNone)
	}
}
