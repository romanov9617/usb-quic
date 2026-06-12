package daemon

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"usb-quic/internal/config"
)

const testPIDFile = "/tmp/usbipd.pid"

func TestRootHelpMatchesProductShape(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	cmd := NewRootCommand(nil, &stdout, nil)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute help: %v", err)
	}

	want := `usage: usb-quicd [options]

  --transport MODE       Required: tcp, quic-client, or quic-server
  --tcp-listen ADDRESS   TCP listen address for tcp and quic-client modes
  --quic-server ADDRESS  QUIC server address for quic-client mode
  --quic-listen ADDRESS  QUIC listen address for quic-server mode
  --upstream ADDRESS     TCP upstream address for tcp and quic-server modes
  --insecure-dev-tls     Use development-only insecure TLS
  -d, --debug            Enable debug logging
  -v, --version          Show version
`

	if got := stdout.String(); got != want {
		t.Fatalf("unexpected help\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestLegacyDaemonFlagsParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "ipv4", args: []string{"--ipv4"}},
		{name: "ipv6", args: []string{"--ipv6"}},
		{name: "device", args: []string{"--device"}},
		{name: "daemon", args: []string{"--daemon"}},
		{name: "debug", args: []string{"--debug"}},
		{name: "pid", args: []string{"--pid", testPIDFile}},
		{name: "attached pid", args: []string{"-P" + testPIDFile}},
		{name: "tcp port", args: []string{"--tcp-port", "3241"}},
		{name: "short flags", args: []string{"-4", "-6", "-e", "-D", "-d", "-P", testPIDFile, "-t", "3241"}},
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

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	cmd := NewRootCommand(nil, &stdout, nil)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute version: %v", err)
	}

	if got := stdout.String(); got != "usb-quicd "+version+"\n" {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestRunPassesConfigToService(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{
		called: false,
		config: config.Daemon{
			BindIPv4:       false,
			BindIPv6:       false,
			DeviceMode:     false,
			Daemonize:      false,
			Debug:          false,
			DevInsecureTLS: false,
			PIDFile:        "",
			QUICAddr:       "",
			QUICListen:     "",
			TCPListen:      "",
			TCPPort:        0,
			TransportMode:  "",
			Upstream:       "",
		},
		err: ErrNotImplemented,
	}
	runtime := Runtime{
		LogLevel: nil,
		Run:      runner.Run,
	}

	cmd := NewRootCommandWithRuntime(nil, bytes.NewBuffer(nil), bytes.NewBuffer(nil), runtime)
	cmd.SetArgs([]string{"-4", "-6", "-e", "-D", "-d", "-P", testPIDFile, "-t", "3241"})

	err := cmd.Execute()
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	if !runner.called {
		t.Fatal("runner was not called")
	}

	want := config.Daemon{
		BindIPv4:       true,
		BindIPv6:       true,
		DeviceMode:     true,
		Daemonize:      true,
		Debug:          true,
		DevInsecureTLS: false,
		PIDFile:        testPIDFile,
		QUICAddr:       defaultQUICAddress,
		QUICListen:     defaultQUICAddress,
		TCPListen:      "",
		TCPPort:        3241,
		TransportMode:  defaultTransportMode,
		Upstream:       defaultUpstream,
	}
	if runner.config != want {
		t.Fatalf("unexpected config: got=%+v want=%+v", runner.config, want)
	}
}

func TestRunPassesUSBQUICConfigToService(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{
		called: false,
		config: config.Daemon{
			BindIPv4:       false,
			BindIPv6:       false,
			DeviceMode:     false,
			Daemonize:      false,
			Debug:          false,
			DevInsecureTLS: false,
			PIDFile:        "",
			QUICAddr:       "",
			QUICListen:     "",
			TCPListen:      "",
			TCPPort:        0,
			TransportMode:  "",
			Upstream:       "",
		},
		err: ErrNotImplemented,
	}
	runtime := Runtime{
		LogLevel: nil,
		Run:      runner.Run,
	}

	cmd := NewRootCommandWithRuntime(nil, bytes.NewBuffer(nil), bytes.NewBuffer(nil), runtime)
	cmd.SetArgs([]string{
		"--transport", "quic-client",
		"--quic-server", "127.0.0.1:14242",
		"--quic-listen", "127.0.0.1:24242",
		"--tcp-listen", "127.0.0.1:13240",
		"--upstream", "127.0.0.1:19000",
		"--insecure-dev-tls",
	})

	err := cmd.Execute()
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	want := config.Daemon{
		BindIPv4:       false,
		BindIPv6:       false,
		DeviceMode:     false,
		Daemonize:      false,
		Debug:          false,
		DevInsecureTLS: true,
		PIDFile:        "",
		QUICAddr:       "127.0.0.1:14242",
		QUICListen:     "127.0.0.1:24242",
		TCPListen:      "127.0.0.1:13240",
		TCPPort:        3240,
		TransportMode:  "quic-client",
		Upstream:       "127.0.0.1:19000",
	}
	if runner.config != want {
		t.Fatalf("unexpected config: got=%+v want=%+v", runner.config, want)
	}
}

func TestVersionDoesNotCallService(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{
		called: false,
		config: config.Daemon{
			BindIPv4:       false,
			BindIPv6:       false,
			DeviceMode:     false,
			Daemonize:      false,
			Debug:          false,
			DevInsecureTLS: false,
			PIDFile:        "",
			QUICAddr:       "",
			QUICListen:     "",
			TCPListen:      "",
			TCPPort:        0,
			TransportMode:  "",
			Upstream:       "",
		},
		err: nil,
	}
	runtime := Runtime{
		LogLevel: nil,
		Run:      runner.Run,
	}

	var stdout bytes.Buffer

	cmd := NewRootCommandWithRuntime(nil, &stdout, bytes.NewBuffer(nil), runtime)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute version: %v", err)
	}

	if runner.called {
		t.Fatal("runner was called")
	}
}

type recordingRunner struct {
	called bool
	config config.Daemon
	err    error
}

func (runner *recordingRunner) Run(_ context.Context, config config.Daemon) error {
	runner.called = true
	runner.config = config

	return runner.err
}
