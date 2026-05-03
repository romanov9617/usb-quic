package daemon

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"usb-quic/internal/adapter/logging"
	"usb-quic/internal/config"
)

const testTCPPort = 3241

var errBusy = errors.New("busy")

func TestRunCreatesAndRemovesPIDFile(t *testing.T) {
	t.Parallel()

	pidFile := filepath.Join(t.TempDir(), "usbipd.pid")
	ctx, cancel := context.WithCancel(t.Context())

	defer cancel()

	errs := make(chan error, 1)
	listener := newFakeListener()

	go func() {
		errs <- runWithListener(ctx, config.Daemon{
			BindIPv4:   true,
			BindIPv6:   false,
			DeviceMode: false,
			Daemonize:  false,
			Debug:      false,
			PIDFile:    pidFile,
			TCPPort:    testTCPPort,
		}, testLogger(), fakeListen(listener, nil))
	}()

	waitForPIDFile(t, pidFile)

	//nolint:gosec // The test reads the PID file path it created under t.TempDir.
	content, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read pid file: %v", err)
	}

	if strings.TrimSpace(string(content)) == "" {
		t.Fatal("pid file is empty")
	}

	cancel()

	waitRunStopped(t, errs)
	waitListenerClosed(t, listener)

	_, err = os.Stat(pidFile)
	if !os.IsNotExist(err) {
		t.Fatalf("pid file still exists: %v", err)
	}
}

func TestRunReturnsPIDFileWriteError(t *testing.T) {
	t.Parallel()

	err := runWithListener(t.Context(), config.Daemon{
		BindIPv4:   false,
		BindIPv6:   false,
		DeviceMode: false,
		Daemonize:  false,
		Debug:      false,
		PIDFile:    filepath.Join(t.TempDir(), "missing", "usbipd.pid"),
		TCPPort:    testTCPPort,
	}, testLogger(), fakeListen(newFakeListener(), nil))
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "write pid file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunReturnsListenError(t *testing.T) {
	t.Parallel()

	err := runWithListener(t.Context(), config.Daemon{
		BindIPv4:   true,
		BindIPv6:   false,
		DeviceMode: false,
		Daemonize:  false,
		Debug:      false,
		PIDFile:    "",
		TCPPort:    testTCPPort,
	}, testLogger(), fakeListen(nil, errBusy))
	if !errors.Is(err, errBusy) {
		t.Fatalf("expected %v, got %v", errBusy, err)
	}
}

func TestRunClosesListenerOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	defer cancel()

	errs := make(chan error, 1)
	listener := newFakeListener()

	go func() {
		errs <- runWithListener(ctx, config.Daemon{
			BindIPv4:   true,
			BindIPv6:   false,
			DeviceMode: false,
			Daemonize:  false,
			Debug:      false,
			PIDFile:    "",
			TCPPort:    testTCPPort,
		}, testLogger(), fakeListen(listener, nil))
	}()

	cancel()

	waitRunStopped(t, errs)
	waitListenerClosed(t, listener)
}

func TestListenAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.Daemon
		want string
	}{
		{
			name: "dual stack default",
			cfg: config.Daemon{
				BindIPv4:   false,
				BindIPv6:   false,
				DeviceMode: false,
				Daemonize:  false,
				Debug:      false,
				PIDFile:    "",
				TCPPort:    testTCPPort,
			},
			want: ":" + "3241",
		},
		{
			name: "ipv4 only",
			cfg: config.Daemon{
				BindIPv4:   true,
				BindIPv6:   false,
				DeviceMode: false,
				Daemonize:  false,
				Debug:      false,
				PIDFile:    "",
				TCPPort:    testTCPPort,
			},
			want: "0.0.0.0:3241",
		},
		{
			name: "ipv6 only",
			cfg: config.Daemon{
				BindIPv4:   false,
				BindIPv6:   true,
				DeviceMode: false,
				Daemonize:  false,
				Debug:      false,
				PIDFile:    "",
				TCPPort:    testTCPPort,
			},
			want: "[::]:3241",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := listenAddress(tt.cfg)
			if got != tt.want {
				t.Fatalf("listenAddress=%q, want=%q", got, tt.want)
			}
		})
	}
}

func waitForPIDFile(t *testing.T, path string) {
	t.Helper()

	deadline := time.After(time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)

	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("pid file was not created")
		case <-ticker.C:
			_, err := os.Stat(path)
			if err == nil {
				return
			}
		}
	}
}

func waitRunStopped(t *testing.T, errs <-chan error) {
	t.Helper()

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not stop")
	}
}

func waitListenerClosed(t *testing.T, listener *fakeListener) {
	t.Helper()

	select {
	case <-listener.closed:
	case <-time.After(time.Second):
		t.Fatal("listener was not closed")
	}
}

func fakeListen(listener net.Listener, err error) listenFunc {
	return func(_ context.Context, _ config.Daemon) (net.Listener, error) {
		return listener, err
	}
}

func testLogger() *logging.Logger {
	return logging.NewTextLogger(io.Discard, logging.NewDefaultLevel())
}

type fakeListener struct {
	closeOnce sync.Once
	closed    chan struct{}
}

func newFakeListener() *fakeListener {
	return &fakeListener{
		closeOnce: sync.Once{},
		closed:    make(chan struct{}),
	}
}

func (listener *fakeListener) Accept() (net.Conn, error) {
	return nil, net.ErrClosed
}

func (listener *fakeListener) Close() error {
	listener.closeOnce.Do(func() {
		close(listener.closed)
	})

	return nil
}

func (listener *fakeListener) Addr() net.Addr {
	return fakeAddr("")
}

type fakeAddr string

func (addr fakeAddr) Network() string {
	return "tcp"
}

func (addr fakeAddr) String() string {
	return string(addr)
}
