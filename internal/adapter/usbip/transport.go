// Package usbip contains USB/IP transport adapters.
package usbip

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"sync"
	"time"

	"usb-quic/internal/adapter/logging"

	"github.com/quic-go/quic-go"
)

const proxyDirections = 2

// Transport proxies USB/IP TCP byte streams over QUIC streams.
type Transport struct {
	dialTimeout time.Duration
	logger      *logging.Logger
	tlsConfig   *tls.Config
	quicConfig  *quic.Config
}

// NewTransport creates a USB/IP TCP transport.
func NewTransport(options ...Option) *Transport {
	transport := &Transport{
		dialTimeout: defaultDialTimeout,
		logger:      logging.NewDefaultLogger(io.Discard),
		tlsConfig:   QUICClientTLS(),
		quicConfig:  QUICConfig(),
	}

	for _, option := range options {
		option(transport)
	}

	return transport
}

// ProxyTCPToQUIC listens for TCP connections and proxies each one into a QUIC stream.
func (transport *Transport) ProxyTCPToQUIC(ctx context.Context, tcpAddress, quicAddress url.URL) error {
	//nolint:exhaustruct // Zero-value ListenConfig uses Go's default TCP listener behavior.
	listenConfig := net.ListenConfig{}

	listener, err := listenConfig.Listen(
		ctx,
		"tcp",
		tcpAddress.Host,
	)
	if err != nil {
		return fmt.Errorf("listen usbip tcp %s: %w", tcpAddress.String(), err)
	}

	defer func() {
		_ = listener.Close()
	}()

	transport.logger.Info(
		"usbip tcp to quic proxy started",
		slog.String("address", listener.Addr().String()),
		slog.String("quic_address", quicAddress.String()),
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
			"usbip tcp connection accepted for quic proxy",
			slog.String("remote_addr", conn.RemoteAddr().String()),
		)

		go transport.proxyTCPConnectionToQUIC(ctx, conn, quicAddress)
	}
}

// ProxyQUICToTCP listens for QUIC streams and proxies each one into a TCP connection.
func (transport *Transport) ProxyQUICToTCP(ctx context.Context, quicAddress, tcpAddress url.URL) error {
	tlsConfig, err := QUICServerTLS()
	if err != nil {
		return fmt.Errorf("build quic server tls config: %w", err)
	}

	listener, err := quic.ListenAddr(quicAddress.Host, tlsConfig, transport.quicConfig)
	if err != nil {
		return fmt.Errorf("listen usbip quic %s: %w", quicAddress.String(), err)
	}

	defer func() {
		_ = listener.Close()
	}()

	transport.logger.Info(
		"usbip quic to tcp proxy started",
		slog.String("address", listener.Addr().String()),
		slog.String("tcp_address", tcpAddress.String()),
	)

	for {
		conn, acceptErr := listener.Accept(ctx)
		if acceptErr != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}

			return fmt.Errorf("accept usbip quic connection: %w", acceptErr)
		}

		transport.logger.Info(
			"usbip quic connection accepted",
			slog.String("remote_addr", conn.RemoteAddr().String()),
		)

		go transport.proxyQUICConnectionToTCP(ctx, conn, tcpAddress)
	}
}

func (transport *Transport) proxyTCPConnectionToQUIC(ctx context.Context, tcpConn net.Conn, quicAddress url.URL) {
	defer closeTCPConnection(tcpConn)

	conn, err := quic.DialAddr(ctx, quicAddress.Host, transport.tlsConfig, transport.quicConfig)
	if err != nil {
		transport.logger.Error(
			"dial usbip quic",
			slog.String("address", quicAddress.String()),
			slog.Any("error", err),
		)

		return
	}

	defer func() {
		_ = conn.CloseWithError(0, "")
	}()

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		transport.logger.Error("open usbip quic stream", slog.Any("error", err))

		return
	}

	defer closeQUICStream(stream)

	transport.logger.Info(
		"usbip proxy stream opened",
		slog.String("tcp_remote_addr", tcpConn.RemoteAddr().String()),
		slog.String("quic_remote_addr", conn.RemoteAddr().String()),
	)

	err = proxyBidirectional(tcpConn, stream)
	if err != nil {
		transport.logger.Debug("usbip proxy stream closed", slog.Any("error", err))
	}
}

func closeListenerOnCancel(ctx context.Context, listener net.Listener) {
	<-ctx.Done()

	_ = listener.Close()
}

func (transport *Transport) proxyQUICConnectionToTCP(ctx context.Context, quicConn *quic.Conn, tcpAddress url.URL) {
	defer func() {
		_ = quicConn.CloseWithError(0, "")
	}()

	for {
		stream, err := quicConn.AcceptStream(ctx)
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return
			}

			transport.logger.Debug("accept usbip quic stream", slog.Any("error", err))

			return
		}

		transport.logger.Info(
			"usbip quic stream accepted",
			slog.String("remote_addr", quicConn.RemoteAddr().String()),
		)

		go transport.proxyQUICStreamToTCP(ctx, stream, tcpAddress)
	}
}

func (transport *Transport) proxyQUICStreamToTCP(ctx context.Context, stream *quic.Stream, tcpAddress url.URL) {
	defer closeQUICStream(stream)

	//nolint:exhaustruct // Only Timeout differs from net.Dialer defaults.
	dialer := net.Dialer{
		Timeout: transport.dialTimeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", tcpAddress.Host)
	if err != nil {
		transport.logger.Error(
			"dial usbip tcp backend",
			slog.String("address", tcpAddress.String()),
			slog.Any("error", err),
		)

		return
	}

	defer closeTCPConnection(conn)

	err = proxyBidirectional(conn, stream)
	if err != nil {
		transport.logger.Debug("usbip backend proxy stream closed", slog.Any("error", err))
	}
}

func proxyBidirectional(tcpConn net.Conn, stream *quic.Stream) error {
	var wg sync.WaitGroup

	errs := make(chan error, proxyDirections)

	wg.Add(proxyDirections)

	go copyAndClose(&wg, errs, stream, tcpConn)
	go copyAndClose(&wg, errs, tcpConn, stream)

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

func copyAndClose(
	wg *sync.WaitGroup,
	errs chan<- error,
	writer io.Writer,
	reader io.Reader,
) {
	defer wg.Done()

	_, err := io.Copy(writer, reader)
	if err != nil && !isExpectedCopyError(err) {
		errs <- fmt.Errorf("copy proxied bytes: %w", err)
	}

	closeWriter(writer)
}

func closeWriter(writer io.Writer) {
	if closer, ok := writer.(interface{ CloseWrite() error }); ok {
		_ = closer.CloseWrite()

		return
	}

	if closer, ok := writer.(io.Closer); ok {
		_ = closer.Close()
	}
}

func closeTCPConnection(conn net.Conn) {
	_ = conn.Close()
}

func closeQUICStream(stream *quic.Stream) {
	_ = stream.Close()
}

func isExpectedCopyError(err error) bool {
	return errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF)
}
