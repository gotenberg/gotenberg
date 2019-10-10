package xhttp

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xrand"
)

// contextMiddleware extends the default echo.Context with
// our custom context.Context.
func contextMiddleware(config conf.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// generate a unique identifier for the request.
			trace := xrand.Get()
			// create the logger for this request using
			// the previous identifier as trace.
			logger := xlog.New(config.LogLevel(), trace)
			// extend the current echo context with our custom
			// context.
			ctx := context.New(c, logger, config)
			// if it's not a multipart/form-data request,
			// there is no need to create a Resource.
			if !isMultipartFormDataEndpoint(config, ctx.Path()) {
				// validate method for healthcheck endpoint.
				if ctx.Path() == pingEndpoint && ctx.Request().Method != http.MethodGet {
					err := doErr(ctx, echo.NewHTTPError(http.StatusMethodNotAllowed))
					return ctx.LogRequestResult(err, false)
				}
				return next(ctx)
			}
			// validate method.
			if ctx.Request().Method != http.MethodPost {
				err := doErr(ctx, echo.NewHTTPError(http.StatusMethodNotAllowed))
				return ctx.LogRequestResult(err, false)
			}
			// validate Content-Type.
			contentType := ctx.Request().Header.Get("Content-Type")
			if !strings.Contains(contentType, "multipart/form-data") {
				err := doErr(ctx, echo.NewHTTPError(http.StatusUnsupportedMediaType))
				return ctx.LogRequestResult(err, false)
			}
			// it's a multipart/form-data request, create a
			// Resource.
			if err := ctx.WithResource(trace); err != nil {
				err = doCleanup(ctx, err)
				err = doErr(ctx, err)
				return ctx.LogRequestResult(err, false)
			}
			return next(ctx)
		}
	}
}

// loggerMiddleware logs the result of a request.
func loggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.MustCastFromEchoContext(c)
			err := next(ctx)
			// we do not want to log healthcheck requests if
			// log level is not set to DEBUG.
			isDebug := ctx.Path() == pingEndpoint
			return ctx.LogRequestResult(err, isDebug)
		}
	}
}

// cleanupMiddleware removes a resource.Resource
// at the end of a request.
func cleanupMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			ctx := context.MustCastFromEchoContext(c)
			return doCleanup(ctx, err)
		}
	}
}

// errorMiddleware handles errors (if any).
func errorMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.MustCastFromEchoContext(c)
			err := next(ctx)
			if err == nil {
				// so far so good!
				return nil
			}
			return doErr(ctx, err)
		}
	}
}

func doCleanup(ctx context.Context, err error) error {
	const op string = "xhttp.cleanup"
	if !ctx.HasResource() {
		// nothing to remove.
		return err
	}
	r := ctx.MustResource()
	// if a webhook URL has been given,
	// do not remove the resource.Resource here because
	// we don't know if the result file has been
	// generated or sent.
	if r.HasArg(resource.WebhookURLArgKey) {
		return err
	}
	// a resource.Resource is associated with our custom context.
	if resourceErr := r.Close(); resourceErr != nil {
		xerr := xerror.New(op, resourceErr)
		ctx.XLogger().ErrorOp(xerror.Op(xerr), xerr)
	}
	return err
}

func doErr(ctx context.Context, err error) error {
	// if it's an error from echo
	// like 404 not found and so on.
	if echoHTTPErr, ok := err.(*echo.HTTPError); ok {
		// required to have a correct status code.
		ctx.Error(echoHTTPErr)
		return echoHTTPErr
	}
	// we log the initial error before returning
	// the HTTP error.
	errOp := xerror.Op(err)
	logger := ctx.XLogger()
	logger.ErrorOp(errOp, err)
	// handle our custom HTTP error.
	var httpErr error
	errCode := xerror.Code(err)
	errMessage := xerror.Message(err)
	switch errCode {
	case xerror.InvalidCode:
		httpErr = echo.NewHTTPError(http.StatusBadRequest, errMessage)
	case xerror.TimeoutCode:
		httpErr = echo.NewHTTPError(http.StatusGatewayTimeout, errMessage)
	default:
		httpErr = echo.NewHTTPError(http.StatusInternalServerError, errMessage)
	}
	// required to have a correct status code.
	ctx.Error(httpErr)
	return httpErr
}
