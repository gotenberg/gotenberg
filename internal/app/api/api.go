package api

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/handler"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/middleware"
	"github.com/thecodingmachine/gotenberg/internal/pkg/config"
)

// New returns an API.
func New(config *config.Config) *echo.Echo {
	api := echo.New()
	api.HideBanner = true
	api.HidePort = true
	api.Use(middleware.Context(config))
	api.Use(middleware.Logger())
	api.Use(middleware.Cleanup())
	api.Use(middleware.Error())
	api.GET(handler.PingEndpoint, handler.Ping)
	api.POST(handler.MergeEndpoint, handler.Merge)
	if !config.EnableChromeEndpoints() && !config.EnableUnoconvEndpoints() {
		return api
	}
	g := api.Group(handler.ConvertGroupEndpoint)
	if config.EnableChromeEndpoints() {
		g.POST(handler.HTMLEndpoint, handler.HTML)
		g.POST(handler.URLEndpoint, handler.URL)
		g.POST(handler.MarkdownEndpoint, handler.Markdown)
	}
	if config.EnableUnoconvEndpoints() {
		g.POST(handler.OfficeEndpoint, handler.Office)
	}
	return api
}
