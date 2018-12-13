package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

// Start starts the API server on port 3000.
func Start() error {
	e := setup()
	// start Chrome headless and
	// unoconv listener with PM2.
	chrome := &pm2.Chrome{}
	unoconv := &pm2.Unoconv{}
	if err := chrome.Launch(); err != nil {
		return err
	}
	if err := unoconv.Launch(); err != nil {
		return err
	}
	// run our API in a goroutine so that it doesn't block.
	go func() {
		notify.Println("http server started on port 3000")
		if err := e.Start(":3000"); err != nil {
			e.Logger.Fatalf("%v", err)
			os.Exit(1)
		}
	}()
	quit := make(chan os.Signal, 1)
	// we'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(quit, os.Interrupt)
	// block until we receive our signal.
	<-quit
	// create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	// doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	notify.Println("shutting down http server... (Ctrl+C to force)")
	return e.Shutdown(ctx)
}

func setup() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Logger())
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
	e.POST("/merge", merge)
	g := e.Group("/convert")
	g.POST("/html", convertHTML)
	g.POST("/markdown", convertMarkdown)
	g.POST("/office", convertOffice)
	return e
}

func newContext(r *resource) (context.Context, context.CancelFunc) {
	webhookURL := r.webhookURL()
	if webhookURL == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		return ctx, cancel
	}
	return context.Background(), nil
}

func print(c echo.Context, p printer.Printer, r *resource) error {
	baseFilename, err := rand.Get()
	if err != nil {
		return fmt.Errorf("getting result file name: %v", err)
	}
	filename := fmt.Sprintf("%s.pdf", baseFilename)
	fpath := fmt.Sprintf("%s/%s", r.dirPath, filename)
	if r.webhookURL() == "" {
		// if no webhook URL given, run conversion
		// and directly return the resulting PDF file
		// or an error.
		if err := p.Print(fpath); err != nil {
			return err
		}
		return c.Attachment(fpath, filename)
	}
	// as a webhook URL has been given, we
	// run the following lines in a goroutine so that
	// it doesn't block.
	go func() {
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
