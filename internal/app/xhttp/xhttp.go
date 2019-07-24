package xhttp

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
)

// New returns a custom echo.Echo.
func New(config conf.Config, processes ...pm2.Process) *echo.Echo {
	srv := echo.New()
	srv.HideBanner = true
	srv.HidePort = true
	srv.Use(contextMiddleware(config, processes...))
	srv.Use(loggerMiddleware())
	srv.Use(cleanupMiddleware())
	srv.Use(errorMiddleware())
	srv.GET(pingEndpoint, pingHandler)
	srv.POST(mergeEndpoint, mergeHandler)
	if config.DisableGoogleChrome() && config.DisableUnoconv() {
		return srv
	}
	g := srv.Group(convertGroupEndpoint)
	if !config.DisableGoogleChrome() {
		g.POST(htmlEndpoint, htmlHandler)
		g.POST(urlEndpoint, urlHandler)
		g.POST(markdownEndpoint, markdownHandler)
	}
	if !config.DisableUnoconv() {
		g.POST(officeEndpoint, officeHandler)
	}
	return srv
}
