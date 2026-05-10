package usbip

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestListExportedDevicesConn(t *testing.T) {
	t.Parallel()

	conn := &recordingReadWriter{
		reader: bytes.NewReader(devlistReply(t, Device{
			Path:                "/sys/devices/pci0000:00/usb1/1-1",
			BusID:               "1-1",
			Busnum:              1,
			Devnum:              2,
			Speed:               2,
			IDVendor:            0x2fe3,
			IDProduct:           0x0001,
			BCDDevice:           0x0100,
			BDeviceClass:        0xef,
			BDeviceSubClass:     0x02,
			BDeviceProtocol:     0x01,
			BConfigurationValue: 0,
			BNumConfigurations:  0,
			Interfaces: []Interface{
				{Class: 0x02, SubClass: 0x02, Protocol: 0x00},
				{Class: 0x0a, SubClass: 0x00, Protocol: 0x00},
			},
		})),
		written: bytes.Buffer{},
	}

	devices, err := ListExportedDevicesConn(conn)
	if err != nil {
		t.Fatalf("list exported devices: %v", err)
	}

	wantRequest := []byte{0x01, 0x11, 0x80, 0x05, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(conn.written.Bytes(), wantRequest) {
		t.Fatalf("request bytes=% x, want % x", conn.written.Bytes(), wantRequest)
	}

	if len(devices) != 1 {
		t.Fatalf("device count=%d, want 1", len(devices))
	}

	device := devices[0]
	if device.BusID != "1-1" || device.Path != "/sys/devices/pci0000:00/usb1/1-1" {
		t.Fatalf("unexpected device identity: %+v", device)
	}

	if device.IDVendor != 0x2fe3 || device.IDProduct != 0x0001 {
		t.Fatalf("unexpected device ids: %04x:%04x", device.IDVendor, device.IDProduct)
	}

	if len(device.Interfaces) != 2 {
		t.Fatalf("interface count=%d, want 2", len(device.Interfaces))
	}
}

func TestListExportedDevicesConnReturnsStatusError(t *testing.T) {
	t.Parallel()

	conn := &recordingReadWriter{
		reader:  bytes.NewReader(devlistErrorReply(1)),
		written: bytes.Buffer{},
	}

	_, err := ListExportedDevicesConn(conn)
	if !errors.Is(err, errUSBIPStatus) {
		t.Fatalf("err=%v, want %v", err, errUSBIPStatus)
	}
}

type recordingReadWriter struct {
	reader  io.Reader
	written bytes.Buffer
}

func (rw *recordingReadWriter) Read(buffer []byte) (int, error) {
	return rw.reader.Read(buffer)
}

func (rw *recordingReadWriter) Write(buffer []byte) (int, error) {
	return rw.written.Write(buffer)
}

func devlistErrorReply(status uint32) []byte {
	var reply [12]byte

	binary.BigEndian.PutUint16(reply[0:2], protocolVersion)
	binary.BigEndian.PutUint16(reply[2:4], opRepDevlist)
	binary.BigEndian.PutUint32(reply[4:8], status)

	return reply[:]
}

func devlistReply(t *testing.T, devices ...Device) []byte {
	t.Helper()

	var buffer bytes.Buffer

	var header [12]byte

	binary.BigEndian.PutUint16(header[0:2], protocolVersion)
	binary.BigEndian.PutUint16(header[2:4], opRepDevlist)
	binary.BigEndian.PutUint32(header[4:8], 0)

	deviceCount, ok := checkedUint32(t, len(devices))
	if !ok {
		return nil
	}

	binary.BigEndian.PutUint32(header[8:12], deviceCount)
	buffer.Write(header[:])

	for _, device := range devices {
		writeDevice(t, &buffer, device)
	}

	return buffer.Bytes()
}

func writeDevice(t *testing.T, writer io.Writer, device Device) {
	t.Helper()

	var data [deviceInfoSize]byte

	copy(data[0:devicePathSize], device.Path)
	copy(data[devicePathSize:devicePathSize+deviceBusIDSize], device.BusID)
	binary.BigEndian.PutUint32(data[288:292], device.Busnum)
	binary.BigEndian.PutUint32(data[292:296], device.Devnum)
	binary.BigEndian.PutUint32(data[296:300], device.Speed)
	binary.BigEndian.PutUint16(data[300:302], device.IDVendor)
	binary.BigEndian.PutUint16(data[302:304], device.IDProduct)
	binary.BigEndian.PutUint16(data[304:306], device.BCDDevice)
	data[306] = device.BDeviceClass
	data[307] = device.BDeviceSubClass
	data[308] = device.BDeviceProtocol
	data[309] = device.BConfigurationValue
	data[310] = device.BNumConfigurations

	interfaceCount, ok := checkedUint8(t, len(device.Interfaces))
	if !ok {
		return
	}

	data[311] = interfaceCount

	_, err := writer.Write(data[:])
	if err != nil {
		t.Fatalf("write device: %v", err)
	}

	for _, iface := range device.Interfaces {
		_, err = writer.Write([]byte{iface.Class, iface.SubClass, iface.Protocol, 0})
		if err != nil {
			t.Fatalf("write interface: %v", err)
		}
	}
}

func checkedUint32(t *testing.T, value int) (uint32, bool) {
	t.Helper()

	if value < 0 || value > int(^uint32(0)) {
		t.Fatalf("value %d overflows uint32", value)

		return 0, false
	}

	return uint32(value), true
}

func checkedUint8(t *testing.T, value int) (uint8, bool) {
	t.Helper()

	if value < 0 || value > int(^uint8(0)) {
		t.Fatalf("value %d overflows uint8", value)

		return 0, false
	}

	return uint8(value), true
}
