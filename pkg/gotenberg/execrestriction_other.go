//go:build !linux

package gotenberg

import "fmt"

// RestrictExec forbids the process, and every process it starts afterward,
// to execute any file but paths and the files beneath them. It relies on
// Landlock, so outside Linux it always fails with
// [ErrExecRestrictionUnavailable].
func RestrictExec(_ []string) error {
	return fmt.Errorf("%w: Landlock requires Linux", ErrExecRestrictionUnavailable)
}
