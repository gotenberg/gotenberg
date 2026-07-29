package libreoffice

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
	libreofficeapi "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
)

func TestWrapUnoException(t *testing.T) {
	testCases := []struct {
		name       string
		options    libreofficeapi.Options
		httpStatus int
	}{
		{
			name:       "invalid page ranges",
			options:    libreofficeapi.Options{PageRanges: "foo"},
			httpStatus: http.StatusBadRequest,
		},
		{
			name:       "unneeded password",
			options:    libreofficeapi.Options{Password: "foo"},
			httpStatus: http.StatusBadRequest,
		},
		{
			name:    "server-side failure",
			options: libreofficeapi.Options{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := fmt.Errorf("supervisor run task: %w", libreofficeapi.ErrUnoException)
			actual := wrapUnoException(source, testCase.options)

			var httpError api.HttpError
			if testCase.httpStatus == 0 {
				if !errors.Is(actual, libreofficeapi.ErrUnoException) {
					t.Fatalf("expected wrapped UNO exception, got %v", actual)
				}

				if errors.As(actual, &httpError) {
					status, _ := httpError.HttpError()
					t.Fatalf("expected an internal error, got HTTP status %d", status)
				}

				return
			}

			if !errors.As(actual, &httpError) {
				t.Fatal("expected an HTTP error")
			}

			status, _ := httpError.HttpError()
			if status != testCase.httpStatus {
				t.Fatalf("expected HTTP status %d, got %d", testCase.httpStatus, status)
			}
		})
	}
}
