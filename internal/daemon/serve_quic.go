package daemon

import (
	"context"
	"errors"
	"fmt"

	"github.com/quic-go/quic-go"

	"usb-quic/internal/transport"
	"usb-quic/internal/tunnel"
)

// ServeQUIC accepts QUIC streams and forwards each stream to the configured upstream.
func (daemon *Daemon) ServeQUIC(ctx context.Context, listener *quic.Listener) error {
	err := transport.ServeQUICListener(ctx, listener, transport.StreamHandlerFunc(daemon.handleStream))
	if err != nil {
		return fmt.Errorf("serve quic listener: %w", err)
	}

	return nil
}

func (daemon *Daemon) handleStream(ctx context.Context, stream tunnel.Endpoint) error {
	defer func() {
		_ = stream.Close()
	}()

	upstream, err := daemon.streamOpener.OpenStream(ctx)
	if err != nil {
		daemon.log.logStreamOpenFailed(nil, err)

		return fmt.Errorf("open upstream stream: %w", err)
	}
	defer func() {
		_ = upstream.Close()
	}()

	daemon.log.logTunnelStarted(nil)

	err = tunnel.BidirectionalCopy(ctx, stream, upstream)
	if err != nil && !errors.Is(err, context.Canceled) {
		daemon.log.logTunnelFailed(nil, err)

		return fmt.Errorf("copy quic stream: %w", err)
	}

	daemon.log.logTunnelStopped(nil)

	return nil
}
