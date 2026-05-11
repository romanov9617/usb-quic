package usbip

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

const (
	commonHeaderSize = 8

	protocolVersion = 0x0111
	opReqDevlist    = 0x8005
	opRepDevlist    = 0x0005

	// OP_REP_DEVLIST carries a fixed-width usbip_usb_device record followed by
	// one fixed-width usbip_usb_interface record per interface.
	devicePathSize       = 256
	deviceBusIDSize      = 32
	deviceBusnumSize     = 4
	deviceDevnumSize     = 4
	deviceSpeedSize      = 4
	deviceIDVendorSize   = 2
	deviceIDProductSize  = 2
	deviceBCDDeviceSize  = 2
	deviceClassSize      = 1
	deviceSubClassSize   = 1
	deviceProtocolSize   = 1
	deviceConfigSize     = 1
	deviceNumConfigsSize = 1
	deviceNumIfacesSize  = 1

	devicePathOffset       = 0
	deviceBusIDOffset      = devicePathOffset + devicePathSize
	deviceBusnumOffset     = deviceBusIDOffset + deviceBusIDSize
	deviceDevnumOffset     = deviceBusnumOffset + deviceBusnumSize
	deviceSpeedOffset      = deviceDevnumOffset + deviceDevnumSize
	deviceIDVendorOffset   = deviceSpeedOffset + deviceSpeedSize
	deviceIDProductOffset  = deviceIDVendorOffset + deviceIDVendorSize
	deviceBCDDeviceOffset  = deviceIDProductOffset + deviceIDProductSize
	deviceClassOffset      = deviceBCDDeviceOffset + deviceBCDDeviceSize
	deviceSubClassOffset   = deviceClassOffset + deviceClassSize
	deviceProtocolOffset   = deviceSubClassOffset + deviceSubClassSize
	deviceConfigOffset     = deviceProtocolOffset + deviceProtocolSize
	deviceNumConfigsOffset = deviceConfigOffset + deviceConfigSize
	deviceNumIfacesOffset  = deviceNumConfigsOffset + deviceNumConfigsSize
	deviceInfoSize         = deviceNumIfacesOffset + deviceNumIfacesSize

	interfaceSize = 4
)

var (
	errUnexpectedReply = errors.New("unexpected usbip devlist reply")
	errUSBIPStatus     = errors.New("usbip devlist status error")
)

// Device describes one exported USB/IP device.
type Device struct {
	Path                string
	BusID               string
	Busnum              uint32
	Devnum              uint32
	Speed               uint32
	IDVendor            uint16
	IDProduct           uint16
	BCDDevice           uint16
	BDeviceClass        uint8
	BDeviceSubClass     uint8
	BDeviceProtocol     uint8
	BConfigurationValue uint8
	BNumConfigurations  uint8
	Interfaces          []Interface
}

// Interface describes one USB interface belonging to an exported device.
type Interface struct {
	Class    uint8
	SubClass uint8
	Protocol uint8
}

// ListExportedDevices requests the exported USB device list from endpoint.
func ListExportedDevices(ctx context.Context, endpoint Endpoint) ([]Device, error) {
	address := endpoint.TCPAddress().Host

	var dialer net.Dialer

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial usbip %s: %w", address, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	return ListExportedDevicesConn(conn)
}

// ListExportedDevicesConn requests the exported USB device list over conn.
func ListExportedDevicesConn(conn io.ReadWriter) ([]Device, error) {
	err := writeDevlistRequest(conn)
	if err != nil {
		return nil, err
	}

	devices, err := readDevlistReply(conn)
	if err != nil {
		return nil, err
	}

	return devices, nil
}

func writeDevlistRequest(writer io.Writer) error {
	var request [8]byte

	binary.BigEndian.PutUint16(request[0:2], protocolVersion)
	binary.BigEndian.PutUint16(request[2:4], opReqDevlist)
	binary.BigEndian.PutUint32(request[4:8], 0)

	_, err := writer.Write(request[:])
	if err != nil {
		return fmt.Errorf("write usbip devlist request: %w", err)
	}

	return nil
}

func readDevlistReply(reader io.Reader) ([]Device, error) {
	var header [12]byte

	_, err := io.ReadFull(reader, header[:])
	if err != nil {
		return nil, fmt.Errorf("read usbip devlist reply header: %w", err)
	}

	version := binary.BigEndian.Uint16(header[0:2])
	reply := binary.BigEndian.Uint16(header[2:4])
	status := binary.BigEndian.Uint32(header[4:8])
	count := binary.BigEndian.Uint32(header[8:12])

	if version != protocolVersion || reply != opRepDevlist {
		return nil, fmt.Errorf("%w: version=0x%04x reply=0x%04x", errUnexpectedReply, version, reply)
	}

	if status != 0 {
		return nil, fmt.Errorf("%w: status=%d", errUSBIPStatus, status)
	}

	devices := make([]Device, 0, count)
	for range count {
		device, err := readDevice(reader)
		if err != nil {
			return nil, err
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func readDevice(reader io.Reader) (Device, error) {
	device, err := readDeviceInfo(reader)
	if err != nil {
		return Device{}, err
	}

	interfaceCount := cap(device.Interfaces)
	device.Interfaces = make([]Interface, 0, interfaceCount)

	for range interfaceCount {
		iface, err := readInterface(reader)
		if err != nil {
			return Device{}, err
		}

		device.Interfaces = append(device.Interfaces, iface)
	}

	return device, nil
}

func readDeviceInfo(reader io.Reader) (Device, error) {
	var data [deviceInfoSize]byte

	_, err := io.ReadFull(reader, data[:])
	if err != nil {
		return Device{}, fmt.Errorf("read usbip device: %w", err)
	}

	device := Device{
		Path:                cString(data[devicePathOffset:deviceBusIDOffset]),
		BusID:               cString(data[deviceBusIDOffset:deviceBusnumOffset]),
		Busnum:              binary.BigEndian.Uint32(data[deviceBusnumOffset:deviceDevnumOffset]),
		Devnum:              binary.BigEndian.Uint32(data[deviceDevnumOffset:deviceSpeedOffset]),
		Speed:               binary.BigEndian.Uint32(data[deviceSpeedOffset:deviceIDVendorOffset]),
		IDVendor:            binary.BigEndian.Uint16(data[deviceIDVendorOffset:deviceIDProductOffset]),
		IDProduct:           binary.BigEndian.Uint16(data[deviceIDProductOffset:deviceBCDDeviceOffset]),
		BCDDevice:           binary.BigEndian.Uint16(data[deviceBCDDeviceOffset:deviceClassOffset]),
		BDeviceClass:        data[deviceClassOffset],
		BDeviceSubClass:     data[deviceSubClassOffset],
		BDeviceProtocol:     data[deviceProtocolOffset],
		BConfigurationValue: data[deviceConfigOffset],
		BNumConfigurations:  data[deviceNumConfigsOffset],
		Interfaces:          nil,
	}

	interfaceCount := int(data[deviceNumIfacesOffset])
	device.Interfaces = make([]Interface, 0, interfaceCount)

	return device, nil
}

func readInterface(reader io.Reader) (Interface, error) {
	var data [interfaceSize]byte

	_, err := io.ReadFull(reader, data[:])
	if err != nil {
		return Interface{}, fmt.Errorf("read usbip interface: %w", err)
	}

	return Interface{
		Class:    data[0],
		SubClass: data[1],
		Protocol: data[2],
	}, nil
}

func cString(data []byte) string {
	index := len(data)
	for i, value := range data {
		if value == 0 {
			index = i

			break
		}
	}

	return strings.TrimRight(string(data[:index]), "\x00")
}
