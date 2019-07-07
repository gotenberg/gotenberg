package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/handler"
)

// Logger helps logging the result of a request.
func Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.MustCastFromEchoContext(c)
			err := next(ctx)
			// we do not want to log healthcheck requests if
			// log level is not set to DEBUG.
			isDebug := ctx.Path() == handler.PingEndpoint
			return ctx.LogRequestResult(err, isDebug)
		}
	}
}
