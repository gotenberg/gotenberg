package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

// Cleanup helps removing a resource at the end of a request.
func Cleanup() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			const op = "middleware.Cleanup"
			err := next(c)
			ctx := context.MustCastFromEchoContext(c)
			r := ctx.Resource()
			if r == nil {
				return err
			}
			// if a webhook URL has been given,
			// do not remove the resource here because
			// we don't know if the result file has been
			// generated or sent.
			if r.Has(resource.WebhookURLFormField) {
				return err
			}
			// a resource is associated with our custom context.
			if resourceErr := r.Close(); resourceErr != nil {
				ctx.StandardLogger().ErrorOp(op, &standarderror.Error{
					Op:  op,
					Err: resourceErr,
				})
			}
			return err
		}
	}
}
