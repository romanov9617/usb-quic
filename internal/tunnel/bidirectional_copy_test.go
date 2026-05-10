package tunnel

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

var errRead = errors.New("read failed")

func TestBidirectionalCopyCopiesBothDirections(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	leftPeer, leftTunnel := newMemoryPair()
	rightPeer, rightTunnel := newMemoryPair()

	errs := make(chan error, 1)
	go func() {
		errs <- BidirectionalCopy(ctx, leftTunnel, rightTunnel)
	}()

	writeString(t, leftPeer, "ping")

	if got := readString(t, rightPeer, len("ping")); got != "ping" {
		t.Fatalf("right peer read %q, want %q", got, "ping")
	}

	writeString(t, rightPeer, "pong")

	if got := readString(t, leftPeer, len("pong")); got != "pong" {
		t.Fatalf("left peer read %q, want %q", got, "pong")
	}

	closePeerWrite(t, leftPeer)

	if got := readEOF(t, rightPeer); got != "" {
		t.Fatalf("right peer read %q before EOF, want empty", got)
	}

	writeString(t, rightPeer, "after-eof")

	if got := readString(t, leftPeer, len("after-eof")); got != "after-eof" {
		t.Fatalf("left peer read %q, want %q", got, "after-eof")
	}

	closePeerWrite(t, rightPeer)

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("bidirectional copy: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("bidirectional copy did not stop")
	}
}

func TestBidirectionalCopyStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	leftPeer, leftTunnel := newMemoryPair()
	rightPeer, rightTunnel := newMemoryPair()

	defer func() {
		_ = leftPeer.Close()
	}()
	defer func() {
		_ = rightPeer.Close()
	}()

	errs := make(chan error, 1)
	go func() {
		errs <- BidirectionalCopy(ctx, leftTunnel, rightTunnel)
	}()

	cancel()

	select {
	case err := <-errs:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("bidirectional copy err=%v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("bidirectional copy did not stop")
	}
}

func TestCopyDirectionClosesWriteOnEOF(t *testing.T) {
	t.Parallel()

	destination := &recordingEndpoint{
		closeWriteCalled: false,
	}

	err := copyDirection(destination, eofEndpoint{})
	if err != nil {
		t.Fatalf("copyDirection: %v", err)
	}

	if !destination.closeWriteCalled {
		t.Fatal("CloseWrite was not called")
	}
}

func TestCopyDirectionReturnsCopyError(t *testing.T) {
	t.Parallel()

	err := copyDirection(nopEndpoint{}, errorReaderEndpoint{})
	if !errors.Is(err, errRead) {
		t.Fatalf("copyDirection err=%v, want %v", err, errRead)
	}
}

func writeString(t *testing.T, endpoint *memoryEndpoint, value string) {
	t.Helper()

	errs := make(chan error, 1)

	go func() {
		_, err := io.WriteString(endpoint, value)
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

func readString(t *testing.T, endpoint *memoryEndpoint, size int) string {
	t.Helper()

	buffer := make([]byte, size)
	errs := make(chan error, 1)

	go func() {
		_, err := io.ReadFull(endpoint, buffer)
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

func readEOF(t *testing.T, endpoint *memoryEndpoint) string {
	t.Helper()

	buffer := make([]byte, 32)
	results := make(chan readResult, 1)

	go func() {
		n, err := endpoint.Read(buffer)
		results <- readResult{
			size: n,
			err:  err,
		}
	}()

	select {
	case result := <-results:
		if !errors.Is(result.err, io.EOF) {
			t.Fatalf("read err=%v, want EOF", result.err)
		}

		return string(buffer[:result.size])
	case <-time.After(time.Second):
		t.Fatal("read EOF timed out")

		return ""
	}
}

func closePeerWrite(t *testing.T, endpoint *memoryEndpoint) {
	t.Helper()

	err := endpoint.CloseWrite()
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("close write: %v", err)
	}
}

func newMemoryPair() (*memoryEndpoint, *memoryEndpoint) {
	leftToRightReader, leftToRightWriter := io.Pipe()
	rightToLeftReader, rightToLeftWriter := io.Pipe()

	left := &memoryEndpoint{
		reader:         rightToLeftReader,
		writer:         leftToRightWriter,
		closeReadOnce:  sync.Once{},
		closeWriteOnce: sync.Once{},
	}
	right := &memoryEndpoint{
		reader:         leftToRightReader,
		writer:         rightToLeftWriter,
		closeReadOnce:  sync.Once{},
		closeWriteOnce: sync.Once{},
	}

	return left, right
}

type memoryEndpoint struct {
	reader         *io.PipeReader
	writer         *io.PipeWriter
	closeReadOnce  sync.Once
	closeWriteOnce sync.Once
}

type nopEndpoint struct{}

func (endpoint nopEndpoint) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (endpoint nopEndpoint) Write(buffer []byte) (int, error) {
	return len(buffer), nil
}

func (endpoint nopEndpoint) Close() error {
	return nil
}

func (endpoint nopEndpoint) CloseRead() error {
	return nil
}

func (endpoint nopEndpoint) CloseWrite() error {
	return nil
}

type errorReaderEndpoint struct{}

func (endpoint errorReaderEndpoint) Read(_ []byte) (int, error) {
	return 0, errRead
}

func (endpoint errorReaderEndpoint) Write(buffer []byte) (int, error) {
	return len(buffer), nil
}

func (endpoint errorReaderEndpoint) Close() error {
	return nil
}

func (endpoint errorReaderEndpoint) CloseRead() error {
	return nil
}

func (endpoint errorReaderEndpoint) CloseWrite() error {
	return nil
}

type eofEndpoint struct{}

func (endpoint eofEndpoint) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (endpoint eofEndpoint) Write(buffer []byte) (int, error) {
	return len(buffer), nil
}

func (endpoint eofEndpoint) Close() error {
	return nil
}

func (endpoint eofEndpoint) CloseRead() error {
	return nil
}

func (endpoint eofEndpoint) CloseWrite() error {
	return nil
}

type recordingEndpoint struct {
	closeWriteCalled bool
}

func (endpoint *recordingEndpoint) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (endpoint *recordingEndpoint) Write(buffer []byte) (int, error) {
	return len(buffer), nil
}

func (endpoint *recordingEndpoint) Close() error {
	return nil
}

func (endpoint *recordingEndpoint) CloseRead() error {
	return nil
}

func (endpoint *recordingEndpoint) CloseWrite() error {
	endpoint.closeWriteCalled = true

	return nil
}

func (endpoint *memoryEndpoint) Read(buffer []byte) (int, error) {
	return endpoint.reader.Read(buffer)
}

func (endpoint *memoryEndpoint) Write(buffer []byte) (int, error) {
	return endpoint.writer.Write(buffer)
}

func (endpoint *memoryEndpoint) Close() error {
	var err error

	endpoint.closeReadOnce.Do(func() {
		err = errors.Join(err, endpoint.reader.CloseWithError(io.ErrClosedPipe))
	})
	endpoint.closeWriteOnce.Do(func() {
		err = errors.Join(err, endpoint.writer.CloseWithError(io.ErrClosedPipe))
	})

	return err
}

func (endpoint *memoryEndpoint) CloseRead() error {
	var err error

	endpoint.closeReadOnce.Do(func() {
		err = endpoint.reader.CloseWithError(io.ErrClosedPipe)
	})

	return err
}

func (endpoint *memoryEndpoint) CloseWrite() error {
	var err error

	endpoint.closeWriteOnce.Do(func() {
		err = endpoint.writer.Close()
	})

	return err
}

type readResult struct {
	size int
	err  error
}
