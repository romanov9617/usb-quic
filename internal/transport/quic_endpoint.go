package transport

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/quic-go/quic-go"
)

const normalStreamClose quic.StreamErrorCode = 0

// QUICStreamEndpoint adapts a bidirectional QUIC stream to tunnel.Endpoint.
type QUICStreamEndpoint struct {
	stream         *quic.Stream
	closeReadOnce  sync.Once
	closeWriteOnce sync.Once
}

// NewQUICStreamEndpoint adapts stream to QUICStreamEndpoint.
func NewQUICStreamEndpoint(stream *quic.Stream) *QUICStreamEndpoint {
	return &QUICStreamEndpoint{
		stream:         stream,
		closeReadOnce:  sync.Once{},
		closeWriteOnce: sync.Once{},
	}
}

// Read reads bytes from the QUIC stream.
//
//nolint:wrapcheck // Read preserves the underlying stream error for io.Copy semantics.
func (endpoint *QUICStreamEndpoint) Read(buffer []byte) (int, error) {
	return endpoint.stream.Read(buffer)
}

// Write writes bytes to the QUIC stream.
//
//nolint:wrapcheck // Write preserves the underlying stream error for io.Copy semantics.
func (endpoint *QUICStreamEndpoint) Write(buffer []byte) (int, error) {
	return endpoint.stream.Write(buffer)
}

// Close cancels reads and closes writes on the QUIC stream.
func (endpoint *QUICStreamEndpoint) Close() error {
	return errors.Join(endpoint.CloseRead(), endpoint.CloseWrite())
}

// CloseRead cancels the read side of the QUIC stream.
func (endpoint *QUICStreamEndpoint) CloseRead() error {
	endpoint.closeReadOnce.Do(func() {
		endpoint.stream.CancelRead(normalStreamClose)
	})

	return nil
}

// CloseWrite closes the write side of the QUIC stream.
func (endpoint *QUICStreamEndpoint) CloseWrite() error {
	var err error

	endpoint.closeWriteOnce.Do(func() {
		err = endpoint.stream.Close()
	})

	if errors.Is(err, io.ErrClosedPipe) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("close quic stream write side: %w", err)
	}

	return nil
}
