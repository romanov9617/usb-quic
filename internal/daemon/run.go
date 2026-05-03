// Package daemon contains daemon runtime behavior.
package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"

	"usb-quic/internal/adapter/logging"
	"usb-quic/internal/config"
)

const pidFileMode = 0o600

type listenFunc func(ctx context.Context, cfg config.Daemon) (net.Listener, error)

// Run starts the daemon lifecycle and blocks until ctx is canceled.
func Run(ctx context.Context, cfg config.Daemon, logger *logging.Logger) error {
	return runWithListener(ctx, cfg, logger, listen)
}

func runWithListener(ctx context.Context, cfg config.Daemon, logger *logging.Logger, listen listenFunc) error {
	logger = normalizedLogger(logger)

	logger.Info(
		"daemon starting",
		slog.Bool("bind_ipv4", cfg.BindIPv4),
		slog.Bool("bind_ipv6", cfg.BindIPv6),
		slog.Bool("device_mode", cfg.DeviceMode),
		slog.Bool("daemonize", cfg.Daemonize),
		slog.Bool("debug", cfg.Debug),
		slog.String("pid_file", cfg.PIDFile),
		slog.Int("tcp_port", cfg.TCPPort),
	)

	listener, err := listen(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeListener(listener)

	cleanup, err := writePIDFile(cfg.PIDFile)
	if err != nil {
		return err
	}
	defer cleanup()

	<-ctx.Done()

	logger.Info("daemon stopped", slog.Any("reason", ctx.Err()))

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

func normalizedLogger(logger *logging.Logger) *logging.Logger {
	if logger != nil {
		return logger
	}

	return logging.NewDefaultLogger(os.Stderr)
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
