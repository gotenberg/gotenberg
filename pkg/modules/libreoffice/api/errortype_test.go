package api

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

func TestLibreofficeErrorType(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"deadline", context.DeadlineExceeded, "timeout"},
		{"canceled", context.Canceled, "context_cancelled"},
		{"wrapped deadline", fmt.Errorf("supervisor run task: %w", context.DeadlineExceeded), "timeout"},
		{"wrapped canceled", fmt.Errorf("supervisor run task: %w", context.Canceled), "context_cancelled"},
		{"invalid pdf formats", ErrInvalidPdfFormats, "invalid_input"},
		{"load failed", ErrLoadFailed, "invalid_input"},
		{"invalid password", ErrInvalidPassword, "invalid_input"},
		{"save failed", ErrSaveFailed, "libreoffice_exception"},
		{"crashed", ErrCrashed, "libreoffice_exception"},
		{"queue size exceeded", gotenberg.ErrMaximumQueueSizeExceeded, "libreoffice_unavailable"},
		{"process restarting", gotenberg.ErrProcessAlreadyRestarting, "libreoffice_unavailable"},
		{"unclassified", errors.New("boom"), "_OTHER"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := libreofficeErrorType(tc.err); got != tc.want {
				t.Errorf("libreofficeErrorType(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
