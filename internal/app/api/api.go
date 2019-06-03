package api

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const pingEndpoint = "/ping"

// Options allows to customize the behaviour
// of the API.
type Options struct {
	DefaultWaitTimeout     float64
	EnableChromeEndpoints  bool
	EnableUnoconvEndpoints bool
	EnablePingLogging      bool
}

// DefaultOptions returns default options.
func DefaultOptions() *Options {
	return &Options{
		DefaultWaitTimeout:     10,
		EnableChromeEndpoints:  true,
		EnableUnoconvEndpoints: true,
		EnablePingLogging:      true,
	}
}

// New returns an API.
func New(opts *Options) *echo.Echo {
	api := echo.New()
	api.HideBanner = true
	api.HidePort = true
	api.Use(loggerConfig(opts.EnablePingLogging))
	api.GET(pingEndpoint, func(c echo.Context) error { return nil })
	g := api.Group("/convert")
	g.Use(handleContext(opts))
	g.Use(handleError())
	g.POST("/merge", merge)
	if !opts.EnableChromeEndpoints && !opts.EnableUnoconvEndpoints {
		return api
	}
	if opts.EnableChromeEndpoints {
		g.POST("/html", convertHTML)
		g.POST("/url", convertURL)
		g.POST("/markdown", convertMarkdown)
	}
	if opts.EnableUnoconvEndpoints {
		g.POST("/office", convertOffice)
	}
	return api
}

// If proper ENV given, skips logging when the ping endpoint is called;
// otherwise logs
func loggerConfig(enablePingLogging bool) echo.MiddlewareFunc {
	if !enablePingLogging {
		return middleware.LoggerWithConfig(middleware.LoggerConfig{
			Skipper: func(c echo.Context) bool {
				r := c.Request()
				return r.URL.Path == pingEndpoint
			},
		})
	}

	return middleware.Logger()
}
