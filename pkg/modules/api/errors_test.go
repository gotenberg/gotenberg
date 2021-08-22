package api

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestNewSentinelHTTPError(t *testing.T) {
	actual := NewSentinelHTTPError(http.StatusInternalServerError, "foo")
	expect := SentinelHTTPError{
		status:  http.StatusInternalServerError,
		message: "foo",
	}

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %v but got %v", expect, actual)
	}
}

func TestSentinelHTTPError_Error(t *testing.T) {
	err := SentinelHTTPError{
		message: "foo",
	}

	actual := err.Error()
	expect := "foo"

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestSentinelHTTPError_HTTPError(t *testing.T) {
	actualStatus, actualMessage := SentinelHTTPError{
		status:  http.StatusInternalServerError,
		message: "foo",
	}.HTTPError()

	expectStatus := http.StatusInternalServerError
	expectMessage := "foo"

	if actualStatus != expectStatus {
		t.Errorf("expected %d but got %d", expectStatus, actualStatus)
	}

	if actualMessage != expectMessage {
		t.Errorf("expected '%s' but got '%s'", expectMessage, actualMessage)
	}
}

func TestSentinelWrappedError_Is(t *testing.T) {
	errSentinel := SentinelHTTPError{}

	err := sentinelWrappedError{
		error:    errors.New("foo"),
		sentinel: errSentinel,
	}

	if !err.Is(errSentinel) {
		t.Error("expected true")
	}
}

func TestSentinelWrappedError_HTTPError(t *testing.T) {
	expectStatus, expectMessage := SentinelHTTPError{
		status:  http.StatusInternalServerError,
		message: "foo",
	}.HTTPError()

	actualStatus, actualMessage := sentinelWrappedError{
		error: errors.New("foo"),
		sentinel: SentinelHTTPError{
			status:  http.StatusInternalServerError,
			message: "foo",
		},
	}.HTTPError()

	if actualStatus != expectStatus {
		t.Errorf("expected %d but got %d", expectStatus, actualStatus)
	}

	if actualMessage != expectMessage {
		t.Errorf("expected '%s' but got '%s'", expectMessage, actualMessage)
	}
}

func TestWrapError(t *testing.T) {
	errFoo := errors.New("foo")

	expect := sentinelWrappedError{
		error: errFoo,
		sentinel: SentinelHTTPError{
			status:  http.StatusInternalServerError,
			message: "foo",
		},
	}

	actual := WrapError(errFoo, SentinelHTTPError{
		status:  http.StatusInternalServerError,
		message: "foo",
	})

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %v but got %v", expect, actual)
	}
}
