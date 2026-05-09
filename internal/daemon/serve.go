package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"

	"usb-quic/internal/tunnel"
)

func (daemon *Daemon) serve(ctx context.Context, listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return fmt.Errorf("listener closed: %w", err)
			}

			return fmt.Errorf("accept tcp connection: %w", err)
		}

		daemon.log.logClientAccepted(conn.RemoteAddr())

		go daemon.handleConnection(ctx, conn)
	}
}

func (daemon *Daemon) handleConnection(ctx context.Context, client net.Conn) {
	defer func() {
		_ = client.Close()
	}()

	upstream, err := daemon.dial(ctx, daemon.cfg)
	if err != nil {
		daemon.log.logUpstreamDialFailed(client.RemoteAddr(), err)

		return
	}
	defer func() {
		_ = upstream.Close()
	}()

	daemon.log.logTunnelStarted(client.RemoteAddr())

	err = tunnel.BidirectionalCopy(ctx, newConnEndpoint(client), upstream)
	if err != nil && !errors.Is(err, context.Canceled) {
		daemon.log.logTunnelFailed(client.RemoteAddr(), err)

		return
	}

	daemon.log.logTunnelStopped(client.RemoteAddr())
}
