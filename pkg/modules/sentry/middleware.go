package sentry

import (
	sentryecho "github.com/getsentry/sentry-go/echo"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func sentryMiddleware() api.Middleware {
	return api.Middleware{
		Stack: api.DefaultStack,
		Handler: sentryecho.New(sentryecho.Options{
			Repanic: true,
		}),
	}
}
