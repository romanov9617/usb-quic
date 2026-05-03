package cli

import (
	"bytes"
	"errors"
	"testing"
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
		{name: attachCommand, args: []string{attachCommand, "-r", localTCPHost, "-b", testBusID}},
		{name: "attach device", args: []string{attachCommand, "-r", localTCPHost, "-d", "0"}},
		{name: detachCommand, args: []string{detachCommand, "-p", "0"}},
		{name: "list local", args: []string{listCommand, "-l"}},
		{name: "list remote", args: []string{listCommand, "-r", localTCPHost}},
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
