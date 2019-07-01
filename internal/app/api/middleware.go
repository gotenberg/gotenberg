package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
	conf "github.com/thecodingmachine/gotenberg/internal/pkg/config"
	log "github.com/thecodingmachine/gotenberg/internal/pkg/logger"
)

func contextMiddleware(config *conf.Config) echo.MiddlewareFunc {
	// middleware for extending the default context
	// with one of our own context.
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// generate a unique identifier for our request.
			trace := random.String(32)
			// create the logger for this request using
			// the previous identifier as trace.
			logger := log.New(config.LogLevel(), trace)
			// extend the current echo context with our standard
			// context.
			ctx := newStandardContext(c, logger, config)
			// if the endpoint is not for liveness, make a
			// context with resource.
			if ctx.Path() != pingEndpoint {
				ctx, err := ctx.withResource(trace)
				if err != nil {
					ctx.Error(err)
					return ctx.logEndOfRequest(err)
				}
			}
			return next(ctx)
		}
	}
}

func loggingMiddleware() echo.MiddlewareFunc {
	// middleware for enabling logging.
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.(*standardContext)
			err := next(ctx)
			if err != nil {
				ctx.Error(err)
			}
			return ctx.logEndOfRequest(err)
		}
	}
}

func finalizeMiddleware() echo.MiddlewareFunc {
	// middleware for removing resources at the end of a request
	// and for improving response in case of error.
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			ctx, ok := c.(*resourceContext)
			// a resource is associated with the context.
			if ok {
				// if a webhookURL has been given,
				// do not remove the resources here because
				// we don't know if the result file has been
				// generated or sent.
				if !ctx.resource.has(webhookURL) {
					if resourceErr := ctx.resource.close(); resourceErr != nil {
						ctx.logger.Error(err)
					}
				}
			}
			if err == nil {
				return nil
			}
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}
			if _, ok := err.(*errBadRequest); ok {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
				return echo.NewHTTPError(http.StatusRequestTimeout, err.Error())
			}
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}
}
