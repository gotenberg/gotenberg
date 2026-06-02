package api

import (
	"context"
	"errors"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestLibreofficeErrorType(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"deadline", context.DeadlineExceeded, "timeout"},
		{"canceled", context.Canceled, "context_cancelled"},
		{"invalid pdf formats", ErrInvalidPdfFormats, "invalid_input"},
		{"uno exception", ErrUnoException, "libreoffice_exception"},
		{"runtime exception", ErrRuntimeException, "libreoffice_exception"},
		{"queue size exceeded", gotenberg.ErrMaximumQueueSizeExceeded, "libreoffice_unavailable"},
		{"process restarting", gotenberg.ErrProcessAlreadyRestarting, "libreoffice_unavailable"},
		{"core dumped", ErrCoreDumped, "unknown"},
		{"unknown", errors.New("boom"), "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := libreofficeErrorType(tc.err); got != tc.want {
				t.Errorf("libreofficeErrorType(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
