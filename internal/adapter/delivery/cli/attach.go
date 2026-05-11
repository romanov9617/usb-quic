package cli

import (
	"context"
	"fmt"
	"net"

	adapterusbip "usb-quic/internal/adapter/usbip"
	"usb-quic/internal/adapter/vhci"
)

func attachRemote(ctx context.Context, endpoint adapterusbip.Endpoint, busid string) (int, error) {
	address := endpoint.TCPAddress().Host

	var dialer net.Dialer

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return 0, fmt.Errorf("dial usbip %s: %w", address, err)
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		_ = conn.Close()

		return 0, fmt.Errorf("%w: dial usbip %s: got %T", ErrUnexpectedConnectionType, address, conn)
	}

	device, err := adapterusbip.ImportDeviceConn(tcpConn, busid)
	if err != nil {
		_ = tcpConn.Close()

		return 0, fmt.Errorf("import usbip device %s from %s: %w", busid, address, err)
	}

	file, err := tcpConn.File()
	if err != nil {
		_ = tcpConn.Close()

		return 0, fmt.Errorf("duplicate usbip socket fd: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	port, err := vhci.Controller{SysfsRoot: ""}.AttachDevice(file.Fd(), device.Busnum, device.Devnum, device.Speed)
	if err != nil {
		_ = tcpConn.Close()

		return 0, fmt.Errorf("attach usbip device %s to vhci_hcd: %w", busid, err)
	}

	_ = tcpConn.Close()

	return port, nil
}

func detachRemote(_ context.Context, port int) error {
	err := vhci.Controller{SysfsRoot: ""}.DetachDevice(port)
	if err != nil {
		return fmt.Errorf("detach vhci_hcd port %d: %w", port, err)
	}

	return nil
}
