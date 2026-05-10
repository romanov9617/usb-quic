package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"

	"github.com/quic-go/quic-go"

	"usb-quic/internal/tunnel"
)

// QUICStreamOpener opens one bidirectional QUIC stream per USB/IP TCP session.
type QUICStreamOpener struct {
	conn *quic.Conn
}

// NewQUICStreamOpener creates a stream opener backed by an existing QUIC connection.
func NewQUICStreamOpener(conn *quic.Conn) *QUICStreamOpener {
	return &QUICStreamOpener{
		conn: conn,
	}
}

// OpenStream opens one bidirectional QUIC stream.
//
//nolint:ireturn // StreamOpener intentionally hides TCP and QUIC endpoint types.
func (opener *QUICStreamOpener) OpenStream(ctx context.Context) (tunnel.Endpoint, error) {
	stream, err := opener.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("open quic stream: %w", err)
	}

	return NewQUICStreamEndpoint(stream), nil
}

// QUICDialStreamOpener keeps one QUIC connection and opens streams on demand.
type QUICDialStreamOpener struct {
	address    string
	tlsConfig  *tls.Config
	quicConfig *quic.Config
	mu         sync.Mutex
	conn       *quic.Conn
}

// NewQUICDialStreamOpener creates a lazy QUIC stream opener.
func NewQUICDialStreamOpener(address string, tlsConfig *tls.Config, quicConfig *quic.Config) *QUICDialStreamOpener {
	return &QUICDialStreamOpener{
		address:    address,
		tlsConfig:  tlsConfig,
		quicConfig: quicConfig,
		mu:         sync.Mutex{},
		conn:       nil,
	}
}

// OpenStream opens one stream on a reused QUIC connection.
//
//nolint:ireturn // StreamOpener intentionally hides TCP and QUIC endpoint types.
func (opener *QUICDialStreamOpener) OpenStream(ctx context.Context) (tunnel.Endpoint, error) {
	conn, err := opener.connection(ctx)
	if err != nil {
		return nil, err
	}

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		opener.forgetConnection(conn)

		return nil, fmt.Errorf("open quic stream: %w", err)
	}

	return NewQUICStreamEndpoint(stream), nil
}

// Close closes the cached QUIC connection, if one has been opened.
func (opener *QUICDialStreamOpener) Close() error {
	opener.mu.Lock()
	conn := opener.conn
	opener.conn = nil
	opener.mu.Unlock()

	if conn == nil {
		return nil
	}

	err := conn.CloseWithError(0, "")
	if errors.Is(err, quic.ErrServerClosed) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("close quic connection: %w", err)
	}

	return nil
}

func (opener *QUICDialStreamOpener) connection(ctx context.Context) (*quic.Conn, error) {
	opener.mu.Lock()
	defer opener.mu.Unlock()

	if opener.conn != nil {
		return opener.conn, nil
	}

	conn, err := quic.DialAddr(ctx, opener.address, opener.tlsConfig, opener.quicConfig)
	if err != nil {
		return nil, fmt.Errorf("dial quic %s: %w", opener.address, err)
	}

	opener.conn = conn

	return conn, nil
}

func (opener *QUICDialStreamOpener) forgetConnection(conn *quic.Conn) {
	opener.mu.Lock()
	defer opener.mu.Unlock()

	if opener.conn == conn {
		opener.conn = nil
	}
}
