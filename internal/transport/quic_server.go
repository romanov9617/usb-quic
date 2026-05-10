package transport

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/quic-go/quic-go"

	"usb-quic/internal/tunnel"
)

// StreamHandler handles one accepted transport stream.
type StreamHandler interface {
	HandleStream(ctx context.Context, stream tunnel.Endpoint) error
}

// StreamHandlerFunc adapts a function to StreamHandler.
type StreamHandlerFunc func(ctx context.Context, stream tunnel.Endpoint) error

// HandleStream handles one accepted transport stream.
func (fn StreamHandlerFunc) HandleStream(ctx context.Context, stream tunnel.Endpoint) error {
	return fn(ctx, stream)
}

// ServeQUICListener accepts QUIC connections and dispatches their streams.
func ServeQUICListener(ctx context.Context, listener *quic.Listener, handler StreamHandler) error {
	ctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup

	defer func() {
		cancel()

		wg.Wait()
	}()

	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			if expectedQUICListenerClose(ctx, err) {
				return nil
			}

			return fmt.Errorf("accept quic connection: %w", err)
		}

		wg.Go(func() {
			_ = serveQUICConnection(ctx, conn, handler)
		})
	}
}

func serveQUICConnection(ctx context.Context, conn *quic.Conn, handler StreamHandler) error {
	var wg sync.WaitGroup

	defer func() {
		_ = conn.CloseWithError(0, "")

		wg.Wait()
	}()

	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			if expectedQUICConnectionClose(ctx) {
				return nil
			}

			return fmt.Errorf("accept quic stream: %w", err)
		}

		wg.Go(func() {
			_ = handler.HandleStream(ctx, NewQUICStreamEndpoint(stream))
		})
	}
}

func expectedQUICListenerClose(ctx context.Context, err error) bool {
	return ctx.Err() != nil || errors.Is(err, quic.ErrServerClosed)
}

func expectedQUICConnectionClose(ctx context.Context) bool {
	return ctx.Err() != nil
}
