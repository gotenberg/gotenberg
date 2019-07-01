package api

import (
	"github.com/labstack/echo/v4"
	conf "github.com/thecodingmachine/gotenberg/internal/pkg/config"
)

const pingEndpoint = "/ping"

// New returns an API.
func New(config *conf.Config) *echo.Echo {
	api := echo.New()
	api.HideBanner = true
	api.HidePort = true
	api.Use(contextMiddleware(config))
	api.Use(loggingMiddleware())
	api.Use(finalizeMiddleware())
	api.GET(pingEndpoint, func(c echo.Context) error { return nil })
	api.POST("/merge", merge)
	if !config.EnableChromeEndpoints() && !config.EnableUnoconvEndpoints() {
		return api
	}
	g := api.Group("/convert")
	if config.EnableChromeEndpoints() {
		g.POST("/html", convertHTML)
		g.POST("/url", convertURL)
		g.POST("/markdown", convertMarkdown)
	}
	if config.EnableUnoconvEndpoints() {
		g.POST("/office", convertOffice)
	}
	return api
}
