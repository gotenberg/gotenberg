package api

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// ContextMock is a helper for tests.
//
//	ctx := &api.ContextMock{Context: &api.Context{}}
type ContextMock struct {
	*Context
}

// SetDirPath sets the context's working directory path.
//
//	ctx := &api.ContextMock{Context: &api.Context{}}
//	ctx.SetDirPath("/foo")
func (ctx *ContextMock) SetDirPath(path string) {
	ctx.dirPath = path
}

// DirPath returns the context's working directory path.
//
//	ctx := &api.ContextMock{Context: &api.Context{}}
//	ctx.SetDirPath("/foo")
//	dirPath := ctx.DirPath()
func (ctx *ContextMock) DirPath() string {
	return ctx.dirPath
}

// SetValues sets the values.
//
//	ctx := &api.ContextMock{Context: &api.Context{}}
//	ctx.SetValues(map[string][]string{
//	  "url": {
//	    "foo",
//	  },
//	})
func (ctx *ContextMock) SetValues(values map[string][]string) {
	ctx.values = values
}

// SetFiles sets the files.
//
//	ctx := &api.ContextMock{Context: &api.Context{}}
//	ctx.SetFiles(map[string]string{
//	  "foo": "/foo",
//	})
func (ctx *ContextMock) SetFiles(files map[string]string) {
	ctx.files = files
}

// SetCancelled sets if the context is cancelled or not.
//
//	ctx := &api.ContextMock{Context: &api.Context{}}
//	ctx.SetCancelled(true)
func (ctx *ContextMock) SetCancelled(cancelled bool) {
	ctx.cancelled = cancelled
}

// OutputPaths returns the registered output paths.
//
//	ctx := &api.ContextMock{Context: &api.Context{}}
//	outputPaths := ctx.OutputPaths()
func (ctx ContextMock) OutputPaths() []string {
	return ctx.outputPaths
}

// SetLogger sets the logger.
//
//	ctx := &api.ContextMock{Context: &api.Context{}}
//	ctx.SetLogger(zap.NewNop())
func (ctx *ContextMock) SetLogger(logger *zap.Logger) {
	ctx.logger = logger
}

// SetEchoContext sets the echo.Context.
//
//	ctx := &api.ContextMock{Context: &api.Context{}}
//	ctx.setEchoContext(c)
func (ctx *ContextMock) SetEchoContext(c echo.Context) {
	ctx.Context.echoCtx = c
}
