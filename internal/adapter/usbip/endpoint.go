// Package usbip contains USB/IP domain entities.
package usbip

import (
	"net"
	"net/url"
	"strconv"
)

// DefaultPort is the IANA-assigned USB/IP TCP port.
const DefaultPort = 3240

// Endpoint identifies a USB/IP endpoint.
type Endpoint struct {
	Host string
	Port int
}

// Address returns the URL for endpoint.
func (endpoint Endpoint) Address() url.URL {
	return endpoint.url("usbip")
}

// TCPAddress returns the TCP URL for endpoint.
func (endpoint Endpoint) TCPAddress() url.URL {
	return endpoint.url("tcp")
}

func (endpoint Endpoint) url(scheme string) url.URL {
	//nolint:exhaustruct // Endpoint URLs only need scheme and network authority.
	return url.URL{
		Scheme: scheme,
		Host: net.JoinHostPort(
			endpoint.Host,
			strconv.Itoa(endpoint.Port),
		),
	}
}
