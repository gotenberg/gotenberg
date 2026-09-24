package gotenberg

import "errors"

// ErrCancelGracefulShutdownContext tells that a module wants to abort a
// graceful shutdown and stops Gotenberg right away as there are no more
// ongoing processes.
var ErrCancelGracefulShutdownContext = errors.New("cancel graceful shutdown's context")
