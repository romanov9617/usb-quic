// Package cli contains the usb-quic command-line presentation layer.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"usb-quic/internal/adapter/logging"
	adapterusbip "usb-quic/internal/adapter/usbip"
	"usb-quic/internal/adapter/vhci"
)

const (
	appName        = "usb-quic"
	defaultTCPPort = 3240
	localTCPHost   = "127.0.0.1"
	portCommand    = "port"
)

// version is injected by build flags. The fallback is used for local builds.
var version = "dev"

type rootOptions struct {
	debug   bool
	log     bool
	tcpPort int
}

// Runtime contains CLI runtime dependencies.
type Runtime struct {
	LogLevel     *logging.LevelVar
	AttachRemote AttachRemoteFunc
	DetachRemote DetachRemoteFunc
	ListImported ListImportedFunc
	ListRemote   ListRemoteFunc
}

// AttachRemoteFunc imports a remote USB/IP device into local vhci_hcd.
type AttachRemoteFunc func(ctx context.Context, endpoint adapterusbip.Endpoint, busid string) (int, error)

// DetachRemoteFunc detaches a USB/IP device from local vhci_hcd.
type DetachRemoteFunc func(ctx context.Context, port int) error

// ListImportedFunc lists USB/IP devices imported into local vhci_hcd.
type ListImportedFunc func(ctx context.Context) ([]vhci.ImportedDevice, error)

// ListRemoteFunc lists devices exported by a remote USB/IP endpoint.
type ListRemoteFunc func(ctx context.Context, endpoint adapterusbip.Endpoint) ([]adapterusbip.Device, error)

// NewRootCommand builds a cobra command tree compatible with the usbip CLI shape.
func NewRootCommand(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	runtime := Runtime{
		LogLevel:     nil,
		AttachRemote: attachRemote,
		DetachRemote: detachRemote,
		ListImported: listImported,
		ListRemote:   adapterusbip.ListExportedDevices,
	}

	return NewRootCommandWithRuntime(stdin, stdout, stderr, runtime)
}

// NewRootCommandWithRuntime builds a cobra command tree with runtime dependencies.
func NewRootCommandWithRuntime(stdin io.Reader, stdout, stderr io.Writer, runtime Runtime) *cobra.Command {
	if runtime.AttachRemote == nil {
		runtime.AttachRemote = attachRemote
	}

	if runtime.DetachRemote == nil {
		runtime.DetachRemote = detachRemote
	}

	if runtime.ListImported == nil {
		runtime.ListImported = listImported
	}

	if runtime.ListRemote == nil {
		runtime.ListRemote = adapterusbip.ListExportedDevices
	}

	opts := rootOptions{
		debug:   false,
		log:     false,
		tcpPort: adapterusbip.DefaultPort,
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:           appName + " [--debug] [--log] [--tcp-port PORT] [version]\n             [help] <command> <args>",
		Short:         "USB/IP-compatible CLI over QUIC",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			configureLogLevel(runtime, opts.debug)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetHelpTemplate(rootHelpTemplate())
	cmd.SetUsageTemplate(rootHelpTemplate())

	cmd.PersistentFlags().BoolVar(&opts.debug, "debug", false, "enable debug output")
	cmd.PersistentFlags().BoolVar(&opts.log, "log", false, "enable logging")
	cmd.PersistentFlags().IntVar(&opts.tcpPort, "tcp-port", defaultTCPPort, "TCP port used by usbip-compatible endpoints")

	cmd.AddCommand(
		newAttachCommand(runtime, &opts),
		newDetachCommand(runtime),
		newListCommand(runtime, &opts),
		newBindCommand(),
		newUnbindCommand(),
		newPortCommand(runtime),
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
