package daemon

import (
	"fmt"
	"net"
)

type connEndpoint struct {
	net.Conn
}

func newConnEndpoint(conn net.Conn) *connEndpoint {
	return &connEndpoint{
		Conn: conn,
	}
}

func (endpoint *connEndpoint) CloseRead() error {
	if conn, ok := endpoint.Conn.(interface{ CloseRead() error }); ok {
		err := conn.CloseRead()
		if err != nil {
			return fmt.Errorf("close read side: %w", err)
		}

		return nil
	}

	err := endpoint.Close()
	if err != nil {
		return fmt.Errorf("close connection: %w", err)
	}

	return nil
}

func (endpoint *connEndpoint) CloseWrite() error {
	if conn, ok := endpoint.Conn.(interface{ CloseWrite() error }); ok {
		err := conn.CloseWrite()
		if err != nil {
			return fmt.Errorf("close write side: %w", err)
		}

		return nil
	}

	err := endpoint.Close()
	if err != nil {
		return fmt.Errorf("close connection: %w", err)
	}

	return nil
}
