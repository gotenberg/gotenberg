package scenario

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
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
	// canaries lists, as "METHOD /path", every request that reached /canary/.
	// A scenario that expects a filter to refuse a URL points it there: the
	// refusal holds only if the list stays empty.
	canaries []string

	// httpsPort is where the HTTPS canary listens, set by start. It records
	// every connection as a request to /canary/https: a converter only gets
	// one through a tunnel the filters let open, and the TLS handshake that
	// follows fails, as the converter does not trust the helper.
	httpsPort int

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

	// A webhook URL that fails every upload.
	srv.POST("/failing-webhook", func(c *echo.Context) error {
		_, err := io.Copy(io.Discard, c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	})

	tsa, err := newTimestampAuthority()
	if err != nil {
		return nil, fmt.Errorf("create timestamp authority: %w", err)
	}
	srv.POST("/tsa", echo.WrapHandler(tsa))
	srv.GET("/static/:path", func(c *echo.Context) error {
		s.recordRequest(&s.file, c.Request())
		path := c.Param("path")
		if strings.Contains(path, "teststore") {
			return c.Attachment(fmt.Sprintf("%s/%s/%s", workdir, c.Request().Header.Get("Gotenberg-Trace"), filepath.Base(path)), filepath.Base(path))
		}
		return c.Attachment(fmt.Sprintf("%s/%s", wd, path), filepath.Base(path))
	})
	// Files served without a Content-Disposition header, as object stores
	// such as S3 often do: under their path, which ends with their filename,
	// or under a path that names nothing, the file given by the query.
	srv.GET("/raw/:path", func(c *echo.Context) error {
		s.recordRequest(&s.file, c.Request())
		return c.File(fmt.Sprintf("%s/%s", wd, c.Param("path")))
	})
	srv.GET("/unnamed", func(c *echo.Context) error {
		s.recordRequest(&s.file, c.Request())
		return c.File(fmt.Sprintf("%s/%s", wd, c.QueryParam("path")))
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
	// failOnHttpStatusCodes tells the main page by its frame: a path-less URL
	// Chromium requests with a trailing slash, and a redirect, both reach a
	// missing page, while a framed missing page is a resource.
	missing := func(c *echo.Context) error {
		s.recordRequest(&s.page, c.Request())
		return c.HTML(http.StatusNotFound, "<html><body><p>Missing</p></body></html>")
	}
	srv.GET("/", missing)
	srv.GET("/missing", missing)
	srv.GET("/redirect-to-missing", func(c *echo.Context) error {
		s.recordRequest(&s.page, c.Request())
		return c.Redirect(http.StatusFound, "/missing")
	})
	srv.GET("/iframe-missing", func(c *echo.Context) error {
		s.recordRequest(&s.page, c.Request())
		return c.HTML(http.StatusOK, `<html><body><p>Framed</p><iframe src="/missing"></iframe></body></html>`)
	})
	srv.GET("/redirect-to-private", func(c *echo.Context) error {
		s.recordRequest(&s.page, c.Request())
		// Redirect the browser to a non-public address so the outbound filter
		// is exercised on the redirected request rather than on this URL.
		return c.Redirect(http.StatusFound, "http://127.0.0.1:9999/redirected")
	})
	// A page that never answers: a request that loads it runs into the API
	// timeout itself, not into a wait of its own, which names itself.
	srv.GET("/slow", func(c *echo.Context) error {
		s.recordRequest(&s.page, c.Request())
		<-c.Request().Context().Done()
		return nil
	})
	// Canaries. Any request under /canary/ counts, whatever its method, a
	// WebSocket handshake included, so a filter that logs a refusal but lets
	// the request through still fails the scenario.
	srv.Any("/canary/*", func(c *echo.Context) error {
		// Recorded first: a request counts once it arrives, even if reading
		// its body fails.
		r := c.Request()
		s.mu.Lock()
		s.canaries = append(s.canaries, r.Method+" "+r.URL.Path)
		s.mu.Unlock()

		_, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusOK, "canary")
	})
	// A redirect to a canary, for a filter that checks the URL it is given
	// but must also check where that URL leads.
	srv.GET("/canary-redirect/*", func(c *echo.Context) error {
		return c.Redirect(http.StatusFound, "/canary/"+c.Param("*"))
	})
	// A testdata file whose "{canary}" tokens become the canary URL, as
	// Gotenberg reaches it. Fixtures cannot hold that URL themselves, since
	// the helper port changes with every scenario. No Content-Disposition:
	// Chromium renders an HTML page instead of downloading it, and
	// downloadFrom takes the filename from the path.
	srv.GET("/canary-page/*", func(c *echo.Context) error {
		path := filepath.Clean(c.Param("*"))
		if !strings.HasPrefix(path, "testdata/") {
			return c.String(http.StatusBadRequest, fmt.Sprintf("path %q is not under testdata/", path))
		}

		b, err := os.ReadFile(filepath.Join(wd, path))
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("read file %q: %s", path, err))
		}

		canary := fmt.Sprintf("http://%s/canary", c.Request().Host)
		host, _, err := net.SplitHostPort(c.Request().Host)
		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("split host %q: %s", c.Request().Host, err))
		}
		s.mu.Lock()
		httpsCanary := fmt.Sprintf("https://%s/canary", net.JoinHostPort(host, fmt.Sprint(s.httpsPort)))
		s.mu.Unlock()
		b = []byte(strings.ReplaceAll(string(b), "{canary}", canary))
		b = []byte(strings.ReplaceAll(string(b), "{canary-https}", httpsCanary))

		contentType := mime.TypeByExtension(filepath.Ext(path))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		return c.Blob(http.StatusOK, contentType, b)
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

// canaryRequests returns the requests that reached /canary/, oldest first.
func (s *server) canaryRequests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.canaries)
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

	httpsLn, err := listenOutsideEphemeralRange()
	if err != nil {
		_ = ln.Close()
		return 0, err
	}

	s.mu.Lock()
	s.httpsPort = httpsLn.Addr().(*net.TCPAddr).Port
	s.mu.Unlock()

	startConfig := echo.StartConfig{
		Listener:   ln,
		HideBanner: true,
		HidePort:   true,
	}

	serveCtx, cancel := context.WithCancel(context.Background())
	s.shutdown = cancel
	s.done = make(chan struct{})

	go s.serveHTTPSCanary(serveCtx, httpsLn)

	go func() {
		defer close(s.done)

		err := startConfig.Start(serveCtx, s.srv)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			godog.Log(ctx, err.Error())
		}
	}()

	return port, nil
}

// serveHTTPSCanary records each connection ln accepts as a canary request,
// then closes it, until ctx ends.
func (s *server) serveHTTPSCanary(ctx context.Context, ln net.Listener) {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}

		s.mu.Lock()
		s.canaries = append(s.canaries, "CONNECT /canary/https")
		s.mu.Unlock()

		_ = conn.Close()
	}
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
