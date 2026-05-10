package daemon

import (
	"net"

	"usb-quic/internal/transport"
)

type options struct {
	listener     net.Listener
	streamOpener transport.StreamOpener
}

// Option configures Daemon.
type Option func(opts *options)

// WithListener configures Daemon to use listener instead of opening one.
func WithListener(listener net.Listener) Option {
	return func(opts *options) {
		opts.listener = listener
	}
}

// WithStreamOpener configures Daemon to open one stream per accepted connection.
func WithStreamOpener(streamOpener transport.StreamOpener) Option {
	return func(opts *options) {
		opts.streamOpener = streamOpener
	}
}
