// Package usbip contains USB/IP transport adapters.
package usbip

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"
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
func WithLogger(logger *slog.Logger) Option {
	return func(transport *Transport) {
		if logger != nil {
			transport.logger = logger
		}
	}
}

// Endpoint identifies a USB/IP TCP endpoint.
type Endpoint struct {
	Host string
	Port int
}

// Address returns the network address for endpoint.
func (endpoint Endpoint) Address() string {
	return net.JoinHostPort(
		endpoint.Host,
		strconv.Itoa(endpoint.Port),
	)
}

// ConnectionHandler handles an accepted USB/IP TCP connection.
type ConnectionHandler interface {
	HandleUSBIPConnection(ctx context.Context, conn net.Conn) error
}

// ConnectionHandlerFunc adapts a function to ConnectionHandler.
type ConnectionHandlerFunc func(ctx context.Context, conn net.Conn) error

// HandleUSBIPConnection handles an accepted USB/IP TCP connection.
func (handler ConnectionHandlerFunc) HandleUSBIPConnection(ctx context.Context, conn net.Conn) error {
	return handler(ctx, conn)
}

// Transport listens for and opens USB/IP TCP connections.
type Transport struct {
	dialTimeout time.Duration
	logger      *slog.Logger
}

// NewTransport creates a USB/IP TCP transport.
func NewTransport(options ...Option) *Transport {
	transport := &Transport{
		dialTimeout: defaultDialTimeout,
		logger:      newDiscardLogger(),
	}

	for _, option := range options {
		option(transport)
	}

	return transport
}

// Listen accepts USB/IP TCP connections until ctx is canceled.
func (transport *Transport) Listen(ctx context.Context, address string, handler ConnectionHandler) error {
	//nolint:exhaustruct // Zero-value ListenConfig uses Go's default TCP listener behavior.
	listenConfig := net.ListenConfig{}

	listener, err := listenConfig.Listen(
		ctx,
		"tcp",
		address,
	)
	if err != nil {
		return fmt.Errorf("listen usbip tcp %s: %w", address, err)
	}

	defer func() {
		_ = listener.Close()
	}()

	transport.logger.Info(
		"usbip tcp listener started",
		slog.String("address", listener.Addr().String()),
	)

	go closeListenerOnCancel(ctx, listener)

	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}

			return fmt.Errorf("accept usbip tcp connection: %w", acceptErr)
		}

		transport.logger.Info(
			"usbip tcp connection accepted",
			slog.String("remote_addr", conn.RemoteAddr().String()),
		)

		go handleConnection(ctx, conn, handler)
	}
}

// Dial opens an outgoing USB/IP TCP connection.
func (transport *Transport) Dial(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
	//nolint:exhaustruct // Only Timeout differs from net.Dialer defaults.
	dialer := net.Dialer{
		Timeout: transport.dialTimeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", endpoint.Address())
	if err != nil {
		return nil, fmt.Errorf("dial usbip tcp %s: %w", endpoint.Address(), err)
	}

	transport.logger.Info(
		"usbip tcp connection opened",
		slog.String("remote_addr", conn.RemoteAddr().String()),
	)

	return conn, nil
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func closeListenerOnCancel(ctx context.Context, listener net.Listener) {
	<-ctx.Done()

	_ = listener.Close()
}

func handleConnection(ctx context.Context, conn net.Conn, handler ConnectionHandler) {
	defer func() {
		_ = conn.Close()
	}()

	_ = handler.HandleUSBIPConnection(ctx, conn)
}
