package usbip

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	opReqImport = 0x8003
	opRepImport = 0x0003
)

var (
	errImportBusIDTooLong  = errors.New("usbip import busid is too long")
	errImportBusIDMismatch = errors.New("usbip import busid mismatch")
	errUSBIPImportStatus   = errors.New("usbip import status error")
)

// ImportDeviceConn requests import of busid over conn. On success, conn must
// remain open and be passed to vhci_hcd for kernel-side USB/IP traffic.
func ImportDeviceConn(conn io.ReadWriter, busid string) (Device, error) {
	err := writeImportRequest(conn, busid)
	if err != nil {
		return Device{}, err
	}

	device, err := readImportReply(conn)
	if err != nil {
		return Device{}, err
	}

	if device.BusID != busid {
		return Device{}, fmt.Errorf("%w: got %q want %q", errImportBusIDMismatch, device.BusID, busid)
	}

	return device, nil
}

func writeImportRequest(writer io.Writer, busid string) error {
	if len(busid) >= deviceBusIDSize {
		return fmt.Errorf("%w: len=%d max=%d", errImportBusIDTooLong, len(busid), deviceBusIDSize-1)
	}

	var request [commonHeaderSize + deviceBusIDSize]byte

	binary.BigEndian.PutUint16(request[0:2], protocolVersion)
	binary.BigEndian.PutUint16(request[2:4], opReqImport)
	binary.BigEndian.PutUint32(request[4:8], 0)
	copy(request[commonHeaderSize:], busid)

	_, err := writer.Write(request[:])
	if err != nil {
		return fmt.Errorf("write usbip import request: %w", err)
	}

	return nil
}

func readImportReply(reader io.Reader) (Device, error) {
	var header [commonHeaderSize]byte

	_, err := io.ReadFull(reader, header[:])
	if err != nil {
		return Device{}, fmt.Errorf("read usbip import reply header: %w", err)
	}

	version := binary.BigEndian.Uint16(header[0:2])
	reply := binary.BigEndian.Uint16(header[2:4])
	status := binary.BigEndian.Uint32(header[4:8])

	if version != protocolVersion || reply != opRepImport {
		return Device{}, fmt.Errorf("%w: version=0x%04x reply=0x%04x", errUnexpectedReply, version, reply)
	}

	if status != 0 {
		return Device{}, fmt.Errorf("%w: status=%d", errUSBIPImportStatus, status)
	}

	device, err := readDeviceInfo(reader)
	if err != nil {
		return Device{}, fmt.Errorf("read usbip import device: %w", err)
	}

	return device, nil
}
