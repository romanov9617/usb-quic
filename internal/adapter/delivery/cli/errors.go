package cli

import "errors"

// ErrNotImplemented marks commands whose USB/IP behavior is not wired yet.
var ErrNotImplemented = errors.New("command is not implemented yet")
