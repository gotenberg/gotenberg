package api

import (
	"context"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// Start starts the API server on port 3000.
func Start() error {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	g := e.Group("/convert")
	g.POST("/html", convertHTML)
	g.POST("/markdown", nil)
	g.POST("/office", nil)
	return e.Start(":3000")
}

func newContext(r *resource) (context.Context, context.CancelFunc) {
	webhookURL := r.webhookURL()
	if webhookURL == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		return ctx, cancel
	}
	return context.Background(), nil
}
