// Package daemon contains the usbipd-like service command-line presentation layer.
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

const appName = "usbipd"

const (
	defaultQUICAddress   = "127.0.0.1:4242"
	defaultTransportMode = "tcp"
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

// NewRootCommand builds a usbipd-like daemon command.
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
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s (usb-quic %s)\n", appName, version)

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
	cmd.Flags().BoolVarP(&opts.version, "version", "v", false, "show version")
	cmd.Flags().StringVar(
		&opts.transportMode,
		"usb-quic-transport",
		opts.transportMode,
		"usb-quic transport mode: tcp, quic-client, or quic-server",
	)
	cmd.Flags().StringVar(
		&opts.quicListen,
		"usb-quic-quic-listen",
		opts.quicListen,
		"usb-quic QUIC listen address for quic-server mode",
	)
	cmd.Flags().StringVar(
		&opts.quicAddr,
		"usb-quic-quic-addr",
		opts.quicAddr,
		"usb-quic QUIC server address for quic-client mode",
	)
	cmd.Flags().StringVar(
		&opts.upstream,
		"usb-quic-upstream",
		opts.upstream,
		"usb-quic TCP upstream address for quic-server mode",
	)
	cmd.Flags().BoolVar(
		&opts.devInsecureTLS,
		"usb-quic-dev-insecure-tls",
		opts.devInsecureTLS,
		"use dev-only self-signed/insecure TLS for local transport smoke tests",
	)
	hideUSBQUICFlags(cmd)
}

func rootHelpTemplate() string {
	return `usage: usbipd [options]

	-4, --ipv4
		Bind to IPv4. Default is both.

	-6, --ipv6
		Bind to IPv6. Default is both.

	-e, --device
		Run in device mode.
		Rather than drive an attached device, create
		a virtual UDC to bind gadgets to.

	-D, --daemon
		Run as a daemon process.

	-d, --debug
		Print debugging information.

	-PFILE, --pid FILE
		Write process id to FILE.
		If no FILE specified, use /var/run/usbipd.pid

	-tPORT, --tcp-port PORT
		Listen on TCP/IP port PORT.

	-h, --help
		Print this help.

	-v, --version
		Show version.

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
		TCPPort:        opts.tcpPort,
		TransportMode:  opts.transportMode,
		Upstream:       opts.upstream,
	}
}

func hideUSBQUICFlags(cmd *cobra.Command) {
	for _, name := range []string{
		"usb-quic-transport",
		"usb-quic-quic-listen",
		"usb-quic-quic-addr",
		"usb-quic-upstream",
		"usb-quic-dev-insecure-tls",
	} {
		err := cmd.Flags().MarkHidden(name)
		if err != nil {
			panic(err)
		}
	}
}
