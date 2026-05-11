package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	adapterusbip "usb-quic/internal/adapter/usbip"
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

const (
	usbClassDefinedAtInterface = 0x00
	usbClassCommunications     = 0x02
	usbClassHID                = 0x03
	usbClassCDCData            = 0x0a
	usbClassMiscellaneous      = 0xef
	usbClassVendorSpecific     = 0xff

	usbSubClassUnknown  = 0x00
	usbSubClassBoot     = 0x01
	usbSubClassAbstract = 0x02

	usbProtocolUnknown              = 0x00
	usbProtocolInterfaceAssociation = 0x01
	usbProtocolMouse                = 0x02
)

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
			return runAttach(cmd, runtime, rootOpts, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.remote, "remote", "r", "", "remote host")
	cmd.Flags().StringVarP(&opts.busid, "busid", "b", "", "remote USB device bus ID")
	cmd.Flags().StringVarP(&opts.device, "device", "d", "", "virtual UDC device ID")
	cmd.SetHelpTemplate(attachHelpTemplate())

	return cmd
}

func runAttach(cmd *cobra.Command, runtime Runtime, rootOpts *rootOptions, opts attachOptions) error {
	busid := opts.busid
	if busid == "" {
		busid = opts.device
	}

	if opts.remote == "" {
		return ErrAttachRemoteRequired
	}

	if busid == "" {
		return ErrAttachBusIDRequired
	}

	_, err := runtime.AttachRemote(cmd.Context(), adapterusbip.Endpoint{
		Host: opts.remote,
		Port: rootOpts.tcpPort,
	}, busid)
	if err != nil {
		return fmt.Errorf("attach remote device %s from %s: %w", busid, opts.remote, err)
	}

	return nil
}

func newDetachCommand(runtime Runtime) *cobra.Command {
	opts := detachOptions{
		port: "",
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "detach <args>",
		Short: "Detach a remote USB device",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDetach(cmd, runtime, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.port, "port", "p", "", "imported USB device port")
	cmd.SetHelpTemplate(detachHelpTemplate())

	return cmd
}

func runDetach(cmd *cobra.Command, runtime Runtime, opts detachOptions) error {
	if opts.port == "" {
		return ErrDetachPortRequired
	}

	port, err := strconv.Atoi(opts.port)
	if err != nil {
		return fmt.Errorf("detach: parse port %q: %w", opts.port, err)
	}

	err = runtime.DetachRemote(cmd.Context(), port)
	if err != nil {
		return fmt.Errorf("detach remote device from port %d: %w", port, err)
	}

	return nil
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
			if opts.remote != "" && !opts.local && !opts.device && !opts.parsable {
				return runListRemote(cmd, runtime, rootOpts, opts)
			}

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

func runListRemote(cmd *cobra.Command, runtime Runtime, rootOpts *rootOptions, opts listOptions) error {
	devices, err := runtime.ListRemote(cmd.Context(), adapterusbip.Endpoint{
		Host: opts.remote,
		Port: rootOpts.tcpPort,
	})
	if err != nil {
		return fmt.Errorf("list remote devices from %s: %w", opts.remote, err)
	}

	writeListRemoteOutput(cmd, opts.remote, devices)

	return nil
}

func writeListRemoteOutput(cmd *cobra.Command, remote string, devices []adapterusbip.Device) {
	if len(devices) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "usbip: info: no exportable devices found on %s\n", remote)

		return
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Exportable USB devices")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "======================")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), " - %s\n", remote)

	for _, device := range devices {
		writeDevice(cmd, device)
	}
}

func writeDevice(cmd *cobra.Command, device adapterusbip.Device) {
	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(),
		"        %s: unknown vendor : unknown product (%04x:%04x)\n",
		device.BusID,
		device.IDVendor,
		device.IDProduct,
	)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "           : %s\n", device.Path)
	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(),
		"           : %s (%02x/%02x/%02x)\n",
		usbClassDescription(device.BDeviceClass, device.BDeviceSubClass, device.BDeviceProtocol),
		device.BDeviceClass,
		device.BDeviceSubClass,
		device.BDeviceProtocol,
	)

	for index, iface := range device.Interfaces {
		_, _ = fmt.Fprintf(
			cmd.OutOrStdout(),
			"           :  %d - %s (%02x/%02x/%02x)\n",
			index,
			usbClassDescription(iface.Class, iface.SubClass, iface.Protocol),
			iface.Class,
			iface.SubClass,
			iface.Protocol,
		)
	}
}

func usbClassDescription(class, subclass, protocol uint8) string {
	if class == usbClassDefinedAtInterface && subclass == usbSubClassUnknown && protocol == usbProtocolUnknown {
		return "(Defined at Interface level)"
	}

	return fmt.Sprintf("%s / %s / %s", usbClassName(class), usbSubClassName(subclass), usbProtocolName(protocol))
}

func usbClassName(class uint8) string {
	switch class {
	case usbClassCommunications:
		return "Communications"
	case usbClassHID:
		return "Human Interface Device"
	case usbClassCDCData:
		return "CDC Data"
	case usbClassMiscellaneous:
		return "Miscellaneous Device"
	case usbClassVendorSpecific:
		return "Vendor Specific Class"
	default:
		return "unknown class"
	}
}

func usbSubClassName(subclass uint8) string {
	switch subclass {
	case usbSubClassUnknown:
		return "unknown subclass"
	case usbSubClassBoot:
		return "Boot Interface Subclass"
	case usbSubClassAbstract:
		return "Abstract (modem)"
	default:
		return "unknown subclass"
	}
}

func usbProtocolName(protocol uint8) string {
	switch protocol {
	case usbProtocolUnknown:
		return "unknown protocol"
	case usbProtocolInterfaceAssociation:
		return "Interface Association"
	case usbProtocolMouse:
		return "Mouse"
	default:
		return "unknown protocol"
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
