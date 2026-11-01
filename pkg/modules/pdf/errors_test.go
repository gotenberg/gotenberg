package pdf

import (
	"errors"
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

// TestHandleError_LithoSteps pins the client message of a failure litho
// reports from its own steps: without the pipeline stage and the mutation it
// locates the failure in, while the logged error keeps them.
func TestHandleError_LithoSteps(t *testing.T) {
	cause := fmt.Errorf("pdf stamp: source page 9: %w", litho.ErrInvalidPageNumber)

	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "mutation at write", err: &litho.PipelineStageError{Index: 1, Name: "write", Err: fmt.Errorf("applying mutation 0: %w", cause)}},
		{name: "staged mutation", err: &litho.PipelineStageError{Index: 2, Name: "apply", Err: fmt.Errorf("applying staged mutation 3 before redaction: %w", cause)}},
		{name: "pipeline stage only", err: &litho.PipelineStageError{Index: 1, Name: "write", Err: cause}},
		{name: "no litho step", err: cause},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := handleError(&stageError{op: "stamp-pdf", filename: "doc.pdf", err: tc.err}, "/tmp/work")

			status, message := api.ParseError(err)
			want := fmt.Sprintf("Stage 'stamp-pdf' failed on 'doc.pdf': %s", cause)
			if status != http.StatusBadRequest || message != want {
				t.Fatalf("got %d %q, want 400 %q", status, message, want)
			}

			if err.Error() == "" || !errors.Is(err, litho.ErrInvalidPageNumber) {
				t.Fatalf("the logged error lost its chain: %v", err)
			}
		})
	}

	// A message that only looks like a litho step stays whole.
	err := handleError(&stageError{op: "rotate", filename: "doc.pdf", err: fmt.Errorf("applying the rotation: %w", litho.ErrInvalidPageNumber)}, "/tmp/work")
	if _, message := api.ParseError(err); message != fmt.Sprintf("Stage 'rotate' failed on 'doc.pdf': applying the rotation: %s", litho.ErrInvalidPageNumber) {
		t.Fatalf("message = %q, want it whole", message)
	}
}
