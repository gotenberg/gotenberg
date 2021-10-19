package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
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
	"golang.org/x/net/http2"
)

func init() {
	gotenberg.MustRegisterModule(API{})
}

// API is a module which provides an HTTP server. Other modules may add routes,
// middlewares or health checks.
type API struct {
	port                      int
	readTimeout               time.Duration
	processTimeout            time.Duration
	writeTimeout              time.Duration
	rootPath                  string
	traceHeader               string
	disableHealthCheckLogging bool

	routes              []Route
	externalMiddlewares []Middleware
	healthChecks        []health.CheckerOption
	gcGraceDuration     time.Duration
	logger              *zap.Logger
	srv                 *echo.Echo
}

// Router is a module interface which adds routes to the API.
type Router interface {
	Routes() ([]Route, error)
}

// Route represents a route from a Router.
type Route struct {
	// Method is the HTTP method of the route (i.e., GET, POST, etc.).
	// Required.
	Method string

	// Path is the sub path of the route. Must start with a slash.
	// Required.
	Path string

	// IsMultipart tells if the route is "multipart/form-data".
	// Optional.
	IsMultipart bool

	// DisableLogging disables the logging for this route.
	// Optional.
	DisableLogging bool

	// Handler is the function which handles the request.
	// Required.
	Handler echo.HandlerFunc
}

// MiddlewareProvider is a module interface which adds middlewares to the API.
type MiddlewareProvider interface {
	Middlewares() ([]Middleware, error)
}

// MiddlewareStack is a type which helps to determine in which stack the
// middlewares provided by the MiddlewareProvider modules should be located.
type MiddlewareStack uint32

const (
	DefaultStack MiddlewareStack = iota
	PreRouterStack
	MultipartStack
)

// MiddlewarePriority is a type which helps to determine the execution order of
// middlewares provided by the MiddlewareProvider modules in a stack.
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
//  middleware := Middleware{
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
	// Stack tells in which stack the middleware should be located.
	// Default to DefaultStack.
	// Optional.
	Stack MiddlewareStack

	// Priority tells if the middleware should be positioned high or not in
	// its stack.
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

// GarbageCollectorGraceDurationIncrementer is a module interface for
// increasing the grace duration provided by the API for the garbage collector.
type GarbageCollectorGraceDurationIncrementer interface {
	AddGraceDuration() time.Duration
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
	mods, err := ctx.Modules(new(Router))
	if err != nil {
		return fmt.Errorf("get routers: %w", err)
	}

	routers := make([]Router, len(mods))
	for i, router := range mods {
		routers[i] = router.(Router)
	}

	for _, router := range routers {
		routes, err := router.Routes()
		if err != nil {
			return fmt.Errorf("get routes: %w", err)
		}

		a.routes = append(a.routes, routes...)
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

	// Grace duration.
	a.gcGraceDuration = a.readTimeout + a.processTimeout + a.writeTimeout

	mods, err = ctx.Modules(new(GarbageCollectorGraceDurationIncrementer))
	if err != nil {
		return fmt.Errorf("get garbage collector grace duration increments: %w", err)
	}

	for _, incrementer := range mods {
		a.gcGraceDuration += incrementer.(GarbageCollectorGraceDurationIncrementer).AddGraceDuration()
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

	routesMap := make(map[string]string, len(a.routes)+1)
	routesMap["/health"] = "/health"

	for _, route := range a.routes {
		if route.Path == "" {
			return errors.New("route with empty path cannot be registered")
		}

		if !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("route '%s' does not start with /", route.Path)
		}

		if route.IsMultipart && !strings.HasPrefix(route.Path, "/forms") {
			return fmt.Errorf("multipart/form-data route '%s' does not start with /forms", route.Path)
		}

		if route.Method == "" {
			return fmt.Errorf("route '%s' has an empty method", route.Path)
		}

		if route.Handler == nil {
			return fmt.Errorf("route '%s' has a nil handler", route.Path)
		}

		if _, ok := routesMap[route.Path]; ok {
			return fmt.Errorf("route '%s' is already registered", route.Path)
		}

		routesMap[route.Path] = route.Path
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
	a.srv.HTTPErrorHandler = httpErrorHandler()

	// Let's prepare the modules' routes.
	var disableLoggingForPaths []string
	for i, route := range a.routes {
		a.routes[i].Path = strings.TrimPrefix(route.Path, "/")

		if route.DisableLogging {
			disableLoggingForPaths = append(disableLoggingForPaths, strings.TrimPrefix(route.Path, "/"))
		}
	}

	// Check if the user wish to add logging entries related to the health
	// check route.
	if a.disableHealthCheckLogging {
		disableLoggingForPaths = append(disableLoggingForPaths, "health")
	}

	// Add the API middlewares.
	a.srv.Pre(
		latencyMiddleware(),
		rootPathMiddleware(a.rootPath),
		traceMiddleware(a.traceHeader),
		timeoutsMiddleware(a.readTimeout, a.processTimeout, a.writeTimeout),
		loggerMiddleware(a.logger, disableLoggingForPaths),
	)

	// Add the modules' middlewares in their respective stacks.
	var externalMultipartMiddlewares []Middleware
	for _, externalMiddleware := range a.externalMiddlewares {
		switch externalMiddleware.Stack {
		case PreRouterStack:
			a.srv.Pre(externalMiddleware.Handler)
		case MultipartStack:
			externalMultipartMiddlewares = append(externalMultipartMiddlewares, externalMiddleware)
		default:
			a.srv.Use(externalMiddleware.Handler)
		}
	}

	hardTimeout := a.processTimeout + (time.Duration(5) * time.Second)

	// Add the modules' routes and their specific middlewares.
	for _, route := range a.routes {
		var middlewares []echo.MiddlewareFunc

		if route.IsMultipart {
			middlewares = append(middlewares, contextMiddleware(a.processTimeout))

			for _, externalMultipartMiddleware := range externalMultipartMiddlewares {
				middlewares = append(middlewares, externalMultipartMiddleware.Handler)
			}
		}

		middlewares = append(middlewares, hardTimeoutMiddleware(hardTimeout))

		a.srv.Add(
			route.Method,
			fmt.Sprintf("%s%s", a.rootPath, route.Path),
			route.Handler,
			middlewares...,
		)
	}

	// Let's not forget the health check route.
	a.srv.GET(
		fmt.Sprintf("%s%s", a.rootPath, "health"),
		func() echo.HandlerFunc {
			checks := append(a.healthChecks, health.WithTimeout(a.processTimeout))
			checker := health.NewChecker(checks...)

			return echo.WrapHandler(health.NewHandler(checker))
		}(),
		hardTimeoutMiddleware(hardTimeout),
	)

	// As the following code is blocking, run it in a goroutine.
	go func() {
		server := &http2.Server{}
		err := a.srv.StartH2CServer(fmt.Sprintf(":%d", a.port), server)
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
	return a.gcGraceDuration
}

// Interface guards.
var (
	_ gotenberg.Module                         = (*API)(nil)
	_ gotenberg.Provisioner                    = (*API)(nil)
	_ gotenberg.Validator                      = (*API)(nil)
	_ gotenberg.App                            = (*API)(nil)
	_ gc.GarbageCollectorGraceDurationModifier = (*API)(nil)
)
