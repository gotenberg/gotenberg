package xerrgroup

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"golang.org/x/sync/errgroup"
)

// Run runs all functions simultaneously and wait until
// execution has completed or an error is encountered.
func Run(fn ...func() error) error {
	const op string = "xerrgroup.Run"
	eg := errgroup.Group{}
	for _, f := range fn {
		eg.Go(f)
	}
	if err := eg.Wait(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}
