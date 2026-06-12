package daemon

import (
	"log/slog"
	"net"
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
		slog.Bool("dev_insecure_tls", cfg.DevInsecureTLS),
		slog.String("pid_file", cfg.PIDFile),
		slog.String("quic_addr", cfg.QUICAddr),
		slog.String("quic_listen", cfg.QUICListen),
		slog.String("tcp_listen", cfg.TCPListen),
		slog.Int("tcp_port", cfg.TCPPort),
		slog.String("transport_mode", cfg.TransportMode),
		slog.String("upstream", cfg.Upstream),
	)
}

func (log *logger) logDaemonStopped(reason error) {
	log.load().Info("daemon stopped", slog.Any("reason", reason))
}

func (log *logger) logClientAccepted(remoteAddr net.Addr) {
	log.load().Info("client accepted", slog.String("remote_addr", addrString(remoteAddr)))
}

func (log *logger) logStreamOpenFailed(remoteAddr net.Addr, err error) {
	log.load().Error(
		"stream open failed",
		slog.String("remote_addr", addrString(remoteAddr)),
		slog.Any("error", err),
	)
}

func (log *logger) logTunnelStarted(remoteAddr net.Addr) {
	log.load().Info("tunnel started", slog.String("remote_addr", addrString(remoteAddr)))
}

func (log *logger) logTunnelStopped(remoteAddr net.Addr) {
	log.load().Info("tunnel stopped", slog.String("remote_addr", addrString(remoteAddr)))
}

func (log *logger) logTunnelFailed(remoteAddr net.Addr, err error) {
	log.load().Error(
		"tunnel failed",
		slog.String("remote_addr", addrString(remoteAddr)),
		slog.Any("error", err),
	)
}

func (log *logger) load() *slog.Logger {
	return log.logger.Load()
}

func addrString(addr net.Addr) string {
	if addr == nil {
		return ""
	}

	return addr.String()
}
