package webhook

import (
	"time"

	"github.com/dlclark/regexp2"
	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(Webhook))
}

// Webhook is a module which provides a middleware for uploading output files
// to any destinations in an asynchronous fashion.
type Webhook struct {
	allowList      *regexp2.Regexp
	denyList       *regexp2.Regexp
	errorAllowList *regexp2.Regexp
	errorDenyList  *regexp2.Regexp
	maxRetry       int
	retryMinWait   time.Duration
	retryMaxWait   time.Duration
	clientTimeout  time.Duration
	disable        bool
}

// Descriptor returns an [Webhook]'s module descriptor.
func (w *Webhook) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "webhook",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("webhook", flag.ExitOnError)
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
	w.allowList = flags.MustRegexp("webhook-allow-list")
	w.denyList = flags.MustRegexp("webhook-deny-list")
	w.errorAllowList = flags.MustRegexp("webhook-error-allow-list")
	w.errorDenyList = flags.MustRegexp("webhook-error-deny-list")
	w.maxRetry = flags.MustInt("webhook-max-retry")
	w.retryMinWait = flags.MustDuration("webhook-retry-min-wait")
	w.retryMaxWait = flags.MustDuration("webhook-retry-max-wait")
	w.clientTimeout = flags.MustDuration("webhook-client-timeout")
	w.disable = flags.MustBool("webhook-disable")

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

// Interface guards.
var (
	_ gotenberg.Module       = (*Webhook)(nil)
	_ gotenberg.Provisioner  = (*Webhook)(nil)
	_ api.MiddlewareProvider = (*Webhook)(nil)
)
