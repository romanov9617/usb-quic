// Package cli contains the usb-quic command-line presentation layer.
package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/spf13/cobra"

	"usb-quic/internal/adapter/logging"
	"usb-quic/internal/adapter/usbip"
)

const (
	appName        = "usb-quic"
	defaultTCPPort = 3240
)

// version is injected by build flags. The fallback is used for local builds.
var version = "dev"

type rootOptions struct {
	debug   bool
	log     bool
	verbose bool
	tcpPort int
}

// Role identifies CLI role-specific behavior.
type Role string

const (
	// RoleClient enables outgoing USB/IP client commands.
	RoleClient Role = "client"
	// RoleServer enables incoming USB/IP server listener.
	RoleServer Role = "server"
)

// USBIPTransport is the USB/IP TCP transport used by CLI commands.
type USBIPTransport interface {
	ProxyTCPToQUIC(ctx context.Context, tcpAddress, quicAddress string) error
	ProxyQUICToTCP(ctx context.Context, quicAddress, tcpAddress string) error
}

// Runtime contains CLI runtime dependencies.
type Runtime struct {
	Role           Role
	LogLevel       *logging.LevelVar
	USBIPTransport USBIPTransport
}

// NewRootCommand builds a cobra command tree compatible with the usbip CLI shape.
func NewRootCommand(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	runtime := Runtime{
		Role:           "",
		LogLevel:       nil,
		USBIPTransport: nil,
	}

	return NewRootCommandWithRuntime(stdin, stdout, stderr, runtime)
}

// NewRootCommandWithRuntime builds a cobra command tree with runtime dependencies.
func NewRootCommandWithRuntime(stdin io.Reader, stdout, stderr io.Writer, runtime Runtime) *cobra.Command {
	opts := rootOptions{
		debug:   false,
		log:     false,
		verbose: false,
		tcpPort: usbip.DefaultPort,
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:           appName + " [-v] [--debug] [--log] [--tcp-port PORT] [version] [help] <command> <args>",
		Short:         "USB/IP-compatible CLI over QUIC",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			configureLogLevel(runtime, opts.verbose)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runtime.Role == RoleServer {
				return listenUSBIP(cmd, runtime, opts.tcpPort)
			}

			return cmd.Help()
		},
	}

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetHelpTemplate(rootHelpTemplate())
	cmd.SetUsageTemplate(rootHelpTemplate())

	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "enable debug logging")
	cmd.PersistentFlags().BoolVar(&opts.debug, "debug", false, "enable debug output")
	cmd.PersistentFlags().BoolVar(&opts.log, "log", false, "enable logging")
	cmd.PersistentFlags().IntVar(&opts.tcpPort, "tcp-port", defaultTCPPort, "TCP port used by usbip-compatible endpoints")

	cmd.AddCommand(
		newAttachCommand(runtime, &opts),
		newDetachCommand(),
		newListCommand(runtime, &opts),
		newBindCommand(),
		newUnbindCommand(),
		newPortCommand(),
		newVersionCommand(),
	)

	return cmd
}

func rootHelpTemplate() string {
	return `usage: {{.Use}}

  attach     Attach a remote USB device
  detach     Detach a remote USB device
  list       List exportable or local USB devices
  bind       Bind device to usbip-host.ko
  unbind     Unbind device from usbip-host.ko
  port       Show imported USB devices
`
}

func notImplemented(name string) error {
	return fmt.Errorf("%s: %w", name, ErrNotImplemented)
}

func configureLogLevel(runtime Runtime, verbose bool) {
	logging.SetVerboseLevel(runtime.LogLevel, verbose)
}

func listenUSBIP(cmd *cobra.Command, runtime Runtime, port int) error {
	if runtime.USBIPTransport == nil {
		return notImplemented("listen")
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	quicAddress := net.JoinHostPort("", strconv.Itoa(port))
	tcpAddress := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	err := runtime.USBIPTransport.ProxyQUICToTCP(ctx, quicAddress, tcpAddress)
	if err != nil {
		return fmt.Errorf("proxy usbip quic to tcp: %w", err)
	}

	return nil
}
