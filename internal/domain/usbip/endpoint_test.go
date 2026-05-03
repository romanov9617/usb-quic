package usbip

import "testing"

func TestEndpointAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		want string
	}{
		{
			name: "ipv4",
			host: "192.0.2.10",
			want: "usbip://192.0.2.10:3240",
		},
		{
			name: "ipv6",
			host: "2001:db8::1",
			want: "usbip://[2001:db8::1]:3240",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			endpoint := Endpoint{
				Host: tt.host,
				Port: DefaultPort,
			}

			address := endpoint.Address()

			if got := address.String(); got != tt.want {
				t.Fatalf("unexpected address: want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestEndpointTCPAddress(t *testing.T) {
	t.Parallel()

	endpoint := Endpoint{
		Host: "127.0.0.1",
		Port: DefaultPort,
	}

	address := endpoint.TCPAddress()

	if got := address.String(); got != "tcp://127.0.0.1:3240" {
		t.Fatalf("unexpected address: %q", got)
	}
}
