package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

// API describes a Gotenberg API.
type API struct {
	srv *echo.Echo
}

// Options allows to customize the behaviour
// of the API.
type Options struct {
	DefaultWaitTimeout     float64
	EnableChromeEndpoints  bool
	EnableUnoconvEndpoints bool
}

type errBadRequest struct {
	err error
}

func (e *errBadRequest) Error() string {
	return e.err.Error()
}

// New returns an API.
func New(opts *Options) *API {
	api := &API{}
	api.srv = echo.New()
	api.srv.HideBanner = true
	api.srv.HidePort = true
	api.srv.Use(middleware.Logger())
	api.srv.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		// middleware for extending default context with our
		// custom constext.
		return func(c echo.Context) error {
			ctx := &resourceContext{c, opts, nil}
			r, err := newResource(ctx)
			if err != nil {
				if resourceErr := r.close(); resourceErr != nil {
					c.Logger().Error(resourceErr)
				}
				return err
			}
			ctx.resource = r
			return next(ctx)
		}
	})
	api.srv.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		// middleware for handling errors and removing resources
		// once the request has been handled.
		return func(c echo.Context) error {
			err := next(c)
			ctx := c.(*resourceContext)
			// if a webhookURL has been given,
			// do not remove the resources here because
			// we don't know if the result file has been
			// generated or sent.
			if !ctx.resource.has(webhookURL) {
				if resourceErr := ctx.resource.close(); resourceErr != nil {
					c.Logger().Error(resourceErr)
				}
			}
			if err != nil {
				if _, ok := err.(*echo.HTTPError); ok {
					return err
				}
				if _, ok := err.(*errBadRequest); ok {
					return echo.NewHTTPError(http.StatusBadRequest, err.Error())
				}
				if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
					return echo.NewHTTPError(http.StatusRequestTimeout)
				}
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			return nil
		}
	})
	api.srv.GET("/ping", func(c echo.Context) error { return nil })
	api.srv.POST("/merge", merge)
	if !opts.EnableChromeEndpoints && !opts.EnableUnoconvEndpoints {
		return api
	}
	g := api.srv.Group("/convert")
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

// Start starts the API.
func (api *API) Start(port string) error {
	return api.srv.Start(port)
}

// Shutdown shutdowns the API.
func (api *API) Shutdown(ctx context.Context) error {
	if api.srv != nil {
		if err := api.srv.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

func convert(ctx *resourceContext, p printer.Printer) error {
	baseFilename, err := rand.Get()
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s.pdf", baseFilename)
	fpath := fmt.Sprintf("%s/%s", ctx.resource.formFilesDirPath, filename)
	// if no webhook URL given, run conversion
	// and directly return the resulting PDF file
	// or an error.
	if !ctx.resource.has(webhookURL) {
		if err := p.Print(fpath); err != nil {
			return err
		}
		if ctx.resource.has(resultFilename) {
			filename, err = ctx.resource.get(resultFilename)
			if err != nil {
				return &errBadRequest{err}
			}
		}
		return ctx.Attachment(fpath, filename)
	}
	// as a webhook URL has been given, we
	// run the following lines in a goroutine so that
	// it doesn't block.
	go func() {
		defer ctx.resource.close() // nolint: errcheck
		if err := p.Print(fpath); err != nil {
			ctx.Logger().Error(err)
			return
		}
		f, err := os.Open(fpath)
		if err != nil {
			ctx.Logger().Error(err)
			return
		}
		defer f.Close() // nolint: errcheck
		// TODO post with resultFilename
		webhook, err := ctx.resource.get(webhookURL)
		if err != nil {
			ctx.Logger().Error(err)
			return
		}
		resp, err := http.Post(webhook, "application/pdf", f) /* #nosec */
		if err != nil {
			ctx.Logger().Error(err)
			return
		}
		defer resp.Body.Close() // nolint: errcheck
	}()
	return nil
}
