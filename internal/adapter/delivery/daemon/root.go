// Package daemon contains the usb-quic proxy service command-line presentation layer.
package daemon

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"usb-quic/internal/adapter/logging"
	adapterusbip "usb-quic/internal/adapter/usbip"
	"usb-quic/internal/config"
)

const appName = "usb-quicd"

const (
	defaultQUICAddress   = "127.0.0.1:4242"
	defaultTransportMode = ""
	defaultUpstream      = "127.0.0.1:3240"
)

// version is injected by build flags. The fallback is used for local builds.
var version = "dev"

type rootOptions struct {
	ipv4           bool
	ipv6           bool
	device         bool
	daemon         bool
	debug          bool
	devInsecureTLS bool
	pidFile        string
	quicAddr       string
	quicListen     string
	tcpListen      string
	tcpPort        int
	transportMode  string
	upstream       string
	version        bool
}

// RunFunc runs the daemon runtime.
type RunFunc func(ctx context.Context, config config.Daemon) error

// Runtime contains daemon runtime dependencies.
type Runtime struct {
	LogLevel *logging.LevelVar
	Run      RunFunc
}

// NewRootCommand builds the usb-quicd service command.
func NewRootCommand(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	runtime := Runtime{
		LogLevel: nil,
		Run:      nil,
	}

	return NewRootCommandWithRuntime(stdin, stdout, stderr, runtime)
}

// NewRootCommandWithRuntime builds a daemon command with runtime dependencies.
func NewRootCommandWithRuntime(stdin io.Reader, stdout, stderr io.Writer, runtime Runtime) *cobra.Command {
	opts := rootOptions{
		ipv4:           false,
		ipv6:           false,
		device:         false,
		daemon:         false,
		debug:          false,
		devInsecureTLS: false,
		pidFile:        "",
		quicAddr:       defaultQUICAddress,
		quicListen:     defaultQUICAddress,
		tcpListen:      "",
		tcpPort:        adapterusbip.DefaultPort,
		transportMode:  defaultTransportMode,
		upstream:       defaultUpstream,
		version:        false,
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this daemon CLI.
	cmd := &cobra.Command{
		Use:           appName + " [options]",
		Short:         "USB/IP daemon-compatible service over QUIC",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			configureLogLevel(runtime, opts.debug)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.version {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", appName, version)

				return nil
			}

			return runDaemon(cmd, runtime, opts)
		},
	}

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetHelpTemplate(rootHelpTemplate())
	cmd.SetUsageTemplate(rootHelpTemplate())

	registerFlags(cmd, &opts)

	return cmd
}

func registerFlags(cmd *cobra.Command, opts *rootOptions) {
	cmd.Flags().BoolVarP(&opts.ipv4, "ipv4", "4", false, "bind to IPv4")
	cmd.Flags().BoolVarP(&opts.ipv6, "ipv6", "6", false, "bind to IPv6")
	cmd.Flags().BoolVarP(&opts.device, "device", "e", false, "run in device mode")
	cmd.Flags().BoolVarP(&opts.daemon, "daemon", "D", false, "run as a daemon process")
	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "print debugging information")
	cmd.Flags().StringVarP(&opts.pidFile, "pid", "P", "", "write process id to file")
	cmd.Flags().IntVarP(&opts.tcpPort, "tcp-port", "t", adapterusbip.DefaultPort, "listen on TCP/IP port")
	cmd.Flags().StringVar(
		&opts.tcpListen,
		"tcp-listen",
		opts.tcpListen,
		"TCP listen address for tcp and quic-client modes",
	)
	cmd.Flags().BoolVarP(&opts.version, "version", "v", false, "show version")
	cmd.Flags().StringVar(
		&opts.transportMode,
		"transport",
		opts.transportMode,
		"transport mode: tcp, quic-client, or quic-server (required)",
	)
	cmd.Flags().StringVar(
		&opts.quicListen,
		"quic-listen",
		opts.quicListen,
		"QUIC listen address for quic-server mode",
	)
	cmd.Flags().StringVar(
		&opts.quicAddr,
		"quic-server",
		opts.quicAddr,
		"QUIC server address for quic-client mode",
	)
	cmd.Flags().StringVar(
		&opts.upstream,
		"upstream",
		opts.upstream,
		"TCP upstream address for tcp and quic-server modes",
	)
	cmd.Flags().BoolVar(
		&opts.devInsecureTLS,
		"insecure-dev-tls",
		opts.devInsecureTLS,
		"use ephemeral self-signed/insecure TLS; development only",
	)
}

func rootHelpTemplate() string {
	return `usage: usb-quicd [options]

  --transport MODE       Required: tcp, quic-client, or quic-server
  --tcp-listen ADDRESS   TCP listen address for tcp and quic-client modes
  --quic-server ADDRESS  QUIC server address for quic-client mode
  --quic-listen ADDRESS  QUIC listen address for quic-server mode
  --upstream ADDRESS     TCP upstream address for tcp and quic-server modes
  --insecure-dev-tls     Use development-only insecure TLS
  -d, --debug            Enable debug logging
  -v, --version          Show version
`
}

func notImplemented() error {
	return fmt.Errorf("%s: %w", appName, ErrNotImplemented)
}

func configureLogLevel(runtime Runtime, verbose bool) {
	logging.SetVerboseLevel(runtime.LogLevel, verbose)
}

func runDaemon(cmd *cobra.Command, runtime Runtime, opts rootOptions) error {
	if runtime.Run == nil {
		return notImplemented()
	}

	err := runtime.Run(cmd.Context(), configFromOptions(opts))
	if err != nil {
		return fmt.Errorf("%s: %w", appName, err)
	}

	return nil
}

func configFromOptions(opts rootOptions) config.Daemon {
	return config.Daemon{
		BindIPv4:       opts.ipv4,
		BindIPv6:       opts.ipv6,
		DeviceMode:     opts.device,
		Daemonize:      opts.daemon,
		Debug:          opts.debug,
		DevInsecureTLS: opts.devInsecureTLS,
		PIDFile:        opts.pidFile,
		QUICAddr:       opts.quicAddr,
		QUICListen:     opts.quicListen,
		TCPListen:      opts.tcpListen,
		TCPPort:        opts.tcpPort,
		TransportMode:  opts.transportMode,
		Upstream:       opts.upstream,
	}
}
