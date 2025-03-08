package scenario

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
	"github.com/labstack/echo/v4"
	"github.com/mholt/archives"
)

type server struct {
	srv *echo.Echo
	// TODO: many requests for static content
	req     *http.Request
	errChan chan error
}

func newServer(ctx context.Context, workdir string) (*server, error) {
	srv := echo.New()
	srv.HideBanner = true
	srv.HidePort = true
	s := &server{
		srv:     srv,
		errChan: make(chan error, 1),
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current directory: %w", err)
	}

	webhookErr := func(err error) error {
		s.errChan <- err
		return err
	}

	webhookHandler := func(c echo.Context) error {
		s.req = c.Request()

		body, err := io.ReadAll(s.req.Body)
		if err != nil {
			return webhookErr(fmt.Errorf("read request body: %w", err))
		}

		cd := s.req.Header.Get("Content-Disposition")
		if cd == "" {
			return webhookErr(fmt.Errorf("no Content-Disposition header"))
		}

		_, params, err := mime.ParseMediaType(cd)
		if err != nil {
			return webhookErr(fmt.Errorf("parse Content-Disposition header: %w", err))
		}

		filename, ok := params["filename"]
		if !ok {
			return webhookErr(errors.New("no filename in Content-Disposition header"))
		}

		dirPath := fmt.Sprintf("%s/%s", workdir, s.req.Header.Get("Gotenberg-Trace"))
		err = os.MkdirAll(dirPath, 0o755)
		if err != nil {
			return webhookErr(fmt.Errorf("create working directory: %w", err))
		}

		fpath := fmt.Sprintf("%s/%s", dirPath, filename)
		file, err := os.Create(fpath)
		if err != nil {
			return webhookErr(fmt.Errorf("create file %q: %w", fpath, err))
		}
		defer file.Close()

		_, err = file.Write(body)
		if err != nil {
			return webhookErr(fmt.Errorf("write file %q: %w", fpath, err))
		}

		if s.req.Header.Get("Content-Type") == "application/zip" {
			var format archives.Zip
			err = format.Extract(ctx, file, func(ctx context.Context, f archives.FileInfo) error {
				source, err := f.Open()
				if err != nil {
					return fmt.Errorf("open file %q: %w", f.Name(), err)
				}
				defer source.Close()

				targetPath := fmt.Sprintf("%s/%s", dirPath, f.Name())
				target, err := os.Create(targetPath)
				if err != nil {
					return fmt.Errorf("create file %q: %w", targetPath, err)
				}
				defer target.Close()

				_, err = io.Copy(target, source)
				if err != nil {
					return fmt.Errorf("copy file %q: %w", targetPath, err)
				}

				return nil
			})
			if err != nil {
				return webhookErr(err)
			}
		}

		return webhookErr(c.String(http.StatusOK, http.StatusText(http.StatusOK)))
	}
	webhookErrorHandler := func(c echo.Context) error {
		s.req = c.Request()
		return webhookErr(c.String(http.StatusOK, http.StatusText(http.StatusOK)))
	}

	srv.POST("/webhook", webhookHandler)
	srv.PATCH("/webhook", webhookHandler)
	srv.PUT("/webhook", webhookHandler)
	srv.POST("/webhook/error", webhookErrorHandler)
	srv.PATCH("/webhook/error", webhookErrorHandler)
	srv.PUT("/webhook/error", webhookErrorHandler)
	srv.GET("/static/:path", func(c echo.Context) error {
		s.req = c.Request()
		path := c.Param("path")
		if strings.Contains(path, "teststore") {
			return c.Attachment(fmt.Sprintf("%s/%s/%s", workdir, s.req.Header.Get("Gotenberg-Trace"), filepath.Base(path)), filepath.Base(path))
		}
		return c.Attachment(fmt.Sprintf("%s/%s", wd, path), filepath.Base(path))
	})

	return s, nil
}

func (s *server) start(ctx context.Context, port int) {
	// As the following code is blocking, run it in a goroutine.
	go func() {
		err := s.srv.Start(fmt.Sprintf(":%d", port))
		if !errors.Is(err, http.ErrServerClosed) {
			godog.Log(ctx, err.Error())
		}
	}()
}

func (s *server) stop(ctx context.Context) error {
	close(s.errChan)
	return s.srv.Shutdown(ctx)
}
