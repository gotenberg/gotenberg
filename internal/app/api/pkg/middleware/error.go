package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

// Error helps handling errors (if any).
func Error() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.MustCastFromEchoContext(c)
			err := next(ctx)
			if err == nil {
				// so far so good!
				return nil
			}
			// we log the initial error before returning
			// the HTTP error.
			errOp := standarderror.Op(err)
			logger := ctx.StandardLogger()
			logger.ErrorOp(errOp, err)
			// handle our custom HTTP error.
			var httpErr error
			errCode := standarderror.Code(err)
			errMessage := standarderror.Message(err)
			switch errCode {
			case standarderror.Invalid:
				httpErr = echo.NewHTTPError(http.StatusBadRequest, errMessage)
			case standarderror.Timeout:
				httpErr = echo.NewHTTPError(http.StatusRequestTimeout, errMessage)
			default:
				httpErr = echo.NewHTTPError(http.StatusInternalServerError, errMessage)
			}
			// required to have a correct status code
			// in the logs.
			ctx.Error(httpErr)
			return httpErr
		}
	}
}
