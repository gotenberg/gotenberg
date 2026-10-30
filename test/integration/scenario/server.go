package scenario

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// request is a request the helper server received, recorded once fully read.
type request struct {
	method  string
	header  http.Header
	cookies []*http.Cookie
	body    []byte
}

// newRequest records r, whose body has already been read into body.
func newRequest(r *http.Request, body []byte) *request {
	return &request{method: r.Method, header: r.Header.Clone(), cookies: r.Cookies(), body: body}
}

// server is the helper HTTP server a scenario points Gotenberg at. Handlers run
// on their own goroutines, so every record lives behind mu.
type server struct {
	srv *echo.Echo

	mu sync.Mutex
	// webhook is the last webhook delivery, recorded once its output is stored
	// on disk, and webhookErr the error that delivery failed with.
	webhook    *request
	webhookErr error
	// webhookArrived signals a recorded webhook delivery. Handlers send on it
	// without blocking and nothing closes it, so a late retry from Gotenberg
	// can never panic the run.
	webhookArrived chan struct{}
	// page is the last page Chromium loaded from /html or /redirect-to-private,
	// and file the last file served from /static.
	page      *request
	file      *request
	eventBody []byte

	shutdown context.CancelFunc
	done     chan struct{}
}

func newServer(ctx context.Context, workdir string) (*server, error) {
	srv := echo.New()
	// The static file handlers below serve absolute paths, which Echo v5's
	// default working-directory filesystem rejects. See newEchoServer in
	// pkg/modules/api.
	srv.Filesystem = echo.NewDefaultFS("/")
	s := &server{
		srv:            srv,
		webhookArrived: make(chan struct{}, 1),
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current directory: %w", err)
	}

	webhookHandler := func(c *echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			err = fmt.Errorf("read request body: %w", err)
			s.recordWebhook(nil, err)
			return err
		}

		rec := newRequest(c.Request(), body)

		err = storeWebhookOutput(ctx, workdir, rec)
		s.recordWebhook(rec, err)
		if err != nil {
			return err
		}

		return c.String(http.StatusOK, http.StatusText(http.StatusOK))
	}

	srv.POST("/webhook", webhookHandler)
	srv.PATCH("/webhook", webhookHandler)
	srv.PUT("/webhook", webhookHandler)

	webhookEventsHandler := func(c *echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		s.mu.Lock()
		s.eventBody = body
		s.mu.Unlock()
		return c.String(http.StatusOK, http.StatusText(http.StatusOK))
	}
	srv.POST("/webhook/events", webhookEventsHandler)
	srv.GET("/static/:path", func(c *echo.Context) error {
		s.recordRequest(&s.file, c.Request())
		path := c.Param("path")
		if strings.Contains(path, "teststore") {
			return c.Attachment(fmt.Sprintf("%s/%s/%s", workdir, c.Request().Header.Get("Gotenberg-Trace"), filepath.Base(path)), filepath.Base(path))
		}
		return c.Attachment(fmt.Sprintf("%s/%s", wd, path), filepath.Base(path))
	})
	srv.GET("/html/:path", func(c *echo.Context) error {
		s.recordRequest(&s.page, c.Request())
		path := fmt.Sprintf("%s/%s", wd, c.Param("path"))
		f, err := os.Open(path)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("open file %q: %s", path, err))
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("read file %q: %s", path, err))
		}
		return c.HTML(http.StatusOK, string(b))
	})
	srv.GET("/redirect-to-private", func(c *echo.Context) error {
		s.recordRequest(&s.page, c.Request())
		// Redirect the browser to a non-public address so the outbound filter
		// is exercised on the redirected request rather than on this URL.
		return c.Redirect(http.StatusFound, "http://127.0.0.1:9999/redirected")
	})

	return s, nil
}

// storeWebhookOutput writes the output a webhook delivered into the working
// directory of its request, and extracts it when it is a zip archive.
func storeWebhookOutput(ctx context.Context, workdir string, rec *request) error {
	filename, ok, err := outputFilename(rec.header.Get("Content-Disposition"))
	if err != nil {
		return err
	}

	contentType := rec.header.Get("Content-Type")
	if !ok {
		if rec.header.Get("Content-Disposition") == "" {
			return errors.New("no Content-Disposition header")
		}

		switch contentType {
		case "application/zip":
			filename = uuid.NewString() + ".zip"
		case "application/pdf":
			filename = uuid.NewString() + ".pdf"
		default:
			return errors.New("no filename in Content-Disposition header")
		}
	}

	dirPath := fmt.Sprintf("%s/%s", workdir, rec.header.Get("Gotenberg-Trace"))
	return storeOutput(ctx, dirPath, filename, contentType, rec.body)
}

// recordRequest records r, whose body is not read, into *slot.
func (s *server) recordRequest(slot **request, r *http.Request) {
	rec := newRequest(r, nil)

	s.mu.Lock()
	defer s.mu.Unlock()
	*slot = rec
}

// recordWebhook records a webhook delivery and signals it.
func (s *server) recordWebhook(rec *request, err error) {
	s.mu.Lock()
	s.webhook, s.webhookErr = rec, err
	s.mu.Unlock()

	select {
	case s.webhookArrived <- struct{}{}:
	default:
	}
}

// waitForWebhook returns once a webhook delivery is recorded, with the error
// it failed with, or once ctx is done or timeout elapses.
func (s *server) waitForWebhook(ctx context.Context, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		s.mu.Lock()
		rec, err := s.webhook, s.webhookErr
		s.mu.Unlock()

		if err != nil {
			return err
		}
		if rec != nil {
			return nil
		}

		select {
		case <-s.webhookArrived:
		case <-ctx.Done():
			return fmt.Errorf("wait for the webhook: %w", ctx.Err())
		case <-deadline.C:
			return fmt.Errorf("no webhook request within %s", timeout)
		}
	}
}

// received returns the last request of the given kind: "webhook request",
// "file request" or "server request".
func (s *server) received(kind string) (*request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var rec *request
	switch kind {
	case "webhook request":
		rec = s.webhook
	case "file request":
		rec = s.file
	case "server request":
		rec = s.page
	default:
		return nil, fmt.Errorf("unknown request kind %q", kind)
	}

	if rec == nil {
		return nil, fmt.Errorf("no %s received", kind)
	}

	return rec, nil
}

func (s *server) getEventBody() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.eventBody
}

func (s *server) start(ctx context.Context) (int, error) {
	ln, err := listenOutsideEphemeralRange()
	if err != nil {
		return 0, err
	}

	port := ln.Addr().(*net.TCPAddr).Port

	startConfig := echo.StartConfig{
		Listener:   ln,
		HideBanner: true,
		HidePort:   true,
	}

	serveCtx, cancel := context.WithCancel(context.Background())
	s.shutdown = cancel
	s.done = make(chan struct{})

	go func() {
		defer close(s.done)

		err := startConfig.Start(serveCtx, s.srv)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			godog.Log(ctx, err.Error())
		}
	}()

	return port, nil
}

func (s *server) stop(ctx context.Context) error {
	if s.shutdown == nil {
		return nil
	}

	s.shutdown()

	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Helper ports come from this range, below the ephemeral ranges of Linux
// (32768-60999) and macOS (49152-65535) that the kernel hands out to ":0"
// listeners and Docker to published container ports. A helper port taken from
// the same range can collide with a Gotenberg container's published port on
// the other address family, and a webhook then reaches that container instead.
const (
	helperPortMin      = 20000
	helperPortMax      = 29999
	helperPortAttempts = 50
)

// listenOutsideEphemeralRange listens, on every address family, on a free port
// of the helper range.
func listenOutsideEphemeralRange() (net.Listener, error) {
	var lastErr error

	for range helperPortAttempts {
		port := helperPortMin + rand.IntN(helperPortMax-helperPortMin+1) // #nosec G404 -- picks a port, not a secret

		// #nosec G102 -- the Gotenberg container reaches the helper through the host gateway.
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return ln, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("listen on a port between %d and %d: %w", helperPortMin, helperPortMax, lastErr)
}
