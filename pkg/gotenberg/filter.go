package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dlclark/regexp2"
)

// ErrFiltered happens if a value is filtered by the [FilterDeadline] function.
var ErrFiltered = errors.New("value filtered")

// FilterDeadline checks if the given value is allowed and not denied according
// to regex patterns. The allowed list uses OR semantics (value must match at
// least one pattern). The denied list uses OR semantics (value is denied if it
// matches any pattern). It returns a [context.DeadlineExceeded] if it takes
// too long to process.
func FilterDeadline(allowed, denied []*regexp2.Regexp, s string, deadline time.Time) error {
	if len(allowed) > 0 {
		matched := false

		for _, pattern := range allowed {
			// FIXME: not ideal to compile everytime, but is there another way to create a clone?
			clone := regexp2.MustCompile(pattern.String(), 0)
			clone.MatchTimeout = time.Until(deadline)

			ok, err := clone.MatchString(s)
			if err != nil {
				if time.Now().After(deadline) {
					return context.DeadlineExceeded
				}

				return fmt.Errorf("'%s' cannot handle '%s': %w", clone.String(), s, err)
			}

			if ok {
				matched = true
				break
			}
		}

		if !matched {
			return fmt.Errorf("'%s' does not match any expression from the allowed list: %w", s, ErrFiltered)
		}
	}

	if len(denied) > 0 {
		for _, pattern := range denied {
			clone := regexp2.MustCompile(pattern.String(), 0)
			clone.MatchTimeout = time.Until(deadline)

			ok, err := clone.MatchString(s)
			if err != nil {
				if time.Now().After(deadline) {
					return context.DeadlineExceeded
				}

				return fmt.Errorf("'%s' cannot handle '%s': %w", clone.String(), s, err)
			}

			if ok {
				return fmt.Errorf("'%s' matches the expression from the denied list: %w", s, ErrFiltered)
			}
		}
	}

	return nil
}
