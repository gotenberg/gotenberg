package pdf

import (
	"fmt"
	"net/http"
	"testing"

	litho "github.com/gotenberg/litho/pkg/pdf"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

func TestHandleError_Status(t *testing.T) {
	for _, tc := range []struct {
		err    error
		expect int
	}{
		{err: litho.ErrNotAPDF, expect: http.StatusBadRequest},
		{err: litho.ErrInvalidPassword, expect: http.StatusBadRequest},
		// Answered 500 while the list lived in Gotenberg.
		{err: litho.ErrInvalidRenderOptions, expect: http.StatusBadRequest},
		{err: litho.ErrNoInvoiceAttachment, expect: http.StatusBadRequest},
		{err: litho.ErrInternalPanic, expect: http.StatusInternalServerError},
		{err: litho.ErrOCRFailed, expect: http.StatusInternalServerError},
	} {
		t.Run(tc.err.Error(), func(t *testing.T) {
			err := &stageError{op: "rotate", filename: "doc.pdf", err: fmt.Errorf("/tmp/work/doc.pdf: %w", tc.err)}

			status, message := api.ParseError(handleError(err, "/tmp/work"))
			if status != tc.expect {
				t.Fatalf("status = %d, want %d (%s)", status, tc.expect, message)
			}
			if tc.expect == http.StatusBadRequest && message != fmt.Sprintf("Stage 'rotate' failed on 'doc.pdf': doc.pdf: %s", tc.err) {
				t.Errorf("message = %q, want the stage, the file and no working directory", message)
			}
		})
	}
}
