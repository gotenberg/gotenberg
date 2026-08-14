package pdfengines

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestFormDataPdfStamps(t *testing.T) {
	for _, tc := range []struct {
		scenario   string
		values     map[string][]string
		expect     []gotenberg.Stamp
		expectErr  bool
		expectCode int
	}{
		{
			scenario: "single text stamp (backward compatible)",
			values: map[string][]string{
				"stampSource":     {"text"},
				"stampExpression": {"CONFIDENTIAL"},
				"stampOptions":    {`{"rot":"45"}`},
			},
			expect: []gotenberg.Stamp{
				{Source: "text", Expression: "CONFIDENTIAL", Options: map[string]string{"rot": "45"}},
			},
		},
		{
			scenario: "multiple stamps aligned by position",
			values: map[string][]string{
				"stampSource":     {"text", "image"},
				"stampExpression": {"ONE"},
				"stampPages":      {"1-2", "3"},
				"stampOptions":    {`{"pos":"tl"}`, `{"pos":"br"}`},
			},
			expect: []gotenberg.Stamp{
				{Source: "text", Expression: "ONE", Pages: "1-2", Options: map[string]string{"pos": "tl"}},
				{Source: "image", Expression: "", Pages: "3", Options: map[string]string{"pos": "br"}},
			},
		},
		{
			scenario: "no stamp fields",
			values:   map[string][]string{},
			expect:   []gotenberg.Stamp{},
		},
		{
			scenario:   "invalid source",
			values:     map[string][]string{"stampSource": {"text", "foo"}},
			expectErr:  true,
			expectCode: http.StatusBadRequest,
		},
		{
			scenario: "invalid options JSON",
			values: map[string][]string{
				"stampSource":  {"text"},
				"stampOptions": {"{"},
			},
			expectErr:  true,
			expectCode: http.StatusBadRequest,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ctx := &api.ContextMock{Context: &api.Context{}}
			ctx.SetValues(tc.values)
			form := ctx.FormData()

			got, err := FormDataPdfStamps(form)

			if tc.expectErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				var httpErr api.HttpError
				if !errors.As(err, &httpErr) {
					t.Fatalf("expected an api.HttpError, got %T", err)
				}
				if status, _ := httpErr.HttpError(); status != tc.expectCode {
					t.Fatalf("status = %d, want %d", status, tc.expectCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !reflect.DeepEqual(got, tc.expect) {
				t.Fatalf("stamps = %#v, want %#v", got, tc.expect)
			}
		})
	}
}

func TestBindStampFiles(t *testing.T) {
	for _, tc := range []struct {
		scenario   string
		stamps     []gotenberg.Stamp
		files      []string
		expect     []gotenberg.Stamp
		expectErr  bool
		expectCode int
	}{
		{
			scenario: "text stamps consume no files",
			stamps:   []gotenberg.Stamp{{Source: "text", Expression: "FOO"}},
			expect:   []gotenberg.Stamp{{Source: "text", Expression: "FOO"}},
		},
		{
			scenario: "image and pdf stamps consume files in order, overwriting expression",
			stamps: []gotenberg.Stamp{
				{Source: "image", Expression: "ignored"},
				{Source: "text", Expression: "MIDDLE"},
				{Source: "pdf"},
			},
			files: []string{"/a.png", "/b.pdf"},
			expect: []gotenberg.Stamp{
				{Source: "image", Expression: "/a.png"},
				{Source: "text", Expression: "MIDDLE"},
				{Source: "pdf", Expression: "/b.pdf"},
			},
		},
		{
			scenario:   "not enough files for the image or pdf stamps",
			stamps:     []gotenberg.Stamp{{Source: "image"}, {Source: "image"}},
			files:      []string{"/a.png"},
			expectErr:  true,
			expectCode: http.StatusBadRequest,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			err := BindStampFiles(tc.stamps, tc.files)

			if tc.expectErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				var httpErr api.HttpError
				if !errors.As(err, &httpErr) {
					t.Fatalf("expected an api.HttpError, got %T", err)
				}
				if status, _ := httpErr.HttpError(); status != tc.expectCode {
					t.Fatalf("status = %d, want %d", status, tc.expectCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !reflect.DeepEqual(tc.stamps, tc.expect) {
				t.Fatalf("stamps = %#v, want %#v", tc.stamps, tc.expect)
			}
		})
	}
}
