package transport

import (
	"context"
	"testing"
	"time"

	"usb-quic/internal/tunnel"
)

func TestServeQUICListenerHandlesAcceptedStream(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	listener := listenQUIC(t)
	defer closeQUICListener(listener)

	accepted := make(chan tunnel.Endpoint, 1)
	errs := make(chan error, 1)

	go func() {
		errs <- ServeQUICListener(ctx, listener, StreamHandlerFunc(
			func(_ context.Context, stream tunnel.Endpoint) error {
				accepted <- stream

				return nil
			},
		))
	}()

	clientConn := dialQUIC(t, ctx, listener)
	defer closeQUICConn(clientConn)

	clientStream, err := clientConn.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open client stream: %v", err)
	}

	writeString(t, clientStream, "request")

	serverEndpoint := waitAcceptedStream(t, ctx, accepted, errs)

	if got := readString(t, serverEndpoint, len("request")); got != "request" {
		t.Fatalf("server read %q, want %q", got, "request")
	}

	writeString(t, serverEndpoint, "response")

	if got := readString(t, clientStream, len("response")); got != "response" {
		t.Fatalf("client read %q, want %q", got, "response")
	}

	err = serverEndpoint.CloseWrite()
	if err != nil {
		t.Fatalf("close server write: %v", err)
	}

	if got := readEOF(t, clientStream); got != "" {
		t.Fatalf("client read %q before EOF, want empty", got)
	}
}

func TestServeQUICListenerStopsOnClosedListener(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	listener := listenQUIC(t)
	errs := make(chan error, 1)

	go func() {
		errs <- ServeQUICListener(ctx, listener, StreamHandlerFunc(
			func(context.Context, tunnel.Endpoint) error {
				return nil
			},
		))
	}()

	closeQUICListener(listener)

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("serve quic listener: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("serve quic listener did not stop: %v", ctx.Err())
	}
}

//nolint:ireturn // Test helper returns the transport boundary interface under test.
func waitAcceptedStream(
	t *testing.T,
	ctx context.Context,
	accepted <-chan tunnel.Endpoint,
	errs <-chan error,
) tunnel.Endpoint {
	t.Helper()

	select {
	case stream := <-accepted:
		return stream
	case err := <-errs:
		t.Fatalf("serve quic listener: %v", err)
	case <-ctx.Done():
		t.Fatalf("accept stream timed out: %v", ctx.Err())
	}

	return nil
}
