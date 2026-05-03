package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type attachOptions struct {
	remote string
	busid  string
	device string
}

type detachOptions struct {
	port string
}

type listOptions struct {
	local    bool
	remote   string
	parsable bool
	device   bool
}

type deviceOptions struct {
	busid string
}

func newAttachCommand(runtime Runtime, rootOpts *rootOptions) *cobra.Command {
	opts := attachOptions{
		remote: "",
		busid:  "",
		device: "",
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "attach <args>",
		Short: "Attach a remote USB device",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = cmd
			_ = runtime
			_ = rootOpts

			return notImplemented("attach")
		},
	}

	cmd.Flags().StringVarP(&opts.remote, "remote", "r", "", "remote host")
	cmd.Flags().StringVarP(&opts.busid, "busid", "b", "", "remote USB device bus ID")
	cmd.Flags().StringVarP(&opts.device, "device", "d", "", "virtual UDC device ID")
	cmd.SetHelpTemplate(attachHelpTemplate())

	return cmd
}

func newDetachCommand() *cobra.Command {
	opts := detachOptions{
		port: "",
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "detach <args>",
		Short: "Detach a remote USB device",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return notImplemented("detach")
		},
	}

	cmd.Flags().StringVarP(&opts.port, "port", "p", "", "imported USB device port")
	cmd.SetHelpTemplate(detachHelpTemplate())

	return cmd
}

func newListCommand(runtime Runtime, rootOpts *rootOptions) *cobra.Command {
	opts := listOptions{
		local:    false,
		remote:   "",
		parsable: false,
		device:   false,
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "list [-p|--parsable] <args>",
		Short: "List exportable or local USB devices",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = cmd
			_ = runtime
			_ = rootOpts

			return notImplemented("list")
		},
	}

	cmd.Flags().BoolVarP(&opts.local, "local", "l", false, "list local USB devices")
	cmd.Flags().StringVarP(&opts.remote, "remote", "r", "", "list exportable devices on remote host")
	cmd.Flags().BoolVarP(&opts.parsable, "parsable", "p", false, "print parsable output")
	cmd.Flags().BoolVarP(&opts.device, "device", "d", false, "list local USB gadgets bound to usbip-vudc")
	cmd.SetHelpTemplate(listHelpTemplate())

	return cmd
}

func newBindCommand() *cobra.Command {
	opts := deviceOptions{
		busid: "",
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "bind <args>",
		Short: "Bind device to usbip-host.ko",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return notImplemented("bind")
		},
	}

	cmd.Flags().StringVarP(&opts.busid, "busid", "b", "", "local USB device bus ID")
	cmd.SetHelpTemplate(bindHelpTemplate())

	return cmd
}

func newUnbindCommand() *cobra.Command {
	opts := deviceOptions{
		busid: "",
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "unbind <args>",
		Short: "Unbind device from usbip-host.ko",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return notImplemented("unbind")
		},
	}

	cmd.Flags().StringVarP(&opts.busid, "busid", "b", "", "local USB device bus ID")
	cmd.SetHelpTemplate(unbindHelpTemplate())

	return cmd
}

func newPortCommand() *cobra.Command {
	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	return &cobra.Command{
		Use:   portCommand,
		Short: "Show imported USB devices",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return notImplemented(portCommand)
		},
	}
}

func attachHelpTemplate() string {
	return `usage: usb-quic attach <args>
    -r, --remote=<host>      The machine with exported USB devices
    -b, --busid=<busid>    Busid of the device on <host>
    -d, --device=<devid>    Id of the virtual UDC on <host>
`
}

func listHelpTemplate() string {
	return `usage: usb-quic list [-p|--parsable] <args>
    -p, --parsable         Parsable list format
    -r, --remote=<host>    List the exportable USB devices on <host>
    -l, --local            List the local USB devices
    -d, --device           List the local USB gadgets bound to usbip-vudc
`
}

func detachHelpTemplate() string {
	return `usage: usb-quic detach <args>
    -p, --port=<port>    vhci_hcd port the device is on
`
}

func bindHelpTemplate() string {
	return `usage: usb-quic bind <args>
    -b, --busid=<busid>    Bind usbip-host.ko to device on <busid>
`
}

func unbindHelpTemplate() string {
	return `usage: usb-quic unbind <args>
    -b, --busid=<busid>    Unbind usbip-host.ko from device on <busid>
`
}

func newVersionCommand() *cobra.Command {
	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}
