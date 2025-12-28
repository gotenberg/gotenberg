package webhook

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dlclark/regexp2"
	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(Webhook))
}

// Webhook is a module that provides a middleware for uploading output files
// to any destinations in an asynchronous fashion.
type Webhook struct {
	enableSyncMode bool
	allowList      *regexp2.Regexp
	denyList       *regexp2.Regexp
	errorAllowList *regexp2.Regexp
	errorDenyList  *regexp2.Regexp
	maxRetry       int
	retryMinWait   time.Duration
	retryMaxWait   time.Duration
	clientTimeout  time.Duration
	asyncCount     atomic.Int64
	disable        bool

	tracer gotenberg.TracerProvider
}

// Descriptor returns an [Webhook]'s module descriptor.
func (w *Webhook) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "webhook",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("webhook", flag.ExitOnError)
			fs.Bool("webhook-enable-sync-mode", false, "Enable synchronous mode for the webhook feature")
			fs.String("webhook-allow-list", "", "Set the allowed URLs for the webhook feature using a regular expression")
			fs.String("webhook-deny-list", "", "Set the denied URLs for the webhook feature using a regular expression")
			fs.String("webhook-error-allow-list", "", "Set the allowed URLs in case of an error for the webhook feature using a regular expression")
			fs.String("webhook-error-deny-list", "", "Set the denied URLs in case of an error for the webhook feature using a regular expression")
			fs.Int("webhook-max-retry", 4, "Set the maximum number of retries for the webhook feature")
			fs.Duration("webhook-retry-min-wait", time.Duration(1)*time.Second, "Set the minimum duration to wait before trying to call the webhook again")
			fs.Duration("webhook-retry-max-wait", time.Duration(30)*time.Second, "Set the maximum duration to wait before trying to call the webhook again")
			fs.Duration("webhook-client-timeout", time.Duration(30)*time.Second, "Set the time limit for requests to the webhook")
			fs.Bool("webhook-disable", false, "Disable the webhook feature")

			return fs
		}(),
		New: func() gotenberg.Module { return new(Webhook) },
	}
}

// Provision sets the module properties.
func (w *Webhook) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	w.enableSyncMode = flags.MustBool("webhook-enable-sync-mode")
	w.allowList = flags.MustRegexp("webhook-allow-list")
	w.denyList = flags.MustRegexp("webhook-deny-list")
	w.errorAllowList = flags.MustRegexp("webhook-error-allow-list")
	w.errorDenyList = flags.MustRegexp("webhook-error-deny-list")
	w.maxRetry = flags.MustInt("webhook-max-retry")
	w.retryMinWait = flags.MustDuration("webhook-retry-min-wait")
	w.retryMaxWait = flags.MustDuration("webhook-retry-max-wait")
	w.clientTimeout = flags.MustDuration("webhook-client-timeout")
	w.disable = flags.MustBool("webhook-disable")
	w.asyncCount.Store(0)

	// Tracer.
	tracerProvider, err := ctx.Module(new(gotenberg.TracerProvider))
	if err != nil {
		return fmt.Errorf("get tracer provider: %w", err)
	}
	w.tracer = tracerProvider.(gotenberg.TracerProvider)

	return nil
}

// Middlewares returns the middleware.
func (w *Webhook) Middlewares() ([]api.Middleware, error) {
	if w.disable {
		return nil, nil
	}

	return []api.Middleware{
		webhookMiddleware(w),
	}, nil
}

// AsyncCount returns the number of asynchronous requests.
func (w *Webhook) AsyncCount() int64 {
	return w.asyncCount.Load()
}

// Interface guards.
var (
	_ gotenberg.Module        = (*Webhook)(nil)
	_ gotenberg.Provisioner   = (*Webhook)(nil)
	_ api.MiddlewareProvider  = (*Webhook)(nil)
	_ api.AsynchronousCounter = (*Webhook)(nil)
)
