package api

import "github.com/labstack/echo/v4"

const pingEndpoint = "/ping"

// Options allows to customize the behaviour
// of the API.
type Options struct {
	DefaultWaitTimeout       float64
	DefaultListenPort        string
	EnableChromeEndpoints    bool
	EnableUnoconvEndpoints   bool
	EnableHealthcheckLogging bool
	DebugProcessStartup      bool
}

// DefaultOptions returns default options.
func DefaultOptions() *Options {
	return &Options{
		DefaultWaitTimeout:       10,
		DefaultListenPort:        "3000",
		EnableChromeEndpoints:    true,
		EnableUnoconvEndpoints:   true,
		EnableHealthcheckLogging: true,
	}
}

// New returns an API.
func New(opts *Options) *echo.Echo {
	api := echo.New()
	api.HideBanner = true
	api.HidePort = true
	api.Use(handleLogging(opts.EnableHealthcheckLogging))
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
