// Package cli contains the usb-quic command-line presentation layer.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
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
	tcpPort int
}

// Execute runs the usb-quic CLI using process stdin/stdout/stderr.
func Execute() error {
	err := NewRootCommand(os.Stdin, os.Stdout, os.Stderr).Execute()
	if err != nil {
		return fmt.Errorf("execute cli: %w", err)
	}

	return nil
}

// NewRootCommand builds a cobra command tree compatible with the usbip CLI shape.
func NewRootCommand(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	opts := rootOptions{
		debug:   false,
		log:     false,
		tcpPort: 0,
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:           appName + " [--debug] [--log] [--tcp-port PORT] [version] [help] <command> <args>",
		Short:         "USB/IP-compatible CLI over QUIC",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
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
		newAttachCommand(),
		newDetachCommand(),
		newListCommand(),
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
