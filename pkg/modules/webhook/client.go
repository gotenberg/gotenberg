package webhook

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/labstack/echo/v5"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// client gathers all the data required to send a request to a webhook.
type client struct {
	url              string
	method           string
	eventsUrl        string
	extraHttpHeaders map[string]string
	startTime        time.Time

	// deliveryTimeout bounds one delivery including retries. See
	// [Webhook.deliveryTimeout].
	deliveryTimeout time.Duration

	client *retryablehttp.Client
	logger *slog.Logger
}

// deliveryContext returns the context one delivery runs on.
//
// It keeps the values of ctx, so trace propagation and logging correlation
// survive, and replaces its cancellation with a fresh budget. Threading the
// conversion context straight through does not work: a delivery starts after
// the handler returned, so that deadline may already be spent and the callback
// would fail without a single attempt.
func (c client) deliveryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := c.deliveryTimeout
	if timeout <= 0 {
		// An unset budget would expire the delivery before its first attempt.
		// [Webhook.deliveryTimeout] never returns a non-positive value, so this
		// only guards a caller that builds a client without one.
		timeout = minDeliveryTimeout
	}

	return context.WithTimeout(context.WithoutCancel(ctx), timeout)
}

// startDeliverySpan starts the span of one delivery to rawURL, which covers
// its attempts and the waits between them. Each attempt gets its own HTTP
// client span from the transport, under this one.
func startDeliverySpan(ctx context.Context, name, rawURL string) (context.Context, trace.Span) {
	return gotenberg.Tracer().Start(ctx, name,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(semconv.URLFull(gotenberg.RedactURL(rawURL))),
	)
}

// send sends the output file to the webhook URL.
func (c client) send(ctx context.Context, body io.Reader, headers map[string]string) (err error) {
	url, method := c.url, c.method
	// Messages and logs name the URL without its credentials, query and
	// fragment, which may hold a token.
	redactedURL := gotenberg.RedactURL(url)

	ctx, cancel := c.deliveryContext(ctx)
	defer cancel()

	ctx, span := startDeliverySpan(ctx, "webhook.send", url)
	defer func() {
		_ = gotenberg.SetSpanStatus(span, err, gotenberg.ClassifyError(err))
		span.End()
	}()

	// The request must carry ctx: retryablehttp.NewRequest builds on
	// [context.Background], and its wait between attempts is a select on the
	// request context, so a contextless request cannot be interrupted.
	req, err := retryablehttp.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("create '%s' request to '%s': %w", method, redactedURL, gotenberg.RedactURLError(err))
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
		// body. But when using a buffered file reader, it does not work.
		// Worse, the "Content-Length" header is also removed. Therefore,
		// to keep this valuable information, we have to trust the caller
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

	// Set last so that extra HTTP headers cannot misreport the version.
	// See https://github.com/gotenberg/gotenberg/issues/1693.
	req.Header.Set("Gotenberg-Version", gotenberg.Version)

	resp, err := c.client.Do(req)
	if err != nil {
		return deliveryFailure(fmt.Errorf("send '%s' request to '%s': %w", method, redactedURL, err))
	}

	// Registered before the status check below. [retryablehttp.Client.Do] hands
	// back a live body for a status it does not retry, which is every 4xx but
	// 429, so returning early without closing it strands the connection and the
	// transport goroutines that serve it for the lifetime of the process. The
	// transport is built per delivery in [gotenberg.NewOutboundHttpClient], so
	// nothing reclaims it later either.
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			c.logger.ErrorContext(ctx, fmt.Sprintf("close response body from '%s': %s", redactedURL, err))
		}
	}()

	// A status the client does not retry, such as a 4xx, comes back as a
	// response. The attempts before it are not known here.
	if resp.StatusCode >= http.StatusBadRequest {
		return deliveryFailure(fmt.Errorf("send '%s' request to '%s': %w", method, redactedURL, &deliveryError{status: resp.Status}))
	}

	// Last piece for calculating the latency.
	finishTime := time.Now()

	// Now let's log!
	attrs := []any{
		slog.String("webhook_url", redactedURL),
		slog.String("method", method),
		slog.Int64("latency", int64(finishTime.Sub(c.startTime))),
		slog.String("latency_human", finishTime.Sub(c.startTime).String()),
		slog.Int64("bytes_out", req.ContentLength),
	}

	c.logger.InfoContext(ctx, "request to webhook handled", attrs...)

	return nil
}

// deliveryError is a request to a webhook URL that failed for good.
type deliveryError struct {
	// attempts counts the requests sent, 0 when unknown.
	attempts int
	// status is the last response's status, such as "500 Internal Server
	// Error", or empty when the URL could not be reached.
	status string
	// err is the last transport error, when the URL could not be reached.
	err error
}

func (e *deliveryError) Error() string {
	switch {
	case e.status != "" && e.attempts == 0:
		return fmt.Sprintf("got status '%s'", e.status)
	case e.status != "":
		return fmt.Sprintf("got status '%s' after %d attempt(s)", e.status, e.attempts)
	default:
		return fmt.Sprintf("giving up after %d attempt(s): %v", e.attempts, e.err)
	}
}

func (e *deliveryError) Unwrap() error {
	return e.err
}

// message returns the client message of the failure, which names what
// happened without the transport's internals, such as addresses.
func (e *deliveryError) message() string {
	after := ""
	switch {
	case e.attempts == 1:
		after = " after 1 attempt"
	case e.attempts > 1:
		after = fmt.Sprintf(" after %d attempts", e.attempts)
	}

	if e.status != "" {
		return fmt.Sprintf("The webhook URL answered %s%s", e.status, after)
	}

	reason := unreachableReason(e.err)
	if reason == "" {
		return fmt.Sprintf("The webhook URL could not be reached%s", after)
	}

	return fmt.Sprintf("The webhook URL could not be reached%s: %s", after, reason)
}

// unreachableReason names why a webhook URL could not be reached, or returns
// "" when the reason would only expose internals, a policy refusal among
// them.
func unreachableReason(err error) string {
	var (
		unresolvable *gotenberg.UnresolvableHostError
		dnsErr       *net.DNSError
		netErr       net.Error
		certErr      *tls.CertificateVerificationError
		unknownCA    x509.UnknownAuthorityError
		hostnameErr  x509.HostnameError
	)

	switch {
	case errors.As(err, &unresolvable):
		return fmt.Sprintf("the host '%s' does not resolve", unresolvable.Host)
	case errors.Is(err, gotenberg.ErrFiltered):
		return ""
	case errors.As(err, &dnsErr) && dnsErr.IsNotFound:
		return fmt.Sprintf("the host '%s' does not resolve", dnsErr.Name)
	case errors.Is(err, syscall.ECONNREFUSED):
		return "the connection was refused"
	case errors.Is(err, syscall.ECONNRESET):
		return "the connection was reset"
	case errors.As(err, &certErr), errors.As(err, &unknownCA), errors.As(err, &hostnameErr):
		return "its TLS certificate is not trusted"
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return "the request timed out"
	default:
		return ""
	}
}

// giveUp is the [retryablehttp.ErrorHandler] of the webhook client: once the
// last attempt failed, it returns a [deliveryError] counting the attempts and
// naming the last status, which the default handler leaves out.
func giveUp(resp *http.Response, err error, attempts int) (*http.Response, error) {
	status := ""
	if resp != nil {
		status = resp.Status
		// Drained so that the connection may be reused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
	}

	// The transport error names the URL whole: redacted, as the messages do.
	return nil, &deliveryError{attempts: attempts, status: status, err: gotenberg.RedactURLError(err)}
}

// deliveryFailure gives err, the failure of a delivery, the client message of
// its [deliveryError] while keeping the status the API answers it with: 500,
// or that of a timeout or a filtered URL, which take precedence.
func deliveryFailure(err error) error {
	var delivery *deliveryError
	if !errors.As(err, &delivery) {
		return err
	}

	return api.WrapError(err, api.NewSentinelHttpError(http.StatusInternalServerError, delivery.message()))
}

// sendEvent sends a structured JSON event to the events URL. It is
// fire-and-forget: failures are logged but do not propagate.
func (c client) sendEvent(ctx context.Context, correlationIdHeader, correlationId string, event map[string]any) {
	b, err := json.Marshal(event)
	if err != nil {
		c.logger.ErrorContext(ctx, fmt.Sprintf("marshal webhook event: %s", err))
		return
	}

	eventsURL := gotenberg.RedactURL(c.eventsUrl)

	ctx, cancel := c.deliveryContext(ctx)
	defer cancel()

	ctx, span := startDeliverySpan(ctx, "webhook.send_event", c.eventsUrl)
	defer span.End()

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, c.eventsUrl, b)
	if err != nil {
		err = gotenberg.RedactURLError(err)
		_ = gotenberg.SetSpanStatus(span, err, gotenberg.ClassifyError(err))
		c.logger.ErrorContext(ctx, fmt.Sprintf("create webhook event request: %s", err))
		return
	}

	req.Header.Set("User-Agent", "Gotenberg")
	for key, value := range c.extraHttpHeaders {
		req.Header.Set(key, value)
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(correlationIdHeader, correlationId)
	req.Header.Set("Gotenberg-Version", gotenberg.Version)

	resp, err := c.client.Do(req)
	if err != nil {
		_ = gotenberg.SetSpanStatus(span, err, gotenberg.ClassifyError(err))
		c.logger.ErrorContext(ctx, fmt.Sprintf("send webhook event to '%s': %s", eventsURL, err))
		return
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			c.logger.ErrorContext(ctx, fmt.Sprintf("close response body from '%s': %s", eventsURL, err))
		}
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		err := fmt.Errorf("webhook event: got status '%s'", resp.Status)
		_ = gotenberg.SetSpanStatus(span, err, gotenberg.ClassifyError(err))
		c.logger.ErrorContext(ctx, fmt.Sprintf("send webhook event to '%s': got status '%s'", eventsURL, resp.Status))
		return
	}

	c.logger.InfoContext(ctx, fmt.Sprintf("webhook event sent to '%s'", eventsURL))
}
