package sentry

import (
	"net/http"

	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

// sentryPanicMiddleware is the middleware from sentry-go/echo for panic reporting and hub setup.
// It should run with a higher priority to ensure the Sentry Hub is available for other middlewares.
func sentryPanicMiddleware() api.Middleware {
	return api.Middleware{
		Stack:    api.DefaultStack,
		Priority: api.HighPriority, // Ensures Sentry Hub is set up early.
		Handler: sentryecho.New(sentryecho.Options{
			Repanic: true,
		}),
	}
}

// sentryErrorCaptureMiddleware captures specific non-panic errors and sends them to Sentry.
// It should run with a lower priority to catch errors from main handlers.
func sentryErrorCaptureMiddleware() api.Middleware {
	return api.Middleware{
		Stack:    api.DefaultStack,
		Priority: api.LowPriority, // Runs after most handlers but before final error response generation.
		Handler: func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				err := next(c)
				if err != nil {
					// Use api.ParseError to determine the HTTP status code this error would generate.
					status, _ := api.ParseError(err)

					if status == http.StatusBadRequest || status == http.StatusInternalServerError || status == http.StatusServiceUnavailable {
						if hub := sentryecho.GetHubFromContext(c); hub != nil {
							hub.CaptureException(err)
						}
					}
				}
				return err // Return the original error to be handled by Echo's main error handler.
			}
		},
	}
}
