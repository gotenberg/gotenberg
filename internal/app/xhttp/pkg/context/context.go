package context

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/normalize"
	"github.com/thecodingmachine/gotenberg/internal/pkg/prinery"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

// Context extends the default echo.Context.
type Context struct {
	echo.Context
	logger    xlog.Logger
	config    conf.Config
	prinry    prinery.Prinery
	resource  resource.Resource
	startTime time.Time
}

// New creates a new Context.
func New(
	c echo.Context,
	logger xlog.Logger,
	config conf.Config,
	prinry prinery.Prinery,
) Context {
	return Context{
		c,
		logger,
		config,
		prinry,
		resource.Resource{},
		time.Now(),
	}
}

/*
MustCastFromEchoContext cast an echo.Context
to our custom Context.

It panics if casting goes wrong.
*/
func MustCastFromEchoContext(c echo.Context) Context {
	const op string = "context.MustCastFromEchoContext"
	ctx, ok := c.(Context)
	if !ok {
		panic(fmt.Sprintf("%s: unable to cast an echo.Context to our custom context.Context", op))
	}
	return ctx
}

/*
XLogger returns the xlog.Logger associated
with the Context.

This method should be used instead of the
default Logger() method coming from
the echo.Context.
*/
func (ctx Context) XLogger() xlog.Logger {
	return ctx.logger
}

// Config returns the conf.Config associated
// with the Context.
func (ctx Context) Config() conf.Config {
	return ctx.config
}

// ProcessesHealthcheck returns an error if
// one of the processes is not viable.
func (ctx Context) ProcessesHealthcheck() error {
	const op string = "context.Context.ProcessesHealthcheck"
	// TODO
	/*processes := ctx.manager.All()
	for _, p := range processes {
		if !ctx.manager.IsViable(p) {
			return xerror.New(
				op,
				fmt.Errorf("'%s' is not viable", p.ID()),
			)
		}
	}*/
	return nil
}

// Prinery returns the instance of prinery.Prinery
// associated with the Context.
func (ctx Context) Prinery() prinery.Prinery {
	return ctx.prinry
}

// WithResource creates a resource.Resource and
// adds it to the Context.
func (ctx *Context) WithResource(directoryName string) error {
	const op string = "context.Context.WithResource"
	resolver := func() (resource.Resource, error) {
		r, err := resource.New(ctx.logger, directoryName)
		if err != nil {
			return r, err
		}
		// retrieve form values from request.
		for _, key := range resource.ArgKeys() {
			r.WithArg(key, ctx.FormValue(string(key)))
		}
		// write form files from request.
		form, err := ctx.MultipartForm()
		if err != nil {
			/*
				(very) special case: one and
				only one file has been sent
				and it is empty.
			*/
			if strings.Contains(err.Error(), io.EOF.Error()) {
				return r, xerror.Invalid(op, "one file has been sent but it is empty: does it exist?", err)
			}
			return r, err
		}
		for _, files := range form.File {
			for _, fh := range files {
				in, err := fh.Open()
				if err != nil {
					return r, err
				}
				defer in.Close() // nolint: errcheck
				filename, err := normalize.String(fh.Filename)
				if err != nil {
					return r, err
				}
				if err := r.WithFile(filename, in); err != nil {
					return r, err
				}
			}
		}
		return r, nil
	}
	resource, err := resolver()
	ctx.resource = resource
	if err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// HasResource returns true if the Context
// has a resource.Resource.
func (ctx Context) HasResource() bool {
	return !reflect.DeepEqual(ctx.resource, resource.Resource{})
}

/*
MustResource returns the resource.Resource
associated with the Context.

It panics if no resource.Resource.
*/
func (ctx Context) MustResource() resource.Resource {
	const op string = "context.Context.MustResource"
	if !ctx.HasResource() {
		panic(fmt.Sprintf("%s: unable to retrieve the resource.Resource from our custom context.Context", op))
	}
	return ctx.resource
}

/*
LogRequestResult logs the result of a request.
This method should only be used by a middleware!

If an error is given, returns the exact same error.
*/
func (ctx Context) LogRequestResult(err error, isDebug bool) error {
	const op string = "context.Context.LogRequestResult"
	req := ctx.Request()
	resp := ctx.Response()
	stopTime := time.Now()
	fields := map[string]interface{}{
		"remote_ip":     ctx.RealIP(),
		"host":          req.Host,
		"uri":           req.RequestURI,
		"method":        req.Method,
		"path":          path(req),
		"referer":       req.Referer(),
		"user_agent":    req.UserAgent(),
		"status":        resp.Status,
		"latency":       lantency(ctx.startTime, stopTime),
		"latency_human": latencyHuman(ctx.startTime, stopTime),
		"bytes_in":      bytesIn(req),
		"bytes_out":     bytesOut(resp),
	}
	if err != nil {
		ctx.logger.WithFields(fields).ErrorfOp(op, "request failed")
		return err
	}
	if isDebug {
		ctx.logger.WithFields(fields).DebugfOp(op, "request handled")
		return nil
	}
	ctx.logger.WithFields(fields).InfofOp(op, "request handled")
	return nil
}

func path(r *http.Request) string {
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	return path
}

func lantency(startTime time.Time, stopTime time.Time) string {
	return strconv.FormatInt(int64(stopTime.Sub(startTime)), 10)
}

func latencyHuman(startTime time.Time, stopTime time.Time) string {
	return stopTime.Sub(startTime).String()
}

func bytesIn(r *http.Request) string {
	bytesIn := r.Header.Get(echo.HeaderContentLength)
	if bytesIn == "" {
		bytesIn = "0"
	}
	return bytesIn
}

func bytesOut(r *echo.Response) string {
	return strconv.FormatInt(r.Size, 10)
}
