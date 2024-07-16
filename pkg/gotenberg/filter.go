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

// FilterDeadline checks if given value is allowed and not denied according to
// regex patterns. It returns a [context.DeadlineExceeded] if it takes too long
// to process.
func FilterDeadline(allowed, denied *regexp2.Regexp, s string, deadline time.Time) error {
	// FIXME: not ideal to compile everytime, but is there another way to create a clone?
	if allowed.String() != "" {
		allow := regexp2.MustCompile(allowed.String(), 0)
		allow.MatchTimeout = time.Until(deadline)

		ok, err := allow.MatchString(s)
		if err != nil {
			if time.Now().After(deadline) {
				return context.DeadlineExceeded
			}
			return fmt.Errorf("'%s' cannot handle '%s': %w", allow.String(), s, err)
		}
		if !ok {
			return fmt.Errorf("'%s' does not match the expression from the allowed list: %w", s, ErrFiltered)
		}
	}

	if denied.String() != "" {
		deny := regexp2.MustCompile(denied.String(), 0)
		deny.MatchTimeout = time.Until(deadline)

		ok, err := deny.MatchString(s)
		if err != nil {
			if time.Now().After(deadline) {
				return context.DeadlineExceeded
			}
			return fmt.Errorf("'%s' cannot handle '%s': %w", deny.String(), s, err)
		}
		if ok {
			return fmt.Errorf("'%s' matches the expression from the denied list: %w", s, ErrFiltered)
		}
	}

	return nil
}
