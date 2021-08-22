package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/gc"
	"github.com/labstack/echo/v4"
	flag "github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func init() {
	gotenberg.MustRegisterModule(API{})
}

// API is a module which provides an HTTP server. Other modules may add
// "multipart/form-data" routes, middlewares or health checks.
type API struct {
	port                      int
	readTimeout               time.Duration
	processTimeout            time.Duration
	writeTimeout              time.Duration
	rootPath                  string
	traceHeader               string
	disableHealthCheckLogging bool
	webhookAllowList          *regexp.Regexp
	webhookDenyList           *regexp.Regexp
	webhookErrorAllowList     *regexp.Regexp
	webhookErrorDenyList      *regexp.Regexp
	webhookMaxRetry           int
	webhookRetryMinWait       time.Duration
	webhookRetryMaxWait       time.Duration
	disableWebhook            bool

	multipartFormDataRoutes []MultipartFormDataRoute
	externalMiddlewares     []Middleware
	healthChecks            []health.CheckerOption
	logger                  *zap.Logger
	srv                     *echo.Echo
}

// MultipartFormDataRouter is a module interface which adds
// "multipart/form-data" routes to the API.
type MultipartFormDataRouter interface {
	Routes() ([]MultipartFormDataRoute, error)
}

// MultipartFormDataRoute represents a "multipart/form-data" route. All routes
// uses the HTTP POST method.
type MultipartFormDataRoute struct {
	// Path is the sub path of the route. Must start with a slash.
	// Required.
	Path string

	// Handler is the function which handles the request.
	// Required.
	Handler func(ctx *Context) error
}

// MiddlewareProvider is a module interface which adds middlewares to the API.
type MiddlewareProvider interface {
	Middlewares() ([]Middleware, error)
}

// MiddlewarePriority is a type which helps to determine the execution order of
// middlewares provided by the MiddlewareProvider modules.
type MiddlewarePriority uint32

const (
	VeryLowPriority MiddlewarePriority = iota
	LowPriority
	MediumPriority
	HighPriority
	VeryHighPriority
)

// Middleware is a middleware which can be added to the API's middlewares
// chain.
//
//  middleware := &Middleware{
//    Handler: func() echo.MiddlewareFunc {
//      return func(next echo.HandlerFunc) echo.HandlerFunc {
//        return func(c echo.Context) error {
//          rootPath := c.Get("rootPath").(string)
//          healthURI := fmt.Sprintf("%shealth", rootPath)
//
//          // Skip the middleware if health check URI.
//          if c.Request().RequestURI == healthURI {
//            // Call the next middleware in the chain.
//            return next(c)
//          }
//
//          // Your middleware process.
//          // ...
//
//          // Call the next middleware in the chain.
//          return next(c)
//        }
//      }
//    }(),
//  }
type Middleware struct {
	// RunBeforeRouter tells if the middleware should run before the router
	// process an HTTP request.
	// Optional.
	RunBeforeRouter bool

	// Priority tells if the middleware should be positioned high or not in
	// the middlewares chain.
	// Default to VeryLowPriority.
	// Optional.
	Priority MiddlewarePriority

	// Handler is the function of the middleware.
	// Required.
	Handler echo.MiddlewareFunc
}

// HealthChecker is a module interface which allows adding health checks to the
// API.
//
// See https://github.com/alexliesenfeld/health for more details.
type HealthChecker interface {
	Checks() ([]health.CheckerOption, error)
}

// Descriptor returns an API's module descriptor.
func (API) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "api",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("api", flag.ExitOnError)
			fs.Int("api-port", 3000, "Set the port on which the API should listen")
			fs.String("api-port-from-env", "", "Set the environment variable with the port on which the API should listen - override the default port")
			fs.Duration("api-read-timeout", time.Duration(30)*time.Second, "Set the maximum duration allowed to read a complete request, including the body")
			fs.Duration("api-process-timeout", time.Duration(30)*time.Second, "Set the maximum duration allowed to process a request")
			fs.Duration("api-write-timeout", time.Duration(30)*time.Second, "Set the maximum duration before timing out writes of the response")
			fs.String("api-root-path", "/", "Set the root path of the API - for service discovery via URL paths")
			fs.String("api-trace-header", "Gotenberg-Trace", "Set the header name to use for identifying requests")
			fs.Bool("api-disable-health-check-logging", false, "Disable health check logging")
			fs.String("api-webhook-allow-list", "", "Set the allowed URLs for the webhook feature using a regular expression")
			fs.String("api-webhook-deny-list", "", "Set the denied URLs for the webhook feature using a regular expression")
			fs.String("api-webhook-error-allow-list", "", "Set the allowed URLs in case of an error for the webhook feature using a regular expression")
			fs.String("api-webhook-error-deny-list", "", "Set the denied URLs in case of an error for the webhook feature using a regular expression")
			fs.Int("api-webhook-max-retry", 4, "Set the maximum number of retries for the webhook feature")
			fs.Duration("api-webhook-retry-min-wait", time.Duration(1)*time.Second, "Set the minimum duration to wait before trying to call the webhook again")
			fs.Duration("api-webhook-retry-max-wait", time.Duration(30)*time.Second, "Set the maximum duration to wait before trying to call the webhook again")
			fs.Bool("api-disable-webhook", false, "Disable the webhook feature")

			return fs
		}(),
		New: func() gotenberg.Module { return new(API) },
	}
}

// Provision sets the module properties.
func (a *API) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	a.port = flags.MustInt("api-port")
	a.readTimeout = flags.MustDuration("api-read-timeout")
	a.processTimeout = flags.MustDuration("api-process-timeout")
	a.writeTimeout = flags.MustDuration("api-write-timeout")
	a.rootPath = flags.MustString("api-root-path")
	a.traceHeader = flags.MustString("api-trace-header")
	a.disableHealthCheckLogging = flags.MustBool("api-disable-health-check-logging")
	a.webhookAllowList = flags.MustRegexp("api-webhook-allow-list")
	a.webhookDenyList = flags.MustRegexp("api-webhook-deny-list")
	a.webhookErrorAllowList = flags.MustRegexp("api-webhook-error-allow-list")
	a.webhookErrorDenyList = flags.MustRegexp("api-webhook-error-deny-list")
	a.webhookMaxRetry = flags.MustInt("api-webhook-max-retry")
	a.webhookRetryMinWait = flags.MustDuration("api-webhook-retry-min-wait")
	a.webhookRetryMaxWait = flags.MustDuration("api-webhook-retry-max-wait")
	a.disableWebhook = flags.MustBool("api-disable-webhook")

	// Port from env?
	portEnvVar := flags.MustString("api-port-from-env")
	if portEnvVar != "" {
		val, ok := os.LookupEnv(portEnvVar)

		if !ok {
			return fmt.Errorf("environment variable '%s' does not exist", portEnvVar)
		}

		if val == "" {
			return fmt.Errorf("environment variable '%s' is empty", portEnvVar)
		}

		port, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("get int value of environment variable '%s': %w", portEnvVar, err)
		}

		a.port = port
	}

	// Get routes from modules.
	mods, err := ctx.Modules(new(MultipartFormDataRouter))
	if err != nil {
		return fmt.Errorf("get multipart/form-data routers: %w", err)
	}

	routers := make([]MultipartFormDataRouter, len(mods))
	for i, router := range mods {
		routers[i] = router.(MultipartFormDataRouter)
	}

	for _, router := range routers {
		routes, err := router.Routes()
		if err != nil {
			return fmt.Errorf("get routes: %w", err)
		}

		a.multipartFormDataRoutes = append(a.multipartFormDataRoutes, routes...)
	}

	// Get middlewares from modules.
	mods, err = ctx.Modules(new(MiddlewareProvider))
	if err != nil {
		return fmt.Errorf("get middleware providers: %w", err)
	}

	middlewareProviders := make([]MiddlewareProvider, len(mods))
	for i, middlewareProvider := range mods {
		middlewareProviders[i] = middlewareProvider.(MiddlewareProvider)
	}

	for _, middlewareProvider := range middlewareProviders {
		middlewares, err := middlewareProvider.Middlewares()
		if err != nil {
			return fmt.Errorf("get middlewares: %w", err)
		}

		a.externalMiddlewares = append(a.externalMiddlewares, middlewares...)
	}

	// Sort middlewares by priority.
	sort.Slice(a.externalMiddlewares, func(i, j int) bool {
		return a.externalMiddlewares[i].Priority > a.externalMiddlewares[j].Priority
	})

	// Get health checks from modules.
	mods, err = ctx.Modules(new(HealthChecker))
	if err != nil {
		return fmt.Errorf("get health checkers: %w", err)
	}

	healthCheckers := make([]HealthChecker, len(mods))
	for i, healthChecker := range mods {
		healthCheckers[i] = healthChecker.(HealthChecker)
	}

	for _, healthChecker := range healthCheckers {
		checks, err := healthChecker.Checks()
		if err != nil {
			return fmt.Errorf("get health checks: %w", err)
		}

		a.healthChecks = append(a.healthChecks, checks...)
	}

	loggerProvider, err := ctx.Module(new(gotenberg.LoggerProvider))
	if err != nil {
		return fmt.Errorf("get logger provider: %w", err)
	}

	logger, err := loggerProvider.(gotenberg.LoggerProvider).Logger(a)
	if err != nil {
		return fmt.Errorf("get logger: %w", err)
	}

	a.logger = logger

	return nil
}

// Validate validates the module properties.
func (a API) Validate() error {
	var err error

	if a.port < 1 || a.port > 65535 {
		err = multierr.Append(err,
			errors.New("port must be more than 1 and less than 65535"),
		)
	}

	if !strings.HasPrefix(a.rootPath, "/") {
		err = multierr.Append(err,
			errors.New("root path must start with /"),
		)
	}

	if !strings.HasSuffix(a.rootPath, "/") {
		err = multierr.Append(err,
			errors.New("root path must end with /"),
		)
	}

	if len(strings.TrimSpace(a.traceHeader)) == 0 {
		err = multierr.Append(err,
			errors.New("trace header must not be empty"),
		)
	}

	if err != nil {
		return err
	}

	routesMap := make(map[string]MultipartFormDataRoute, len(a.multipartFormDataRoutes))

	for _, route := range a.multipartFormDataRoutes {
		if route.Path == "" {
			return errors.New("route with empty path cannot be registered")
		}

		if !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("route %s does not start with /", route.Path)
		}

		if route.Handler == nil {
			return fmt.Errorf("route %s has a nil handler", route.Path)
		}

		if _, ok := routesMap[route.Path]; ok {
			return fmt.Errorf("route %s is already registered", route.Path)
		}

		routesMap[route.Path] = route
	}

	for _, middleware := range a.externalMiddlewares {
		if middleware.Handler == nil {
			return errors.New("a middleware has a nil handler")
		}
	}

	return nil
}

// Start starts the HTTP server.
func (a *API) Start() error {
	a.srv = echo.New()
	a.srv.HideBanner = true
	a.srv.HidePort = true
	a.srv.Server.ReadTimeout = a.readTimeout
	a.srv.Server.WriteTimeout = a.writeTimeout
	a.srv.HTTPErrorHandler = httpErrorHandler(a.traceHeader)

	a.srv.Pre(
		latencyMiddleware(),
		rootPathMiddleware(a.rootPath),
		traceMiddleware(a.traceHeader),
		loggerMiddleware(a.logger, a.disableHealthCheckLogging),
	)

	for _, externalMiddleware := range a.externalMiddlewares {
		if externalMiddleware.RunBeforeRouter {
			a.srv.Pre(externalMiddleware.Handler)

			continue
		}

		a.srv.Use(externalMiddleware.Handler)
	}

	hardTimeout := a.processTimeout + (time.Duration(5) * time.Second)

	a.srv.GET(
		fmt.Sprintf("%shealth", a.rootPath),
		func() echo.HandlerFunc {
			checks := append(a.healthChecks, health.WithTimeout(a.processTimeout))
			checker := health.NewChecker(checks...)

			return func(echoCtx echo.Context) error {
				health.NewHandler(checker).ServeHTTP(echoCtx.Response().Writer, echoCtx.Request())

				return nil
			}
		}(),
		timeoutMiddleware(hardTimeout),
	)

	formsGroup := a.srv.Group(
		fmt.Sprintf("%sforms", a.rootPath),
		contextMiddleware(
			contextMiddlewareConfig{
				traceHeader: a.traceHeader,
				timeout: struct {
					process time.Duration
					write   time.Duration
				}{
					process: a.processTimeout,
					write:   a.writeTimeout,
				},
				webhook: struct {
					allowList      *regexp.Regexp
					denyList       *regexp.Regexp
					errorAllowList *regexp.Regexp
					errorDenyList  *regexp.Regexp
					maxRetry       int
					retryMinWait   time.Duration
					retryMaxWait   time.Duration
					disable        bool
				}{
					allowList:      a.webhookAllowList,
					denyList:       a.webhookDenyList,
					errorAllowList: a.webhookErrorAllowList,
					errorDenyList:  a.webhookErrorDenyList,
					maxRetry:       a.webhookMaxRetry,
					retryMinWait:   a.webhookRetryMinWait,
					retryMaxWait:   a.webhookRetryMaxWait,
					disable:        a.disableWebhook,
				},
			},
		),
		timeoutMiddleware(hardTimeout),
	)

	// Add routes from other modules.
	for _, route := range a.multipartFormDataRoutes {
		formsGroup.POST(
			route.Path,
			func(route MultipartFormDataRoute) echo.HandlerFunc {
				return func(c echo.Context) error {
					ctx := c.Get("context").(*Context)

					err := route.Handler(ctx)
					if err != nil {
						return fmt.Errorf("handle request: %w", err)
					}

					return nil
				}
			}(route),
		)
	}

	// As the listen method is blocking, run it in a goroutine.
	go func() {
		err := a.srv.Start(fmt.Sprintf(":%d", a.port))
		if !errors.Is(err, http.ErrServerClosed) {
			a.logger.Fatal(err.Error())
		}
	}()

	return nil
}

// StartupMessage returns a custom startup message.
func (a API) StartupMessage() string {
	return fmt.Sprintf("server listening on port %d", a.port)
}

// Stop stops the HTTP server.
func (a API) Stop(ctx context.Context) error {
	return a.srv.Shutdown(ctx)
}

// GraceDuration updates the expiration time of files and directories parsed by
// the gc.GarbageCollector.
func (a API) GraceDuration() time.Duration {
	duration := a.readTimeout + a.processTimeout + a.writeTimeout

	if a.disableWebhook {
		return duration
	}

	for i := 0; i < a.webhookMaxRetry; i++ {
		// Yep... Golang does not allow int * time.Duration.
		duration += a.webhookRetryMaxWait
	}

	return duration
}

// Interface guards.
var (
	_ gotenberg.Module                         = (*API)(nil)
	_ gotenberg.Provisioner                    = (*API)(nil)
	_ gotenberg.Validator                      = (*API)(nil)
	_ gotenberg.App                            = (*API)(nil)
	_ gc.GarbageCollectorGraceDurationModifier = (*API)(nil)
)
