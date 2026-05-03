package cli

import (
	"bytes"
	"errors"
	"testing"
)

const testBusID = "1-1"

func TestRootHelpMatchesUSBIPShape(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	cmd := NewRootCommand(nil, &stdout, nil)
	cmd.SetArgs([]string{"help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute help: %v", err)
	}

	want := `usage: usb-quic [-v] [--debug] [--log] [--tcp-port PORT] [version] [help] <command> <args>

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
		{name: "attach", args: []string{"attach", "-r", localTCPHost, "-b", testBusID}},
		{name: "detach", args: []string{"detach", "-p", "0"}},
		{name: "list local", args: []string{"list", "-l"}},
		{name: "list remote", args: []string{"list", "-r", localTCPHost}},
		{name: "bind", args: []string{"bind", "-b", testBusID}},
		{name: "unbind", args: []string{"unbind", "-b", testBusID}},
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
