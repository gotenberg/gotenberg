package middleware

import (
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/handler"
	"github.com/thecodingmachine/gotenberg/internal/pkg/config"
	"github.com/thecodingmachine/gotenberg/internal/pkg/logger"
	"github.com/thecodingmachine/gotenberg/internal/pkg/random"
)

const (
	// TestingTraceEnvVar is an environment
	// variable used in some tests.
	TestingTraceEnvVar string = "TESTING_TRACE"
	// TestsTracePrefix helps
	// creating all resources inside a prefix.
	// Only used in some tests
	// to check if the resources
	// have been removed.
	TestsTracePrefix string = "tmp"
)

// Context helps extending the default echo.Context with
// our custom context.
func Context(config *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var trace string
			if os.Getenv(TestingTraceEnvVar) == "1" {
				trace = fmt.Sprintf("%s/%s", TestsTracePrefix, random.Get())
			} else {
				// generate a unique identifier for the request.
				trace = random.Get()
			}
			// create the logger for this request using
			// the previous identifier as trace.
			logger := logger.New(config.LogLevel(), trace)
			// extend the current echo context with our custom
			// context.
			ctx := context.New(c, logger, config)
			// if its an healthcheck request, there
			// is no resource associated to it.
			if ctx.Path() == handler.PingEndpoint {
				return next(ctx)
			}
			// if the endpoint is not for healthcheck, associate a
			// resource to our custom context.
			if err := ctx.WithResource(trace); err != nil {
				// required to have a correct status code.
				ctx.Error(err)
				return ctx.LogRequestResult(err, false)
			}
			return next(ctx)
		}
	}
}
