package usbip

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func TestImportDeviceConn(t *testing.T) {
	t.Parallel()

	conn := &recordingReadWriter{
		reader: bytes.NewReader(importReply(t, Device{
			Path:                testDevicePath,
			BusID:               testBusID,
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
			Interfaces:          nil,
		})),
		written: bytes.Buffer{},
	}

	device, err := ImportDeviceConn(conn, testBusID)
	if err != nil {
		t.Fatalf("import device: %v", err)
	}

	wantRequest := append([]byte{0x01, 0x11, 0x80, 0x03, 0x00, 0x00, 0x00, 0x00}, make([]byte, deviceBusIDSize)...)
	copy(wantRequest[commonHeaderSize:], testBusID)

	if !bytes.Equal(conn.written.Bytes(), wantRequest) {
		t.Fatalf("request bytes=% x, want % x", conn.written.Bytes(), wantRequest)
	}

	if device.BusID != testBusID || device.Busnum != 1 || device.Devnum != 2 || device.Speed != 2 {
		t.Fatalf("unexpected device: %+v", device)
	}
}

func TestImportDeviceConnReturnsStatusError(t *testing.T) {
	t.Parallel()

	conn := &recordingReadWriter{
		reader:  bytes.NewReader(importErrorReply(2)),
		written: bytes.Buffer{},
	}

	_, err := ImportDeviceConn(conn, testBusID)
	if !errors.Is(err, errUSBIPImportStatus) {
		t.Fatalf("err=%v, want %v", err, errUSBIPImportStatus)
	}
}

func TestImportDeviceConnRejectsDifferentBusID(t *testing.T) {
	t.Parallel()

	conn := &recordingReadWriter{
		reader: bytes.NewReader(importReply(t, Device{
			Path:                "/sys/devices/pci0000:00/usb1/1-2",
			BusID:               "1-2",
			Busnum:              1,
			Devnum:              3,
			Speed:               2,
			IDVendor:            0x2fe3,
			IDProduct:           0x0001,
			BCDDevice:           0x0100,
			BDeviceClass:        0xef,
			BDeviceSubClass:     0x02,
			BDeviceProtocol:     0x01,
			BConfigurationValue: 0,
			BNumConfigurations:  0,
			Interfaces:          nil,
		})),
		written: bytes.Buffer{},
	}

	_, err := ImportDeviceConn(conn, testBusID)
	if !errors.Is(err, errImportBusIDMismatch) {
		t.Fatalf("err=%v, want %v", err, errImportBusIDMismatch)
	}
}

func importErrorReply(status uint32) []byte {
	var reply [commonHeaderSize]byte

	binary.BigEndian.PutUint16(reply[0:2], protocolVersion)
	binary.BigEndian.PutUint16(reply[2:4], opRepImport)
	binary.BigEndian.PutUint32(reply[4:8], status)

	return reply[:]
}

func importReply(t *testing.T, device Device) []byte {
	t.Helper()

	var buffer bytes.Buffer

	var header [commonHeaderSize]byte

	binary.BigEndian.PutUint16(header[0:2], protocolVersion)
	binary.BigEndian.PutUint16(header[2:4], opRepImport)
	binary.BigEndian.PutUint32(header[4:8], 0)
	buffer.Write(header[:])
	writeDevice(t, &buffer, device)

	return buffer.Bytes()[:commonHeaderSize+deviceInfoSize]
}
