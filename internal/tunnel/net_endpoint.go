package tunnel

import (
	"fmt"
	"net"
)

// NetEndpoint adapts a net.Conn to Endpoint.
type NetEndpoint struct {
	net.Conn
}

// NewNetEndpoint adapts conn to NetEndpoint.
func NewNetEndpoint(conn net.Conn) *NetEndpoint {
	return &NetEndpoint{
		Conn: conn,
	}
}

// CloseRead closes the read side when conn supports half-close, or closes conn.
func (endpoint *NetEndpoint) CloseRead() error {
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

// CloseWrite closes the write side when conn supports half-close, or closes conn.
func (endpoint *NetEndpoint) CloseWrite() error {
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
