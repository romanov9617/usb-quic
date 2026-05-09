package daemon

import "net"

type options struct {
	listener net.Listener
}

// Option configures Daemon.
type Option func(opts *options)

// WithListener configures Daemon to use listener instead of opening one.
func WithListener(listener net.Listener) Option {
	return func(opts *options) {
		opts.listener = listener
	}
}
