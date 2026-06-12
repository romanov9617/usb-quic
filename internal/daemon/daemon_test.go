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
	"usb-quic/internal/tunnel"
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
		daemon := New(config.Daemon{
			BindIPv4:       true,
			BindIPv6:       false,
			DeviceMode:     false,
			Daemonize:      false,
			Debug:          false,
			DevInsecureTLS: false,
			PIDFile:        pidFile,
			QUICAddr:       "",
			QUICListen:     "",
			TCPListen:      "",
			TCPPort:        testTCPPort,
			TransportMode:  "",
			Upstream:       "",
		}, testLogger(), WithListener(listener))
		errs <- daemon.Run(ctx)
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

	daemon := New(config.Daemon{
		BindIPv4:       false,
		BindIPv6:       false,
		DeviceMode:     false,
		Daemonize:      false,
		Debug:          false,
		DevInsecureTLS: false,
		PIDFile:        filepath.Join(t.TempDir(), "missing", "usbipd.pid"),
		QUICAddr:       "",
		QUICListen:     "",
		TCPListen:      "",
		TCPPort:        testTCPPort,
		TransportMode:  "",
		Upstream:       "",
	}, testLogger(), WithListener(newFakeListener()))

	err := daemon.Run(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "write pid file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunReturnsListenError(t *testing.T) {
	t.Parallel()

	daemon := New(config.Daemon{
		BindIPv4:       true,
		BindIPv6:       false,
		DeviceMode:     false,
		Daemonize:      false,
		Debug:          false,
		DevInsecureTLS: false,
		PIDFile:        "",
		QUICAddr:       "",
		QUICListen:     "",
		TCPListen:      "",
		TCPPort:        -1,
		TransportMode:  "",
		Upstream:       "",
	}, testLogger())

	err := daemon.Run(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunClosesListenerOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	defer cancel()

	errs := make(chan error, 1)
	listener := newFakeListener()

	go func() {
		daemon := New(config.Daemon{
			BindIPv4:       true,
			BindIPv6:       false,
			DeviceMode:     false,
			Daemonize:      false,
			Debug:          false,
			DevInsecureTLS: false,
			PIDFile:        "",
			QUICAddr:       "",
			QUICListen:     "",
			TCPListen:      "",
			TCPPort:        testTCPPort,
			TransportMode:  "",
			Upstream:       "",
		}, testLogger(), WithListener(listener))
		errs <- daemon.Run(ctx)
	}()

	cancel()

	waitRunStopped(t, errs)
	waitListenerClosed(t, listener)
}

func TestRunWaitsForActiveConnectionHandlers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	clientPeer, clientConn := newMemoryConnPair()
	upstreamPeer, upstreamConn := newMemoryConnPair()
	listener := newSingleConnListener(clientConn)
	handlerDone := make(chan struct{})
	errs := make(chan error, 1)

	defer func() {
		_ = clientPeer.Close()
	}()
	defer func() {
		_ = upstreamPeer.Close()
	}()

	daemon := New(config.Daemon{
		BindIPv4:       true,
		BindIPv6:       false,
		DeviceMode:     false,
		Daemonize:      false,
		Debug:          false,
		DevInsecureTLS: false,
		PIDFile:        "",
		QUICAddr:       "",
		QUICListen:     "",
		TCPListen:      "",
		TCPPort:        testTCPPort,
		TransportMode:  "",
		Upstream:       "",
	}, testLogger(), WithListener(listener), WithStreamOpener(blockingStreamOpener{
		endpoint: upstreamConn,
		done:     handlerDone,
	}))

	go func() {
		errs <- daemon.Run(ctx)
	}()

	waitConnectionAccepted(t, listener)
	cancel()

	select {
	case err := <-errs:
		t.Fatalf("run returned before active handler finished: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(handlerDone)
	waitRunStopped(t, errs)
}

func TestListenAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.Daemon
		want string
	}{
		{
			name: "explicit address",
			cfg:  daemonConfig("127.0.0.1:13240", ""),
			want: "127.0.0.1:13240",
		},
		{
			name: "dual stack default",
			cfg: config.Daemon{
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
				TCPPort:        testTCPPort,
				TransportMode:  "",
				Upstream:       "",
			},
			want: ":" + "3241",
		},
		{
			name: "ipv4 only",
			cfg: config.Daemon{
				BindIPv4:       true,
				BindIPv6:       false,
				DeviceMode:     false,
				Daemonize:      false,
				Debug:          false,
				DevInsecureTLS: false,
				PIDFile:        "",
				QUICAddr:       "",
				QUICListen:     "",
				TCPListen:      "",
				TCPPort:        testTCPPort,
				TransportMode:  "",
				Upstream:       "",
			},
			want: "0.0.0.0:3241",
		},
		{
			name: "ipv6 only",
			cfg: config.Daemon{
				BindIPv4:       false,
				BindIPv6:       true,
				DeviceMode:     false,
				Daemonize:      false,
				Debug:          false,
				DevInsecureTLS: false,
				PIDFile:        "",
				QUICAddr:       "",
				QUICListen:     "",
				TCPListen:      "",
				TCPPort:        testTCPPort,
				TransportMode:  "",
				Upstream:       "",
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

func TestUpstreamAddress(t *testing.T) {
	t.Parallel()

	if got := upstreamAddress(daemonConfig("", "127.0.0.1:19000")); got != "127.0.0.1:19000" {
		t.Fatalf("upstreamAddress=%q, want explicit upstream", got)
	}

	if got := upstreamAddress(daemonConfig("", "")); got != defaultUpstreamAddress() {
		t.Fatalf("upstreamAddress=%q, want default %q", got, defaultUpstreamAddress())
	}
}

func daemonConfig(tcpListen, upstream string) config.Daemon {
	return config.Daemon{
		BindIPv4:       false,
		BindIPv6:       false,
		DeviceMode:     false,
		Daemonize:      false,
		Debug:          false,
		DevInsecureTLS: false,
		PIDFile:        "",
		QUICAddr:       "",
		QUICListen:     "",
		TCPListen:      tcpListen,
		TCPPort:        0,
		TransportMode:  "",
		Upstream:       upstream,
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

type singleConnListener struct {
	conn       net.Conn
	accepted   chan struct{}
	closed     chan struct{}
	acceptOnce sync.Once
	closeOnce  sync.Once
}

func newSingleConnListener(conn net.Conn) *singleConnListener {
	return &singleConnListener{
		conn:       conn,
		accepted:   make(chan struct{}),
		closed:     make(chan struct{}),
		acceptOnce: sync.Once{},
		closeOnce:  sync.Once{},
	}
}

func (listener *singleConnListener) Accept() (net.Conn, error) {
	var conn net.Conn

	listener.acceptOnce.Do(func() {
		conn = listener.conn
		close(listener.accepted)
	})

	if conn != nil {
		return conn, nil
	}

	<-listener.closed

	return nil, net.ErrClosed
}

func (listener *singleConnListener) Close() error {
	listener.closeOnce.Do(func() {
		close(listener.closed)
	})

	return nil
}

func (listener *singleConnListener) Addr() net.Addr {
	return fakeAddr("")
}

func waitConnectionAccepted(t *testing.T, listener *singleConnListener) {
	t.Helper()

	select {
	case <-listener.accepted:
	case <-time.After(time.Second):
		t.Fatal("connection was not accepted")
	}
}

type blockingStreamOpener struct {
	endpoint *memoryConn
	done     <-chan struct{}
}

//nolint:ireturn // Test helper returns the transport boundary interface under test.
func (opener blockingStreamOpener) OpenStream(context.Context) (tunnel.Endpoint, error) {
	return blockingEndpoint{
		Endpoint: opener.endpoint,
		done:     opener.done,
	}, nil
}

type blockingEndpoint struct {
	tunnel.Endpoint

	done <-chan struct{}
}

func (endpoint blockingEndpoint) Close() error {
	<-endpoint.done

	return endpoint.Endpoint.Close()
}

type fakeAddr string

func (addr fakeAddr) Network() string {
	return "tcp"
}

func (addr fakeAddr) String() string {
	return string(addr)
}
