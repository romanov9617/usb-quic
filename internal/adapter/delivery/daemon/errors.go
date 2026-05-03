package daemon

import "errors"

// ErrNotImplemented marks daemon behavior that is not wired yet.
var ErrNotImplemented = errors.New("daemon is not implemented yet")
