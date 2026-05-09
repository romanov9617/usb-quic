// Package config contains application configuration structures.
package config

// Daemon contains normalized daemon runtime settings.
type Daemon struct {
	BindIPv4       bool   `json:"bind_ipv4"`
	BindIPv6       bool   `json:"bind_ipv6"`
	DeviceMode     bool   `json:"device_mode"`
	Daemonize      bool   `json:"daemonize"`
	Debug          bool   `json:"debug"`
	DevInsecureTLS bool   `json:"dev_insecure_tls"`
	PIDFile        string `json:"pid_file"`
	QUICAddr       string `json:"quic_addr"`
	QUICListen     string `json:"quic_listen"`
	TCPPort        int    `json:"tcp_port"`
	TransportMode  string `json:"transport_mode"`
	Upstream       string `json:"upstream"`
}
