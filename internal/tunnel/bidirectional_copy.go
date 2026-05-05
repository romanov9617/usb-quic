// Package tunnel contains transport-neutral byte tunneling primitives.
package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
)

const copyDirections = 2

// Endpoint is one side of a bidirectional byte stream.
type Endpoint interface {
	io.ReadWriteCloser
	CloseRead() error
	CloseWrite() error
}

// BidirectionalCopy copies bytes between left and right until both directions
// complete, either endpoint fails, or ctx is canceled. A clean read shutdown in
// one direction closes only the peer write side, so the opposite direction can
// keep carrying bytes.
func BidirectionalCopy(ctx context.Context, left, right Endpoint) error {
	errCh := make(chan error, copyDirections)

	go func() {
		errCh <- copyDirection(right, left)
	}()

	go func() {
		errCh <- copyDirection(left, right)
	}()

	var err error

	for range copyDirections {
		select {
		case copyErr := <-errCh:
			if copyErr != nil {
				err = errors.Join(err, copyErr)
				_ = left.Close()
				_ = right.Close()
			}

		case <-ctx.Done():
			_ = left.Close()
			_ = right.Close()

			return fmt.Errorf("bidirectional copy canceled: %w", ctx.Err())
		}
	}

	return err
}

func copyDirection(dst, src Endpoint) error {
	_, err := io.Copy(dst, src)

	if isBenign(err) {
		err = dst.CloseWrite()
		if err != nil {
			return fmt.Errorf("close write side: %w", err)
		}

		return nil
	}

	_ = dst.Close()
	_ = src.CloseRead()

	return fmt.Errorf("copy bytes: %w", err)
}

func isBenign(err error) bool {
	return err == nil ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.ErrClosedPipe)
}
