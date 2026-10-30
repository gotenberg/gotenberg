package scenario

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/cucumber/godog"
	"github.com/testcontainers/testcontainers-go"
)

// defaultContainers pools the containers "I have a default Gotenberg
// container" starts, so that a scenario reuses one an earlier scenario left
// instead of paying for a start. A pooled container serves one scenario at a
// time: concurrent scenarios never share one.
var defaultContainers struct {
	mu   sync.Mutex
	idle []testcontainers.Container
	all  []testcontainers.Container
}

// acquireDefaultContainer returns an idle pooled container, or starts one.
func acquireDefaultContainer(ctx context.Context) (testcontainers.Container, error) {
	defaultContainers.mu.Lock()
	if n := len(defaultContainers.idle); n > 0 {
		c := defaultContainers.idle[n-1]
		defaultContainers.idle = defaultContainers.idle[:n-1]
		defaultContainers.mu.Unlock()
		return c, nil
	}
	defaultContainers.mu.Unlock()

	c, err := startGotenbergContainer(ctx, nil)
	if err != nil {
		return nil, err
	}

	defaultContainers.mu.Lock()
	defer defaultContainers.mu.Unlock()
	defaultContainers.all = append(defaultContainers.all, c)

	return c, nil
}

// releaseDefaultContainer hands a pooled container back for another scenario.
func releaseDefaultContainer(c testcontainers.Container) {
	defaultContainers.mu.Lock()
	defer defaultContainers.mu.Unlock()
	defaultContainers.idle = append(defaultContainers.idle, c)
}

// discardDefaultContainer terminates a pooled container that must not serve
// another scenario.
func discardDefaultContainer(ctx context.Context, c testcontainers.Container) error {
	defaultContainers.mu.Lock()
	defaultContainers.all = slices.DeleteFunc(defaultContainers.all, func(o testcontainers.Container) bool { return o == c })
	defaultContainers.mu.Unlock()

	return c.Terminate(ctx, testcontainers.StopTimeout(0))
}

// TerminateDefaultContainers terminates every pooled container. Call it once a
// run of the suite ends.
func TerminateDefaultContainers(ctx context.Context) error {
	defaultContainers.mu.Lock()
	all := defaultContainers.all
	defaultContainers.all, defaultContainers.idle = nil, nil
	defaultContainers.mu.Unlock()

	var errs []error
	for _, c := range all {
		err := c.Terminate(ctx, testcontainers.StopTimeout(0))
		if err != nil {
			errs = append(errs, fmt.Errorf("terminate pooled Gotenberg container: %w", err))
		}
	}

	return errors.Join(errs...)
}

// isolatedSteps are the step texts whose assertions depend on a container no
// other scenario used: they read its logs, stop it, or load it on purpose.
var isolatedSteps = []string{
	"log the following entries",
	"I stop the Gotenberg container",
	"concurrent",
	"sequential",
}

// needsOwnContainer reports whether sc must not reuse, nor hand back, a pooled
// container: it is tagged @isolated or has one of [isolatedSteps].
func needsOwnContainer(sc *godog.Scenario) bool {
	for _, tag := range sc.Tags {
		if tag.Name == "@isolated" {
			return true
		}
	}

	for _, step := range sc.Steps {
		for _, text := range isolatedSteps {
			if strings.Contains(step.Text, text) {
				return true
			}
		}
	}

	return false
}
