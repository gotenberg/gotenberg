package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func handleLogging(enableHealthcheckLogging bool) echo.MiddlewareFunc {
	if enableHealthcheckLogging {
		// default logging middleware.
		return middleware.Logger()
	}
	// middleware for skipping logging when the ping endpoint is called.
	return middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == pingEndpoint
		},
	})
}

func handleContext(opts *Options) echo.MiddlewareFunc {
	// middleware for extending default context with our
	// custom constext.
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := &resourceContext{c, opts, nil}
			r, err := newResource(ctx)
			if err != nil {
				if resourceErr := r.close(); resourceErr != nil {
					c.Logger().Error(resourceErr)
				}
				return err
			}
			ctx.resource = r
			return next(ctx)
		}
	}
}

func handleError() echo.MiddlewareFunc {
	// middleware for handling errors and removing resources
	// once the request has been handled.
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			ctx := c.(*resourceContext)
			// if a webhookURL has been given,
			// do not remove the resources here because
			// we don't know if the result file has been
			// generated or sent.
			if !ctx.resource.has(webhookURL) {
				if resourceErr := ctx.resource.close(); resourceErr != nil {
					c.Logger().Error(resourceErr)
				}
			}
			if err != nil {
				if _, ok := err.(*echo.HTTPError); ok {
					return err
				}
				if _, ok := err.(*errBadRequest); ok {
					return echo.NewHTTPError(http.StatusBadRequest, err.Error())
				}
				if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
					return echo.NewHTTPError(http.StatusRequestTimeout)
				}
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			return nil
		}
	}
}
