// Package usbip contains USB/IP transport adapters.
package usbip

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	"usb-quic/internal/adapter/logging"
)

const quicALPN = "usb-quic"

const (
	proxyDirections        = 2
	tlsPrivateKeyBits      = 2048
	tlsSerialNumberBits    = 128
	tlsCertificateLifetime = 24 * time.Hour
)

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
		tlsConfig:   newClientTLSConfig(),
		quicConfig:  newQUICConfig(),
	}

	for _, option := range options {
		option(transport)
	}

	return transport
}

// ProxyTCPToQUIC listens for TCP connections and proxies each one into a QUIC stream.
func (transport *Transport) ProxyTCPToQUIC(ctx context.Context, tcpAddress, quicAddress string) error {
	//nolint:exhaustruct // Zero-value ListenConfig uses Go's default TCP listener behavior.
	listenConfig := net.ListenConfig{}

	listener, err := listenConfig.Listen(
		ctx,
		"tcp",
		tcpAddress,
	)
	if err != nil {
		return fmt.Errorf("listen usbip tcp %s: %w", tcpAddress, err)
	}

	defer func() {
		_ = listener.Close()
	}()

	transport.logger.Info(
		"usbip tcp to quic proxy started",
		slog.String("address", listener.Addr().String()),
		slog.String("quic_address", quicAddress),
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
func (transport *Transport) ProxyQUICToTCP(ctx context.Context, quicAddress, tcpAddress string) error {
	tlsConfig, err := newServerTLSConfig()
	if err != nil {
		return fmt.Errorf("build quic server tls config: %w", err)
	}

	listener, err := quic.ListenAddr(quicAddress, tlsConfig, transport.quicConfig)
	if err != nil {
		return fmt.Errorf("listen usbip quic %s: %w", quicAddress, err)
	}

	defer func() {
		_ = listener.Close()
	}()

	transport.logger.Info(
		"usbip quic to tcp proxy started",
		slog.String("address", listener.Addr().String()),
		slog.String("tcp_address", tcpAddress),
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

func (transport *Transport) proxyTCPConnectionToQUIC(ctx context.Context, tcpConn net.Conn, quicAddress string) {
	defer closeTCPConnection(tcpConn)

	conn, err := quic.DialAddr(ctx, quicAddress, transport.tlsConfig, transport.quicConfig)
	if err != nil {
		transport.logger.Error(
			"dial usbip quic",
			slog.String("address", quicAddress),
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

func (transport *Transport) proxyQUICConnectionToTCP(ctx context.Context, quicConn *quic.Conn, tcpAddress string) {
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

func (transport *Transport) proxyQUICStreamToTCP(ctx context.Context, stream *quic.Stream, tcpAddress string) {
	defer closeQUICStream(stream)

	//nolint:exhaustruct // Only Timeout differs from net.Dialer defaults.
	dialer := net.Dialer{
		Timeout: transport.dialTimeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", tcpAddress)
	if err != nil {
		transport.logger.Error(
			"dial usbip tcp backend",
			slog.String("address", tcpAddress),
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

func copyAndClose(wg *sync.WaitGroup, errs chan<- error, writer io.Writer, reader io.Reader) {
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

func newClientTLSConfig() *tls.Config {
	//nolint:exhaustruct // Only QUIC ALPN and development certificate verification policy differ from defaults.
	return &tls.Config{
		NextProtos:         []string{quicALPN},
		InsecureSkipVerify: true, //nolint:gosec // Local development proxy uses ephemeral self-signed QUIC certificates.
	}
}

func newServerTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, tlsPrivateKeyBits)
	if err != nil {
		return nil, fmt.Errorf("generate tls key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), tlsSerialNumberBits)

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate tls serial number: %w", err)
	}

	now := time.Now()
	//nolint:exhaustruct // Self-signed development certificate only needs fields used by TLS handshake.
	template := x509.Certificate{
		SerialNumber: serialNumber,
		//nolint:exhaustruct // CommonName is sufficient for this self-signed development certificate.
		Subject: pkix.Name{
			CommonName: "usb-quic",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(tlsCertificateLifetime),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create tls certificate: %w", err)
	}

	//nolint:exhaustruct // PEM certificate block does not require headers.
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})
	//nolint:exhaustruct // PEM key block does not require headers.
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load tls key pair: %w", err)
	}

	//nolint:exhaustruct // Only certificate and QUIC ALPN differ from TLS defaults.
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{quicALPN},
	}, nil
}

func newQUICConfig() *quic.Config {
	//nolint:exhaustruct // Only datagram support is explicitly controlled for byte-stream proxying.
	return &quic.Config{
		EnableDatagrams: false,
	}
}
