package webhook

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// client gathers all the data required to send a request to a webhook.
type client struct {
	url              string
	method           string
	errorUrl         string
	errorMethod      string
	extraHttpHeaders map[string]string
	startTime        time.Time

	client *retryablehttp.Client
	logger *zap.Logger
}

// send call the webhook either to send the success response or the error response.
func (c client) send(body io.Reader, headers map[string]string, erroed bool) error {
	URL := c.url
	if erroed {
		URL = c.errorUrl
	}

	method := c.method
	if erroed {
		method = c.errorMethod
	}

	req, err := retryablehttp.NewRequest(method, URL, body)
	if err != nil {
		return fmt.Errorf("create '%s' request to '%s': %w", method, URL, err)
	}

	req.Header.Set("User-Agent", "Gotenberg")

	// Extra HTTP headers are the custom headers from the user.
	for key, value := range c.extraHttpHeaders {
		req.Header.Set(key, value)
	}

	// Middleware caller's headers > extra HTTP headers from the user.

	contentLength, ok := headers[echo.HeaderContentLength]
	if ok {
		// Golang "http" package should automatically calculate the size of the
		// body. But, when using a buffered file reader, it does not work.
		// Worse, the "Content-Length" header is also removed. Therefore, in
		// order to keep this valuable information, we have to trust the caller
		// by reading the value of the "Content-Length" entry and set it as the
		// content length of the request. It's kinda suboptimal, but hey, at
		// least it works.

		bodySize, err := strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			return fmt.Errorf("parse content length entry: %w", err)
		}

		req.ContentLength = bodySize
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send '%s' request to '%s': %w", method, URL, err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("send '%s' request to '%s': got status: '%s'", method, URL, resp.Status)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			c.logger.Error(fmt.Sprintf("close response body from '%s': %s", URL, err))
		}
	}()

	// Last piece for calculating the latency.
	finishTime := time.Now()

	// Now let's log!
	fields := make([]zap.Field, 5)
	fields[0] = zap.String("webhook_url", URL)
	fields[1] = zap.String("method", method)
	fields[2] = zap.Int64("latency", int64(finishTime.Sub(c.startTime)))
	fields[3] = zap.String("latency_human", finishTime.Sub(c.startTime).String())
	fields[4] = zap.Int64("bytes_out", req.ContentLength)

	if erroed {
		c.logger.Warn("request to webhook with error details handled", fields...)

		return nil
	}

	c.logger.Info("request to webhook handled", fields...)

	return nil
}

// leveledLogger is wrapper around a [zap.Logger] which is used by the
// [retryablehttp.Client].
type leveledLogger struct {
	logger *zap.Logger
}

// Error logs a message at error level using the wrapped zap.Logger.
func (leveled leveledLogger) Error(msg string, keysAndValues ...interface{}) {
	leveled.logger.Error(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Warn logs a message at warning level using the wrapped zap.Logger.
func (leveled leveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	leveled.logger.Warn(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Info logs a message at info level using the wrapped zap.Logger.
func (leveled leveledLogger) Info(msg string, keysAndValues ...interface{}) {
	leveled.logger.Info(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Debug logs a message at debug level using the wrapped zap.Logger.
func (leveled leveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	leveled.logger.Debug(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Interface guards.
var (
	_ retryablehttp.LeveledLogger = (*leveledLogger)(nil)
)
