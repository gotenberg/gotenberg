package handler

import (
	"github.com/labstack/echo/v4"
)

// Ping is the endpoint for healthcheck.
func Ping(c echo.Context) error {
	return nil
}
