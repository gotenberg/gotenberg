package webhook

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	flag "github.com/spf13/pflag"
	"go.uber.org/multierr"
)

func init() {
	gotenberg.MustRegisterModule(Webhook{})
}

// Webhook is a module which provides a middleware for uploading output files
// to any destinations in an asynchronous fashion.
type Webhook struct {
	allowList      *regexp.Regexp
	denyList       *regexp.Regexp
	errorAllowList *regexp.Regexp
	errorDenyList  *regexp.Regexp
	maxRetry       int
	retryMinWait   time.Duration
	retryMaxWait   time.Duration
	clientTimeout  time.Duration
	disable        bool
}

// Descriptor returns an Webhook's module descriptor.
func (Webhook) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "webhook",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("webhook", flag.ExitOnError)
			// Deprecated flags.
			fs.String("api-webhook-allow-list", "", "Set the allowed URLs for the webhook feature using a regular expression")
			fs.String("api-webhook-deny-list", "", "Set the denied URLs for the webhook feature using a regular expression")
			fs.String("api-webhook-error-allow-list", "", "Set the allowed URLs in case of an error for the webhook feature using a regular expression")
			fs.String("api-webhook-error-deny-list", "", "Set the denied URLs in case of an error for the webhook feature using a regular expression")
			fs.Int("api-webhook-max-retry", 4, "Set the maximum number of retries for the webhook feature")
			fs.Duration("api-webhook-retry-min-wait", time.Duration(1)*time.Second, "Set the minimum duration to wait before trying to call the webhook again")
			fs.Duration("api-webhook-retry-max-wait", time.Duration(30)*time.Second, "Set the maximum duration to wait before trying to call the webhook again")
			fs.Bool("api-disable-webhook", false, "Disable the webhook feature")

			var err error
			err = multierr.Append(err, fs.MarkDeprecated("api-webhook-allow-list", "use webhook-allow-list instead"))
			err = multierr.Append(err, fs.MarkDeprecated("api-webhook-deny-list", "use webhook-deny-list instead"))
			err = multierr.Append(err, fs.MarkDeprecated("api-webhook-error-allow-list", "use webhook-error-allow-list instead"))
			err = multierr.Append(err, fs.MarkDeprecated("api-webhook-error-deny-list", "use webhook-error-deny-list instead"))
			err = multierr.Append(err, fs.MarkDeprecated("api-webhook-max-retry", "use webhook-max-retry instead"))
			err = multierr.Append(err, fs.MarkDeprecated("api-webhook-retry-min-wait", "use webhook-retry-min-wait instead"))
			err = multierr.Append(err, fs.MarkDeprecated("api-webhook-retry-max-wait", "use webhook-retry-max-wait instead"))
			err = multierr.Append(err, fs.MarkDeprecated("api-disable-webhook", "use webhook-disable instead"))

			if err != nil {
				panic(fmt.Errorf("create deprecated flags for the webhook module: %v", err))
			}

			// New flags.
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
	w.allowList = flags.MustDeprecatedRegexp("api-webhook-allow-list", "webhook-allow-list")
	w.denyList = flags.MustDeprecatedRegexp("api-webhook-deny-list", "webhook-deny-list")
	w.errorAllowList = flags.MustDeprecatedRegexp("api-webhook-error-allow-list", "webhook-error-allow-list")
	w.errorDenyList = flags.MustDeprecatedRegexp("api-webhook-error-deny-list", "webhook-error-deny-list")
	w.maxRetry = flags.MustDeprecatedInt("api-webhook-max-retry", "webhook-max-retry")
	w.retryMinWait = flags.MustDeprecatedDuration("api-webhook-retry-min-wait", "webhook-retry-min-wait")
	w.retryMaxWait = flags.MustDeprecatedDuration("api-webhook-retry-min-wait", "webhook-retry-max-wait")
	w.clientTimeout = flags.MustDuration("webhook-client-timeout")
	w.disable = flags.MustDeprecatedBool("api-disable-webhook", "webhook-disable")

	return nil
}

// Middlewares returns the middleware.
func (w Webhook) Middlewares() ([]api.Middleware, error) {
	if w.disable {
		return nil, nil
	}

	return []api.Middleware{
		webhookMiddleware(w),
	}, nil
}

// AddGraceDuration increases the grace duration provided by the API for the
// garbage collector.
func (w Webhook) AddGraceDuration() time.Duration {
	var duration time.Duration

	if w.disable {
		return duration
	}

	for i := 0; i < w.maxRetry; i++ {
		// Yep... Golang does not allow int * time.Duration.
		duration += w.retryMaxWait
	}

	return duration
}

// Interface guards.
var (
	_ gotenberg.Module                             = (*Webhook)(nil)
	_ gotenberg.Provisioner                        = (*Webhook)(nil)
	_ api.MiddlewareProvider                       = (*Webhook)(nil)
	_ api.GarbageCollectorGraceDurationIncrementer = (*Webhook)(nil)
)
