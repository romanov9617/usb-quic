package daemon

import (
	"log/slog"
	"os"
	"sync/atomic"

	"usb-quic/internal/adapter/logging"
	"usb-quic/internal/config"
)

type logger struct {
	logger atomic.Pointer[slog.Logger]
}

func newLogger(base *logging.Logger) *logger {
	if base == nil {
		base = logging.NewDefaultLogger(os.Stderr)
	}

	log := &logger{
		logger: atomic.Pointer[slog.Logger]{},
	}
	log.logger.Store(base)

	return log
}

func (log *logger) logDaemonStarting(cfg config.Daemon) {
	log.load().Info(
		"daemon starting",
		slog.Bool("bind_ipv4", cfg.BindIPv4),
		slog.Bool("bind_ipv6", cfg.BindIPv6),
		slog.Bool("device_mode", cfg.DeviceMode),
		slog.Bool("daemonize", cfg.Daemonize),
		slog.Bool("debug", cfg.Debug),
		slog.String("pid_file", cfg.PIDFile),
		slog.Int("tcp_port", cfg.TCPPort),
	)
}

func (log *logger) logDaemonStopped(reason error) {
	log.load().Info("daemon stopped", slog.Any("reason", reason))
}

func (log *logger) load() *slog.Logger {
	return log.logger.Load()
}
