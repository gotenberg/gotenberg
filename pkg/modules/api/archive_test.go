package api

import (
	"archive/zip"
	"bytes"
	"io"
	"log/slog"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// zipEntry is an entry of a test archive. A name ending with a slash is a
// directory.
type zipEntry struct {
	name    string
	content []byte
}

func buildZip(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for _, e := range entries {
		f, err := w.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.Write(e.content)
		if err != nil {
			t.Fatal(err)
		}
	}

	err := w.Close()
	if err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

// newArchiveContext sends archive under the given form field, next to a
// plain file, and returns the resulting context.
func newArchiveContext(t *testing.T, field, filename string, archive []byte, cfg archiveConfig, bodyLimit int64) (*Context, error) {
	t.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write(archive)
	if err != nil {
		t.Fatal(err)
	}

	part, err = writer.CreateFormFile("files", "plain.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.WriteString(part, "plain")
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/forms/chromium/convert/html", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))

	ctx, cancel, err := newContext(echo.New().NewContext(req, httptest.NewRecorder()), slog.New(slog.DiscardHandler), fs, 10*time.Second, bodyLimit, downloadFromConfig{disable: true}, cfg)
	t.Cleanup(cancel)

	return ctx, err
}

func TestNewContext_Archives(t *testing.T) {
	cfg := archiveConfig{maxEntries: 10, maxRatio: 100}

	t.Run("unpacks the entries next to the other files", func(t *testing.T) {
		archive := buildZip(t,
			zipEntry{name: "index.html", content: []byte("<html></html>")},
			zipEntry{name: "assets/", content: nil},
			zipEntry{name: "assets/style.css", content: []byte("body{}")},
			zipEntry{name: "nested.zip", content: buildZip(t, zipEntry{name: "inner.txt", content: []byte("inner")})},
			zipEntry{name: "__MACOSX/._index.html", content: []byte("metadata")},
			zipEntry{name: "._style.css", content: []byte("metadata")},
		)

		ctx, err := newArchiveContext(t, ArchivesFormField, "site.zip", archive, cfg, 0)
		if err != nil {
			t.Fatalf("newContext: %v", err)
		}

		got := slices.Sorted(maps.Keys(ctx.files))
		want := []string{"index.html", "nested.zip", "plain.txt", "style.css"}
		if !slices.Equal(got, want) {
			t.Fatalf("files = %v, want %v", got, want)
		}

		data, err := os.ReadFile(ctx.files["style.css"])
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "body{}" {
			t.Fatalf("style.css = %q, want its content", data)
		}
	})

	t.Run("a zip under another field stays a file", func(t *testing.T) {
		archive := buildZip(t, zipEntry{name: "index.html", content: []byte("<html></html>")})

		ctx, err := newArchiveContext(t, "files", "site.zip", archive, cfg, 0)
		if err != nil {
			t.Fatalf("newContext: %v", err)
		}

		got := slices.Sorted(maps.Keys(ctx.files))
		if !slices.Equal(got, []string{"plain.txt", "site.zip"}) {
			t.Fatalf("files = %v, want the archive as a file", got)
		}
	})

	for _, tc := range []struct {
		name      string
		archive   func(t *testing.T) []byte
		bodyLimit int64
		want      string
	}{
		{
			name:    "not a zip archive",
			archive: func(*testing.T) []byte { return []byte("not a zip") },
			want:    "'site.zip' is not a zip archive",
		},
		{
			name: "too many entries",
			archive: func(t *testing.T) []byte {
				entries := make([]zipEntry, 11)
				for i := range entries {
					entries[i] = zipEntry{name: strings.Repeat("a", i+1) + ".txt"}
				}
				return buildZip(t, entries...)
			},
			want: "holds more than 10 entries",
		},
		{
			name: "zip bomb",
			archive: func(t *testing.T) []byte {
				// Zeros deflate about a thousand to one.
				return buildZip(t, zipEntry{name: "zeros.txt", content: make([]byte, 4<<20)})
			},
			want: "unpacks to more than 100 times its size",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newArchiveContext(t, ArchivesFormField, "site.zip", tc.archive(t), cfg, tc.bodyLimit)

			status, message := ParseError(err)
			if status != http.StatusBadRequest || !strings.Contains(message, tc.want) {
				t.Fatalf("got %d %q, want a 400 containing %q", status, message, tc.want)
			}
		})
	}

	t.Run("the unpacked bytes count against the body limit", func(t *testing.T) {
		archive := buildZip(t, zipEntry{name: "big.txt", content: bytes.Repeat([]byte("abcdefgh"), 4096)})

		_, err := newArchiveContext(t, ArchivesFormField, "site.zip", archive, archiveConfig{maxEntries: 10, maxRatio: 1000}, int64(len(archive))+2048)

		status, _ := ParseError(err)
		if status != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", status)
		}
	})
}
