package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"usb-quic/internal/transport"
	"usb-quic/internal/tunnel"
)

func TestHandleStreamCopiesBytesBothDirections(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	quicPeer, quicStream := newMemoryConnPair()
	upstreamPeer, upstreamConn := newMemoryConnPair()
	daemon := testDaemon(transport.OpenStreamFunc(func(context.Context) (tunnel.Endpoint, error) {
		return upstreamConn, nil
	}))
	errs := make(chan error, 1)

	go func() {
		errs <- daemon.handleStream(ctx, quicStream)
	}()

	writeMemoryString(t, quicPeer, "request")

	if got := readMemoryString(t, upstreamPeer, len("request")); got != "request" {
		t.Fatalf("upstream read %q, want %q", got, "request")
	}

	writeMemoryString(t, upstreamPeer, "response")

	if got := readMemoryString(t, quicPeer, len("response")); got != "response" {
		t.Fatalf("quic peer read %q, want %q", got, "response")
	}

	closeMemoryWrite(t, quicPeer)
	closeMemoryWrite(t, upstreamPeer)

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("handle stream: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("handler did not stop")
	}
}

func TestHandleStreamReturnsUpstreamOpenError(t *testing.T) {
	t.Parallel()

	quicPeer, quicStream := newMemoryConnPair()
	defer func() {
		_ = quicPeer.Close()
	}()

	daemon := testDaemon(transport.OpenStreamFunc(func(context.Context) (tunnel.Endpoint, error) {
		return nil, errBusy
	}))

	err := daemon.handleStream(t.Context(), quicStream)
	if !errors.Is(err, errBusy) {
		t.Fatalf("handle stream err=%v, want %v", err, errBusy)
	}
}
