// Package di assembles application entrypoints.
package di

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"usb-quic/internal/adapter/delivery/cli"
	"usb-quic/internal/adapter/delivery/daemon"
	"usb-quic/internal/adapter/logging"
	"usb-quic/internal/config"
	runtimedaemon "usb-quic/internal/daemon"
	"usb-quic/internal/transport"
)

const (
	daemonTransportTCP        = "tcp"
	daemonTransportQUICClient = "quic-client"
	daemonTransportQUICServer = "quic-server"
	devNextProto              = "usb-quic-dev"
	devRSAKeyBits             = 2048
)

var errDevTLSRequired = errors.New("dev insecure tls is required for current quic transport")
var errUnknownTransportMode = errors.New("unknown daemon transport mode")

// Streams contains process streams used by commands.
type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// ExecuteCLI builds and executes the usb-quic operator CLI.
func ExecuteCLI(stdin io.Reader, stdout, stderr io.Writer) error {
	const name = "cli"

	return execute(
		stdin,
		stdout,
		stderr,
		newRootCommand,
		name,
	)
}

// ExecuteDaemon builds and executes the daemon CLI.
func ExecuteDaemon(stdin io.Reader, stdout, stderr io.Writer) error {
	const name = "daemon"

	return execute(
		stdin,
		stdout,
		stderr,
		newDaemonCommand,
		name,
	)
}

func execute(
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	commandProvider any,
	name string,
) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	streams := Streams{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}

	var command *cobra.Command

	application := fx.New(
		fx.NopLogger,
		fx.Supply(streams),
		fx.Provide(newLogLevel),
		fx.Provide(newLogger),
		fx.Provide(newDaemonRun),
		fx.Provide(commandProvider),
		fx.Populate(&command),
	)

	err := application.Err()
	if err != nil {
		return fmt.Errorf("build %s container: %w", name, err)
	}

	command.SetContext(ctx)

	err = command.Execute()
	if err != nil {
		return fmt.Errorf("execute %s: %w", name, err)
	}

	return nil
}

func newLogLevel() *logging.LevelVar {
	return logging.NewVerboseLevel(false)
}

func newLogger(streams Streams, level *logging.LevelVar) *logging.Logger {
	return logging.NewTextLogger(streams.Stderr, level)
}

func newRootCommand(streams Streams, level *logging.LevelVar) *cobra.Command {
	runtime := cli.Runtime{
		LogLevel:     level,
		AttachRemote: nil,
		BindLocal:    nil,
		DetachRemote: nil,
		ListImported: nil,
		ListRemote:   nil,
		UnbindLocal:  nil,
	}

	return cli.NewRootCommandWithRuntime(
		streams.Stdin,
		streams.Stdout,
		streams.Stderr,
		runtime,
	)
}

func newDaemonRun(logger *logging.Logger) daemon.RunFunc {
	return func(ctx context.Context, cfg config.Daemon) error {
		return runDaemon(ctx, cfg, logger)
	}
}

func runDaemon(ctx context.Context, cfg config.Daemon, logger *logging.Logger) error {
	switch cfg.TransportMode {
	case "", daemonTransportTCP:
		err := runtimedaemon.New(cfg, logger).Run(ctx)
		if err != nil {
			return fmt.Errorf("run tcp daemon: %w", err)
		}

		return nil
	case daemonTransportQUICClient:
		return runQUICClientDaemon(ctx, cfg, logger)
	case daemonTransportQUICServer:
		return runQUICServerDaemon(ctx, cfg, logger)
	default:
		return fmt.Errorf("%w %q", errUnknownTransportMode, cfg.TransportMode)
	}
}

func runQUICClientDaemon(ctx context.Context, cfg config.Daemon, logger *logging.Logger) error {
	tlsConfig, err := clientTLSConfig(cfg)
	if err != nil {
		return err
	}

	streamOpener := transport.NewQUICDialStreamOpener(cfg.QUICAddr, tlsConfig, nil)
	stopClose := context.AfterFunc(ctx, func() {
		_ = streamOpener.Close()
	})

	defer func() {
		stopClose()

		_ = streamOpener.Close()
	}()

	err = runtimedaemon.New(cfg, logger, runtimedaemon.WithStreamOpener(streamOpener)).Run(ctx)
	if err != nil {
		return fmt.Errorf("run quic client daemon: %w", err)
	}

	return nil
}

func runQUICServerDaemon(ctx context.Context, cfg config.Daemon, logger *logging.Logger) error {
	tlsConfig, err := serverTLSConfig(cfg)
	if err != nil {
		return err
	}

	listener, err := quic.ListenAddr(cfg.QUICListen, tlsConfig, nil)
	if err != nil {
		return fmt.Errorf("listen quic %s: %w", cfg.QUICListen, err)
	}
	defer func() {
		_ = listener.Close()
	}()

	logger.Info(
		"quic daemon listening",
		slog.String("quic_listen", listener.Addr().String()),
		slog.String("tcp_upstream", cfg.Upstream),
	)

	go func() {
		<-ctx.Done()

		_ = listener.Close()
	}()

	upstream := transport.NewTCPStreamOpener(cfg.Upstream)

	err = runtimedaemon.New(cfg, logger, runtimedaemon.WithStreamOpener(upstream)).ServeQUIC(ctx, listener)
	if err != nil {
		return fmt.Errorf("run quic server daemon: %w", err)
	}

	return nil
}

func clientTLSConfig(cfg config.Daemon) (*tls.Config, error) {
	if !cfg.DevInsecureTLS {
		return nil, errDevTLSRequired
	}

	//nolint:exhaustruct // Dev-only TLS config sets only fields needed by quic-go.
	return &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // Explicit dev-only smoke-test mode.
		NextProtos:         []string{devNextProto},
	}, nil
}

func serverTLSConfig(cfg config.Daemon) (*tls.Config, error) {
	if !cfg.DevInsecureTLS {
		return nil, errDevTLSRequired
	}

	cert, err := devSelfSignedCert()
	if err != nil {
		return nil, err
	}

	//nolint:exhaustruct // Dev-only TLS config sets only fields needed by quic-go.
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{devNextProto},
	}, nil
}

func devSelfSignedCert() (tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, devRSAKeyBits)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate dev tls key: %w", err)
	}

	//nolint:exhaustruct // Dev certificate sets only fields required for a local handshake.
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{ //nolint:exhaustruct // CommonName is enough for the dev certificate.
			CommonName: "usb-quic-dev",
		},
		NotBefore: time.Now().Add(-time.Minute),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		DNSNames:  []string{"localhost"},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create dev tls certificate: %w", err)
	}

	keyDER := x509.MarshalPKCS1PrivateKey(key)
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:    "CERTIFICATE",
		Headers: map[string]string{},
		Bytes:   certDER,
	})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: map[string]string{},
		Bytes:   keyDER,
	})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse dev tls certificate: %w", err)
	}

	return cert, nil
}

func newDaemonCommand(
	streams Streams,
	level *logging.LevelVar,
	run daemon.RunFunc,
) *cobra.Command {
	runtime := daemon.Runtime{
		LogLevel: level,
		Run:      run,
	}

	return daemon.NewRootCommandWithRuntime(
		streams.Stdin,
		streams.Stdout,
		streams.Stderr,
		runtime,
	)
}
