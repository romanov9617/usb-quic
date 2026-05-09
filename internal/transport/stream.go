// Package transport contains stream-opening transport adapters.
package transport

import (
	"context"
	"fmt"
	"net"

	"usb-quic/internal/tunnel"
)

// StreamOpener opens one byte stream for one USB/IP TCP session.
type StreamOpener interface {
	OpenStream(ctx context.Context) (tunnel.Endpoint, error)
}

// OpenStreamFunc adapts a function to StreamOpener.
type OpenStreamFunc func(ctx context.Context) (tunnel.Endpoint, error)

// OpenStream opens one byte stream.
//
//nolint:ireturn // StreamOpener intentionally hides TCP and future QUIC endpoint types.
func (fn OpenStreamFunc) OpenStream(ctx context.Context) (tunnel.Endpoint, error) {
	return fn(ctx)
}

// TCPStreamOpener opens a TCP connection for each stream.
type TCPStreamOpener struct {
	address string
	dialer  net.Dialer
}

// NewTCPStreamOpener creates a stream opener backed by TCP dials.
func NewTCPStreamOpener(address string) *TCPStreamOpener {
	return &TCPStreamOpener{
		address: address,
		//nolint:exhaustruct // Default Dialer is sufficient for the TCP prototype transport.
		dialer: net.Dialer{},
	}
}

// OpenStream opens one TCP-backed byte stream.
//
//nolint:ireturn // StreamOpener intentionally hides TCP and future QUIC endpoint types.
func (opener *TCPStreamOpener) OpenStream(ctx context.Context) (tunnel.Endpoint, error) {
	conn, err := opener.dialer.DialContext(ctx, "tcp", opener.address)
	if err != nil {
		return nil, fmt.Errorf("dial tcp stream %s: %w", opener.address, err)
	}

	return tunnel.NewNetEndpoint(conn), nil
}
