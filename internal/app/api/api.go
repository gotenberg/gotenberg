package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

// API describes a Gotenberg API.
type API struct {
	srv     *echo.Echo
	chrome  pm2.Process
	unoconv pm2.Process
	opts    *Options
}

// Options allows to customize the behaviour
// of the API.
type Options struct {
	DisableGoogleChrome bool
	DisableUnoconv      bool
}

// New returns a new API.
func New(opts *Options) *API {
	if opts == nil {
		opts = &Options{
			DisableGoogleChrome: false,
			DisableUnoconv:      false,
		}
	}
	return &API{
		chrome:  &pm2.Chrome{},
		unoconv: &pm2.Unoconv{},
		opts:    opts,
	}
}

// Start starts the API.
func (api *API) Start(port string) error {
	if !api.opts.DisableGoogleChrome {
		if err := api.chrome.Start(); err != nil {
			return err
		}
	}
	if !api.opts.DisableUnoconv {
		if err := api.unoconv.Start(); err != nil {
			return err
		}
	}
	api.srv = createEchoHTTPServer(api.opts)
	return api.srv.Start(port)
}

// Shutdown shutdowns the API.
func (api *API) Shutdown(ctx context.Context) error {
	// shutdown the HTTP server first with a deadline to
	// wait for.
	if api.srv != nil {
		if err := api.srv.Shutdown(ctx); err != nil {
			return err
		}
	}
	// then our PM2 processes.
	if api.chrome.State() == pm2.RunningState {
		if err := api.chrome.Shutdown(); err != nil {
			return err
		}
	}
	if api.unoconv.State() == pm2.RunningState {
		if err := api.unoconv.Shutdown(); err != nil {
			return err
		}
	}
	return nil
}

func createEchoHTTPServer(opts *Options) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Logger())
	// FIXME is this middleware required? (see https://echo.labstack.com/guide/error-handling)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := next(c); err != nil {
				// TODO should return a better HTTP status code
				// than 500 for some cases.
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("%v", err))
			}
			return nil
		}
	})
	e.GET("/ping", func(c echo.Context) error { return nil })
	e.POST("/merge", merge)
	if opts.DisableGoogleChrome && opts.DisableUnoconv {
		return e
	}
	g := e.Group("/convert")
	if !opts.DisableGoogleChrome {
		g.POST("/html", convertHTML)
		g.POST("/url", convertURL)
		g.POST("/markdown", convertMarkdown)
	}
	if !opts.DisableUnoconv {
		g.POST("/office", convertOffice)
	}
	return e
}

func print(c echo.Context, p printer.Printer, r *resource) error {
	baseFilename, err := rand.Get()
	if err != nil {
		return hijackErr(fmt.Errorf("getting result file name: %v", err), r)
	}
	filename := fmt.Sprintf("%s.pdf", baseFilename)
	fpath := fmt.Sprintf("%s/%s", r.dirPath, filename)
	// if no webhook URL given, run conversion
	// and directly return the resulting PDF file
	// or an error.
	if r.webhookURL() == "" {
		defer r.removeAll()
		if err := p.Print(fpath); err != nil {
			return err
		}
		if r.filename() != "" {
			filename = r.filename()
		}
		return c.Attachment(fpath, filename)
	}
	// as a webhook URL has been given, we
	// run the following lines in a goroutine so that
	// it doesn't block.
	go func() {
		defer r.removeAll()
		if err := p.Print(fpath); err != nil {
			c.Logger().Errorf("%v", err)
			return
		}
		f, err := os.Open(fpath)
		if err != nil {
			c.Logger().Errorf("%v", err)
			return
		}
		defer f.Close()
		resp, err := http.Post(r.webhookURL(), "application/pdf", f)
		if err != nil {
			c.Logger().Errorf("%v", err)
			return
		}
		defer resp.Body.Close()
	}()
	return nil
}

func hijackErr(err error, r *resource) error {
	if r != nil {
		defer r.removeAll()
	}
	return err
}
