package usbip

import (
	"crypto/tls"
	"time"

	"github.com/quic-go/quic-go"

	"usb-quic/internal/adapter/logging"
)

// DefaultPort is the IANA-assigned USB/IP TCP port.
const DefaultPort = 3240

const defaultDialTimeout = 5 * time.Second

// Option configures Transport.
type Option func(*Transport)

// WithDialTimeout configures outgoing USB/IP TCP connection timeout.
func WithDialTimeout(timeout time.Duration) Option {
	return func(transport *Transport) {
		if timeout > 0 {
			transport.dialTimeout = timeout
		}
	}
}

// WithLogger configures structured transport logging.
func WithLogger(logger *logging.Logger) Option {
	return func(transport *Transport) {
		if logger != nil {
			transport.logger = logger
		}
	}
}

// WithTLSConfig configures QUIC TLS settings.
func WithTLSConfig(config *tls.Config) Option {
	return func(transport *Transport) {
		if config != nil {
			transport.tlsConfig = config
		}
	}
}

// WithQUICConfig configures QUIC transport settings.
func WithQUICConfig(config *quic.Config) Option {
	return func(transport *Transport) {
		if config != nil {
			transport.quicConfig = config
		}
	}
}
