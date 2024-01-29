package api

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestNewSentinelHttpError(t *testing.T) {
	actual := NewSentinelHttpError(http.StatusInternalServerError, "foo")
	expect := SentinelHttpError{
		status:  http.StatusInternalServerError,
		message: "foo",
	}

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %v but got %v", expect, actual)
	}
}

func TestSentinelHttpError_Error(t *testing.T) {
	err := SentinelHttpError{
		message: "foo",
	}

	actual := err.Error()
	expect := "foo"

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestSentinelHttpError_HttpError(t *testing.T) {
	actualStatus, actualMessage := SentinelHttpError{
		status:  http.StatusInternalServerError,
		message: "foo",
	}.HttpError()

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
	errSentinel := SentinelHttpError{}

	err := sentinelWrappedError{
		error:    errors.New("foo"),
		sentinel: errSentinel,
	}

	if !err.Is(errSentinel) {
		t.Error("expected true")
	}
}

func TestSentinelWrappedError_HttpError(t *testing.T) {
	expectStatus, expectMessage := SentinelHttpError{
		status:  http.StatusInternalServerError,
		message: "foo",
	}.HttpError()

	actualStatus, actualMessage := sentinelWrappedError{
		error: errors.New("foo"),
		sentinel: SentinelHttpError{
			status:  http.StatusInternalServerError,
			message: "foo",
		},
	}.HttpError()

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
		sentinel: SentinelHttpError{
			status:  http.StatusInternalServerError,
			message: "foo",
		},
	}

	actual := WrapError(errFoo, SentinelHttpError{
		status:  http.StatusInternalServerError,
		message: "foo",
	})

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %v but got %v", expect, actual)
	}
}
