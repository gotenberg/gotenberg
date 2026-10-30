package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// StopApps stops every [Ingress] first, then the other apps, each group
// concurrently. The other apps stop even when an Ingress fails to: ctx bounds
// the whole shutdown. stopped is called with each app that stopped without
// error.
func StopApps(ctx context.Context, apps []App, stopped func(App)) error {
	var ingresses, others []App
	for _, app := range apps {
		if _, ok := app.(Ingress); ok {
			ingresses = append(ingresses, app)
			continue
		}
		others = append(others, app)
	}

	return errors.Join(
		stopAll(ctx, ingresses, stopped),
		stopAll(ctx, others, stopped),
	)
}

func stopAll(ctx context.Context, apps []App, stopped func(App)) error {
	errs := make([]error, len(apps))

	var wg sync.WaitGroup
	for i, app := range apps {
		wg.Go(func() {
			err := app.Stop(ctx)
			if err != nil {
				errs[i] = fmt.Errorf("stop %s: %w", appID(app), err)
				return
			}
			stopped(app)
		})
	}
	wg.Wait()

	return errors.Join(errs...)
}

// appID returns the module ID of app, if any.
func appID(app App) string {
	mod, ok := app.(Module)
	if !ok {
		return fmt.Sprintf("%T", app)
	}

	return mod.Descriptor().ID
}
