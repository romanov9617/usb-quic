// Package daemon contains the usbipd-like service command-line presentation layer.
package daemon

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"usb-quic/internal/adapter/logging"
	domainusbip "usb-quic/internal/domain/usbip"
)

const appName = "usbipd"

// version is injected by build flags. The fallback is used for local builds.
var version = "dev"

type rootOptions struct {
	ipv4    bool
	ipv6    bool
	device  bool
	daemon  bool
	debug   bool
	pidFile string
	tcpPort int
	version bool
}

// Runtime contains daemon runtime dependencies.
type Runtime struct {
	LogLevel *logging.LevelVar
}

// NewRootCommand builds a usbipd-like daemon command.
func NewRootCommand(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	runtime := Runtime{
		LogLevel: nil,
	}

	return NewRootCommandWithRuntime(stdin, stdout, stderr, runtime)
}

// NewRootCommandWithRuntime builds a daemon command with runtime dependencies.
func NewRootCommandWithRuntime(stdin io.Reader, stdout, stderr io.Writer, runtime Runtime) *cobra.Command {
	opts := rootOptions{
		ipv4:    false,
		ipv6:    false,
		device:  false,
		daemon:  false,
		debug:   false,
		pidFile: "",
		tcpPort: domainusbip.DefaultPort,
		version: false,
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

			return notImplemented()
		},
	}

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetHelpTemplate(rootHelpTemplate())
	cmd.SetUsageTemplate(rootHelpTemplate())

	cmd.Flags().BoolVarP(&opts.ipv4, "ipv4", "4", false, "bind to IPv4")
	cmd.Flags().BoolVarP(&opts.ipv6, "ipv6", "6", false, "bind to IPv6")
	cmd.Flags().BoolVarP(&opts.device, "device", "e", false, "run in device mode")
	cmd.Flags().BoolVarP(&opts.daemon, "daemon", "D", false, "run as a daemon process")
	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "print debugging information")
	cmd.Flags().StringVarP(&opts.pidFile, "pid", "P", "", "write process id to file")
	cmd.Flags().IntVarP(&opts.tcpPort, "tcp-port", "t", domainusbip.DefaultPort, "listen on TCP/IP port")
	cmd.Flags().BoolVarP(&opts.version, "version", "v", false, "show version")

	return cmd
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
