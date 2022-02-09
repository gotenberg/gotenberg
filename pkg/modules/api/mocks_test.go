package api

import (
	"reflect"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func TestContextMock_SetDirPath(t *testing.T) {
	mock := &ContextMock{&Context{}}
	mock.SetDirPath("/foo")

	actual := mock.dirPath
	expect := "/foo"

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestContextMock_SetValues(t *testing.T) {
	mock := &ContextMock{&Context{}}
	mock.SetValues(map[string][]string{
		"foo": {"foo"},
	})

	actual := mock.values
	expect := map[string][]string{
		"foo": {"foo"},
	}

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %+v but got: %+v", expect, actual)
	}
}

func TestContextMock_SetFiles(t *testing.T) {
	mock := &ContextMock{&Context{}}
	mock.SetFiles(map[string]string{
		"foo": "/foo",
	})

	actual := mock.files
	expect := map[string]string{
		"foo": "/foo",
	}

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %+v but got: %+v", expect, actual)
	}
}

func TestContextMock_SetCancelled(t *testing.T) {
	mock := &ContextMock{&Context{}}
	mock.SetCancelled(true)

	actual := mock.cancelled

	if !actual {
		t.Errorf("expected %t but got %t", true, actual)
	}
}

func TestContextMock_OutputPaths(t *testing.T) {
	mock := ContextMock{
		&Context{
			outputPaths: []string{"/foo"},
		},
	}

	actual := mock.OutputPaths()
	expect := []string{"/foo"}

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %+v but got: %+v", expect, actual)
	}
}

func TestContextMock_SetLogger(t *testing.T) {
	mock := ContextMock{&Context{}}

	expect := zap.NewNop()
	mock.SetLogger(expect)

	actual := mock.logger

	if actual != expect {
		t.Errorf("expected %v but got %v", expect, actual)
	}
}

func TestContextMock_SetEchoContext(t *testing.T) {
	mock := ContextMock{&Context{}}

	expect := echo.New().NewContext(nil, nil)
	mock.SetEchoContext(expect)

	actual := mock.echoCtx

	if actual != expect {
		t.Errorf("expected %v but got %v", expect, actual)
	}
}
