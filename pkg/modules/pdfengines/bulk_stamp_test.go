package pdfengines

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestFormDataPdfBulkStamps(t *testing.T) {
	t.Run("parses valid entries", func(t *testing.T) {
		ctx := &api.ContextMock{Context: &api.Context{}}
		ctx.SetValues(map[string][]string{
			"stamps": {`[
				{"source":"text","expression":"CONFIDENTIAL","pages":"1","options":{"rot":"45"}},
				{"source":"image","file":"watermark.png","options":{"pos":"br"}}
			]`},
		})

		form := ctx.FormData()
		got := formDataPdfBulkStamps(form, true)
		if form.Validate() != nil {
			t.Fatalf("expected no validation error, got %v", form.Validate())
		}

		want := []bulkStampSpec{
			{
				Source:     gotenberg.StampSourceText,
				Expression: "CONFIDENTIAL",
				Pages:      "1",
				Options:    map[string]string{"rot": "45"},
			},
			{
				Source:  gotenberg.StampSourceImage,
				File:    "watermark.png",
				Options: map[string]string{"pos": "br"},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("bulk stamps = %#v, want %#v", got, want)
		}
	})

	t.Run("rejects missing expression for text source", func(t *testing.T) {
		ctx := &api.ContextMock{Context: &api.Context{}}
		ctx.SetValues(map[string][]string{
			"stamps": {`[{"source":"text"}]`},
		})

		form := ctx.FormData()
		_ = formDataPdfBulkStamps(form, true)
		err := form.Validate()
		if err == nil {
			t.Fatal("expected an error")
		}
		want := "form field 'stamps' is invalid (got '[{\"source\":\"text\"}]', resulting to entry 0 requires a non-empty expression for text source)"
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("rejects missing file for image source", func(t *testing.T) {
		ctx := &api.ContextMock{Context: &api.Context{}}
		ctx.SetValues(map[string][]string{
			"stamps": {`[{"source":"image"}]`},
		})

		form := ctx.FormData()
		_ = formDataPdfBulkStamps(form, true)
		err := form.Validate()
		if err == nil {
			t.Fatal("expected an error")
		}
		want := "form field 'stamps' is invalid (got '[{\"source\":\"image\"}]', resulting to entry 0 requires a non-empty file for image or pdf source)"
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	})
}

func TestResolveBulkStamps(t *testing.T) {
	t.Run("resolves uploaded stamp files", func(t *testing.T) {
		ctx := &api.Context{}
		stamps, err := resolveBulkStamps(ctx, []bulkStampSpec{
			{
				Source:     gotenberg.StampSourceText,
				Expression: "CONFIDENTIAL",
				Options:    map[string]string{"rot": "45"},
			},
			{
				Source:  gotenberg.StampSourceImage,
				File:    "watermark.png",
				Options: map[string]string{"pos": "br"},
			},
		}, []string{"/tmp/watermark.png"})
		if err != nil {
			t.Fatalf("resolve bulk stamps: %v", err)
		}

		want := []gotenberg.Stamp{
			{
				Source:     gotenberg.StampSourceText,
				Expression: "CONFIDENTIAL",
				Options:    map[string]string{"rot": "45"},
			},
			{
				Source:     gotenberg.StampSourceImage,
				Expression: "/tmp/watermark.png",
				Options:    map[string]string{"pos": "br"},
			},
		}
		if !reflect.DeepEqual(stamps, want) {
			t.Fatalf("resolved stamps = %#v, want %#v", stamps, want)
		}
	})

	t.Run("resolves multiple uploaded image stamp files by filename", func(t *testing.T) {
		ctx := &api.Context{}
		stamps, err := resolveBulkStamps(ctx, []bulkStampSpec{
			{
				Source:  gotenberg.StampSourceImage,
				File:    "watermark.png",
				Options: map[string]string{"pos": "br"},
			},
			{
				Source:  gotenberg.StampSourceImage,
				File:    "image.png",
				Options: map[string]string{"pos": "tl"},
			},
		}, []string{"/tmp/watermark.png", "/tmp/image.png"})
		if err != nil {
			t.Fatalf("resolve bulk stamps: %v", err)
		}

		want := []gotenberg.Stamp{
			{
				Source:     gotenberg.StampSourceImage,
				Expression: "/tmp/watermark.png",
				Options:    map[string]string{"pos": "br"},
			},
			{
				Source:     gotenberg.StampSourceImage,
				Expression: "/tmp/image.png",
				Options:    map[string]string{"pos": "tl"},
			},
		}
		if !reflect.DeepEqual(stamps, want) {
			t.Fatalf("resolved stamps = %#v, want %#v", stamps, want)
		}
	})

	t.Run("rejects unknown uploaded stamp file", func(t *testing.T) {
		ctx := &api.Context{}
		_, err := resolveBulkStamps(ctx, []bulkStampSpec{
			{Source: gotenberg.StampSourceImage, File: "missing.png"},
		}, nil)
		if err == nil {
			t.Fatal("expected an error")
		}

		var httpErr api.HttpError
		if !errors.As(err, &httpErr) {
			t.Fatalf("expected HTTP error, got %v", err)
		}
		status, message := httpErr.HttpError()
		if status != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
		}
		wantMessage := "Invalid form data: bulk stamp entry 0 references unknown uploaded stamp file 'missing.png'"
		if message != wantMessage {
			t.Fatalf("message = %q, want %q", message, wantMessage)
		}
	})

	t.Run("rejects duplicate uploaded stamp filenames", func(t *testing.T) {
		ctx := &api.Context{}
		_, err := resolveBulkStamps(ctx, []bulkStampSpec{
			{Source: gotenberg.StampSourceImage, File: "watermark.png"},
		}, []string{"/tmp/a/watermark.png", "/tmp/b/watermark.png"})
		if err == nil {
			t.Fatal("expected an error")
		}

		var httpErr api.HttpError
		if !errors.As(err, &httpErr) {
			t.Fatalf("expected HTTP error, got %v", err)
		}
		status, message := httpErr.HttpError()
		if status != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
		}
		wantMessage := "Invalid form data: duplicate stamp filename 'watermark.png'; use unique filenames for bulk stamps"
		if message != wantMessage {
			t.Fatalf("message = %q, want %q", message, wantMessage)
		}
	})
}

func TestBulkStampStub_AppliesStampsInOrder(t *testing.T) {
	ctx := &api.Context{}
	var calls []string
	engine := &gotenberg.PdfEngineMock{
		StampMock: func(_ context.Context, _ *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
			calls = append(calls, fmt.Sprintf("%s:%s", inputPath, stamp.Expression))
			return nil
		},
	}

	stamps := []gotenberg.Stamp{
		{Source: gotenberg.StampSourceText, Expression: "one"},
		{Source: gotenberg.StampSourceText, Expression: "two"},
	}
	inputPaths := []string{"a.pdf", "b.pdf"}

	err := BulkStampStub(ctx, engine, stamps, inputPaths)
	if err != nil {
		t.Fatalf("bulk stamp stub: %v", err)
	}

	want := []string{"a.pdf:one", "a.pdf:two", "b.pdf:one", "b.pdf:two"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}
