package daemon

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"usb-quic/internal/config"
	"usb-quic/internal/tunnel"
)

func TestHandleConnectionCopiesBytesBothDirections(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	clientPeer, clientConn := newMemoryConnPair()
	upstreamPeer, upstreamConn := newMemoryConnPair()
	daemon := testDaemon(func(context.Context, config.Daemon) (tunnel.Endpoint, error) {
		return upstreamConn, nil
	})
	done := make(chan struct{})

	go func() {
		daemon.handleConnection(ctx, clientConn)
		close(done)
	}()

	writeMemoryString(t, clientPeer, "request")

	if got := readMemoryString(t, upstreamPeer, len("request")); got != "request" {
		t.Fatalf("upstream read %q, want %q", got, "request")
	}

	writeMemoryString(t, upstreamPeer, "response")

	if got := readMemoryString(t, clientPeer, len("response")); got != "response" {
		t.Fatalf("client read %q, want %q", got, "response")
	}

	closeMemoryWrite(t, clientPeer)
	closeMemoryWrite(t, upstreamPeer)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not stop")
	}
}

func TestHandleConnectionClosesClientOnDialError(t *testing.T) {
	t.Parallel()

	clientPeer, clientConn := newMemoryConnPair()
	daemon := testDaemon(func(context.Context, config.Daemon) (tunnel.Endpoint, error) {
		return nil, errBusy
	})
	done := make(chan struct{})

	go func() {
		daemon.handleConnection(t.Context(), clientConn)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not stop")
	}

	_, err := clientPeer.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("expected closed client connection")
	}
}

func testDaemon(dial dialFunc) *Daemon {
	daemon := New(config.Daemon{
		BindIPv4:   false,
		BindIPv6:   false,
		DeviceMode: false,
		Daemonize:  false,
		Debug:      false,
		PIDFile:    "",
		TCPPort:    testTCPPort,
	}, testLogger())
	daemon.dial = dial

	return daemon
}

func newMemoryConnPair() (*memoryConn, *memoryConn) {
	leftToRightReader, leftToRightWriter := io.Pipe()
	rightToLeftReader, rightToLeftWriter := io.Pipe()

	left := &memoryConn{
		reader:         rightToLeftReader,
		writer:         leftToRightWriter,
		closeReadOnce:  sync.Once{},
		closeWriteOnce: sync.Once{},
		localAddr:      fakeAddr("left"),
		remoteAddr:     fakeAddr("right"),
	}
	right := &memoryConn{
		reader:         leftToRightReader,
		writer:         rightToLeftWriter,
		closeReadOnce:  sync.Once{},
		closeWriteOnce: sync.Once{},
		localAddr:      fakeAddr("right"),
		remoteAddr:     fakeAddr("left"),
	}

	return left, right
}

func writeMemoryString(t *testing.T, conn *memoryConn, value string) {
	t.Helper()

	errs := make(chan error, 1)

	go func() {
		_, err := io.WriteString(conn, value)
		errs <- err
	}()

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("write %q: %v", value, err)
		}
	case <-time.After(time.Second):
		t.Fatalf("write %q timed out", value)
	}
}

func readMemoryString(t *testing.T, conn *memoryConn, size int) string {
	t.Helper()

	buffer := make([]byte, size)
	errs := make(chan error, 1)

	go func() {
		_, err := io.ReadFull(conn, buffer)
		errs <- err
	}()

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("read %d bytes: %v", size, err)
		}
	case <-time.After(time.Second):
		t.Fatalf("read %d bytes timed out", size)
	}

	return string(buffer)
}

func closeMemoryWrite(t *testing.T, conn *memoryConn) {
	t.Helper()

	err := conn.CloseWrite()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("close write: %v", err)
	}
}

type memoryConn struct {
	reader         *io.PipeReader
	writer         *io.PipeWriter
	closeReadOnce  sync.Once
	closeWriteOnce sync.Once
	localAddr      net.Addr
	remoteAddr     net.Addr
}

func (conn *memoryConn) Read(buffer []byte) (int, error) {
	return conn.reader.Read(buffer)
}

func (conn *memoryConn) Write(buffer []byte) (int, error) {
	return conn.writer.Write(buffer)
}

func (conn *memoryConn) Close() error {
	var err error

	conn.closeReadOnce.Do(func() {
		err = errors.Join(err, conn.reader.CloseWithError(io.ErrClosedPipe))
	})
	conn.closeWriteOnce.Do(func() {
		err = errors.Join(err, conn.writer.CloseWithError(io.ErrClosedPipe))
	})

	return err
}

func (conn *memoryConn) CloseRead() error {
	var err error

	conn.closeReadOnce.Do(func() {
		err = conn.reader.CloseWithError(io.ErrClosedPipe)
	})

	return err
}

func (conn *memoryConn) CloseWrite() error {
	var err error

	conn.closeWriteOnce.Do(func() {
		err = conn.writer.Close()
	})

	return err
}

func (conn *memoryConn) LocalAddr() net.Addr {
	return conn.localAddr
}

func (conn *memoryConn) RemoteAddr() net.Addr {
	return conn.remoteAddr
}

func (conn *memoryConn) SetDeadline(time.Time) error {
	return nil
}

func (conn *memoryConn) SetReadDeadline(time.Time) error {
	return nil
}

func (conn *memoryConn) SetWriteDeadline(time.Time) error {
	return nil
}
