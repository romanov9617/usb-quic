package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"usb-quic/internal/adapter/logging"
	adapterusbip "usb-quic/internal/adapter/usbip"
)

const testBusID = "1-1"

const (
	attachCommand = "attach"
	bindCommand   = "bind"
	detachCommand = "detach"
	helpFlag      = "--help"
	listCommand   = "list"
	unbindCommand = "unbind"
)

func TestRootHelpMatchesUSBIPShape(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	cmd := NewRootCommand(nil, &stdout, nil)
	cmd.SetArgs([]string{"help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute help: %v", err)
	}

	want := `usage: usb-quic [--debug] [--log] [--tcp-port PORT] [version]
             [help] <command> <args>

  attach     Attach a remote USB device
  detach     Detach a remote USB device
  list       List exportable or local USB devices
  bind       Bind device to usbip-host.ko
  unbind     Unbind device from usbip-host.ko
  port       Show imported USB devices
`

	if got := stdout.String(); got != want {
		t.Fatalf("unexpected help\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	cmd := NewRootCommand(nil, &stdout, nil)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute version: %v", err)
	}

	if got := stdout.String(); got != version+"\n" {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestUSBIPCompatibleCommandsParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "list local", args: []string{listCommand, "-l"}},
		{name: "list device", args: []string{listCommand, "-d"}},
		{name: bindCommand, args: []string{bindCommand, "-b", testBusID}},
		{name: unbindCommand, args: []string{unbindCommand, "-b", testBusID}},
		{name: portCommand, args: []string{portCommand}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := NewRootCommand(nil, bytes.NewBuffer(nil), bytes.NewBuffer(nil))
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if !errors.Is(err, ErrNotImplemented) {
				t.Fatalf("expected ErrNotImplemented, got %v", err)
			}
		})
	}
}

func TestAttachRemoteCommand(t *testing.T) {
	t.Parallel()

	runtime := Runtime{
		LogLevel: logging.NewDefaultLevel(),
		AttachRemote: func(_ context.Context, endpoint adapterusbip.Endpoint, busid string) (int, error) {
			if endpoint.Host != localTCPHost || endpoint.Port != 4321 {
				t.Fatalf("endpoint=%+v, want host=%s port=4321", endpoint, localTCPHost)
			}

			if busid != testBusID {
				t.Fatalf("busid=%q, want %q", busid, testBusID)
			}

			return 0, nil
		},
		DetachRemote: nil,
		ListRemote: func(context.Context, adapterusbip.Endpoint) ([]adapterusbip.Device, error) {
			t.Fatal("ListRemote must not be called by attach")

			return nil, nil
		},
	}

	cmd := NewRootCommandWithRuntime(nil, bytes.NewBuffer(nil), bytes.NewBuffer(nil), runtime)
	cmd.SetArgs([]string{"--tcp-port", "4321", attachCommand, "-r", localTCPHost, "-b", testBusID})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute attach: %v", err)
	}
}

func TestAttachRemoteCommandAcceptsDeviceFlagAsBusID(t *testing.T) {
	t.Parallel()

	runtime := Runtime{
		LogLevel: logging.NewDefaultLevel(),
		AttachRemote: func(_ context.Context, _ adapterusbip.Endpoint, busid string) (int, error) {
			if busid != "0" {
				t.Fatalf("busid=%q, want %q", busid, "0")
			}

			return 0, nil
		},
		DetachRemote: nil,
		ListRemote:   nil,
	}

	cmd := NewRootCommandWithRuntime(nil, bytes.NewBuffer(nil), bytes.NewBuffer(nil), runtime)
	cmd.SetArgs([]string{attachCommand, "-r", localTCPHost, "-d", "0"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute attach: %v", err)
	}
}

func TestAttachRemoteCommandRequiresRemoteAndBusID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "remote",
			args: []string{attachCommand, "-b", testBusID},
			want: "attach: remote host is required",
		},
		{
			name: "busid",
			args: []string{attachCommand, "-r", localTCPHost},
			want: "attach: busid is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := NewRootCommandWithRuntime(nil, bytes.NewBuffer(nil), bytes.NewBuffer(nil), Runtime{
				LogLevel: logging.NewDefaultLevel(),
				AttachRemote: func(context.Context, adapterusbip.Endpoint, string) (int, error) {
					t.Fatal("AttachRemote must not be called for invalid args")

					return 0, nil
				},
				DetachRemote: nil,
				ListRemote:   nil,
			})
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil || err.Error() != tt.want {
				t.Fatalf("err=%v, want %q", err, tt.want)
			}
		})
	}
}

func TestDetachRemoteCommand(t *testing.T) {
	t.Parallel()

	runtime := Runtime{
		LogLevel:     logging.NewDefaultLevel(),
		AttachRemote: nil,
		DetachRemote: func(_ context.Context, port int) error {
			if port != 3 {
				t.Fatalf("port=%d, want 3", port)
			}

			return nil
		},
		ListRemote: nil,
	}

	cmd := NewRootCommandWithRuntime(nil, bytes.NewBuffer(nil), bytes.NewBuffer(nil), runtime)
	cmd.SetArgs([]string{detachCommand, "-p", "3"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute detach: %v", err)
	}
}

func TestDetachRemoteCommandRequiresPort(t *testing.T) {
	t.Parallel()

	cmd := NewRootCommandWithRuntime(nil, bytes.NewBuffer(nil), bytes.NewBuffer(nil), Runtime{
		LogLevel:     logging.NewDefaultLevel(),
		AttachRemote: nil,
		DetachRemote: func(context.Context, int) error {
			t.Fatal("DetachRemote must not be called for invalid args")

			return nil
		},
		ListRemote: nil,
	})
	cmd.SetArgs([]string{detachCommand})

	err := cmd.Execute()
	if !errors.Is(err, ErrDetachPortRequired) {
		t.Fatalf("err=%v, want %v", err, ErrDetachPortRequired)
	}
}

func TestListRemoteCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	runtime := Runtime{
		LogLevel:     logging.NewDefaultLevel(),
		AttachRemote: nil,
		DetachRemote: nil,
		ListRemote: func(_ context.Context, endpoint adapterusbip.Endpoint) ([]adapterusbip.Device, error) {
			if endpoint.Host != localTCPHost || endpoint.Port != 4321 {
				t.Fatalf("endpoint=%+v, want host=%s port=4321", endpoint, localTCPHost)
			}

			return []adapterusbip.Device{
				{
					Path:                "/sys/bus/usb/devices/1-1",
					BusID:               testBusID,
					Busnum:              0,
					Devnum:              0,
					Speed:               0,
					IDVendor:            0x2fe3,
					IDProduct:           0x0001,
					BCDDevice:           0,
					BDeviceClass:        0xef,
					BDeviceSubClass:     0x02,
					BDeviceProtocol:     0x01,
					BConfigurationValue: 0,
					BNumConfigurations:  0,
					Interfaces: []adapterusbip.Interface{
						{Class: 0x02, SubClass: 0x02, Protocol: 0x00},
					},
				},
			}, nil
		},
	}

	cmd := NewRootCommandWithRuntime(nil, &stdout, nil, runtime)
	cmd.SetArgs([]string{"--tcp-port", "4321", listCommand, "-r", localTCPHost})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute list remote: %v", err)
	}

	want := `Exportable USB devices
======================
 - 127.0.0.1
        1-1: unknown vendor : unknown product (2fe3:0001)
           : /sys/bus/usb/devices/1-1
           : Miscellaneous Device / Abstract (modem) / Interface Association (ef/02/01)
           :  0 - Communications / Abstract (modem) / unknown protocol (02/02/00)
`

	if got := stdout.String(); got != want {
		t.Fatalf("unexpected output\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestListRemoteCommandNoDevices(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	runtime := Runtime{
		LogLevel:     logging.NewDefaultLevel(),
		AttachRemote: nil,
		DetachRemote: nil,
		ListRemote: func(context.Context, adapterusbip.Endpoint) ([]adapterusbip.Device, error) {
			return nil, nil
		},
	}

	cmd := NewRootCommandWithRuntime(nil, &stdout, nil, runtime)
	cmd.SetArgs([]string{listCommand, "-r", localTCPHost})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute list remote: %v", err)
	}

	want := "usbip: info: no exportable devices found on 127.0.0.1\n"
	if got := stdout.String(); got != want {
		t.Fatalf("unexpected output: want %q, got %q", want, got)
	}
}

func TestUSBIPCompatibleSubcommandHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: attachCommand,
			args: []string{attachCommand, helpFlag},
			want: `usage: usb-quic attach <args>
    -r, --remote=<host>      The machine with exported USB devices
    -b, --busid=<busid>    Busid of the device on <host>
    -d, --device=<devid>    Id of the virtual UDC on <host>
`,
		},
		{
			name: listCommand,
			args: []string{listCommand, helpFlag},
			want: `usage: usb-quic list [-p|--parsable] <args>
    -p, --parsable         Parsable list format
    -r, --remote=<host>    List the exportable USB devices on <host>
    -l, --local            List the local USB devices
    -d, --device           List the local USB gadgets bound to usbip-vudc
`,
		},
		{
			name: detachCommand,
			args: []string{detachCommand, helpFlag},
			want: `usage: usb-quic detach <args>
    -p, --port=<port>    vhci_hcd port the device is on
`,
		},
		{
			name: bindCommand,
			args: []string{bindCommand, helpFlag},
			want: `usage: usb-quic bind <args>
    -b, --busid=<busid>    Bind usbip-host.ko to device on <busid>
`,
		},
		{
			name: unbindCommand,
			args: []string{unbindCommand, helpFlag},
			want: `usage: usb-quic unbind <args>
    -b, --busid=<busid>    Unbind usbip-host.ko from device on <busid>
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer

			cmd := NewRootCommand(nil, &stdout, nil)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("execute help: %v", err)
			}

			if got := stdout.String(); got != tt.want {
				t.Fatalf("unexpected help\nwant:\n%s\ngot:\n%s", tt.want, got)
			}
		})
	}
}
