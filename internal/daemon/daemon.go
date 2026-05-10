// Package daemon contains daemon runtime behavior.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"

	"usb-quic/internal/adapter/logging"
	"usb-quic/internal/config"
	domainusbip "usb-quic/internal/domain/usbip"
	"usb-quic/internal/transport"
)

const pidFileMode = 0o600

type listenFunc func(ctx context.Context, cfg config.Daemon) (net.Listener, error)

// Daemon contains daemon runtime state and dependencies.
type Daemon struct {
	cfg          config.Daemon
	log          *logger
	listen       listenFunc
	streamOpener transport.StreamOpener
}

// New creates a daemon runtime.
func New(cfg config.Daemon, logger *logging.Logger, opts ...Option) *Daemon {
	options := options{
		listener:     nil,
		streamOpener: nil,
	}
	for _, opt := range opts {
		opt(&options)
	}

	daemon := &Daemon{
		cfg:          cfg,
		log:          newLogger(logger),
		listen:       listen,
		streamOpener: transport.NewTCPStreamOpener(defaultUpstreamAddress()),
	}

	if options.listener != nil {
		daemon.listen = func(context.Context, config.Daemon) (net.Listener, error) {
			return options.listener, nil
		}
	}

	if options.streamOpener != nil {
		daemon.streamOpener = options.streamOpener
	}

	return daemon
}

// Run starts the daemon runtime and blocks until ctx is canceled.
func (daemon *Daemon) Run(ctx context.Context) error {
	daemon.log.logDaemonStarting(daemon.cfg)

	listener, err := daemon.listen(ctx, daemon.cfg)
	if err != nil {
		return err
	}
	defer closeListener(listener)

	cleanup, err := writePIDFile(daemon.cfg.PIDFile)
	if err != nil {
		return err
	}
	defer cleanup()

	var wg sync.WaitGroup

	errs := make(chan error, 1)

	go func() {
		errs <- daemon.serve(ctx, listener, &wg)
	}()

	<-ctx.Done()
	closeListener(listener)

	err = <-errs
	if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, context.Canceled) {
		return err
	}

	wg.Wait()
	daemon.log.logDaemonStopped(ctx.Err())

	return nil
}

func listen(ctx context.Context, cfg config.Daemon) (net.Listener, error) {
	address := listenAddress(cfg)

	//nolint:exhaustruct // Default ListenConfig is sufficient for daemon lifecycle binding.
	listenConfig := net.ListenConfig{}

	listener, err := listenConfig.Listen(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen tcp %s: %w", address, err)
	}

	return listener, nil
}

func defaultUpstreamAddress() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(domainusbip.DefaultPort))
}

func listenAddress(cfg config.Daemon) string {
	host := ""
	if cfg.BindIPv4 && !cfg.BindIPv6 {
		host = "0.0.0.0"
	}

	if cfg.BindIPv6 && !cfg.BindIPv4 {
		host = "::"
	}

	return net.JoinHostPort(host, strconv.Itoa(cfg.TCPPort))
}

func closeListener(listener net.Listener) {
	_ = listener.Close()
}

func writePIDFile(path string) (func(), error) {
	if path == "" {
		return func() {}, nil
	}

	err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), pidFileMode)
	if err != nil {
		return nil, fmt.Errorf("write pid file %s: %w", path, err)
	}

	return func() {
		_ = os.Remove(path)
	}, nil
}
