package usbip

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultSysfsRoot = "/sys"
	usbDevicesPath   = "bus/usb/devices"

	attrBConfigurationValue = "bConfigurationValue"
	attrBDeviceClass        = "bDeviceClass"
	attrBDeviceProtocol     = "bDeviceProtocol"
	attrBDeviceSubClass     = "bDeviceSubClass"
	attrBNumConfigurations  = "bNumConfigurations"
	attrBCDDevice           = "bcdDevice"
	attrBusnum              = "busnum"
	attrDevnum              = "devnum"
	attrDriver              = "driver"
	attrIDProduct           = "idProduct"
	attrIDVendor            = "idVendor"
	attrPathSpeed           = "speed"

	usbHubClass = 0x09

	usbIPSpeedUnknown = 0
	usbIPSpeedLow     = 1
	usbIPSpeedFull    = 2
	usbIPSpeedHigh    = 3
	usbIPSpeedSuper   = 5
)

// LocalDeviceLister lists local USB devices from sysfs.
type LocalDeviceLister struct {
	SysfsRoot string
}

// ListLocalDevices returns local USB devices that have an active driver.
func (lister LocalDeviceLister) ListLocalDevices(_ context.Context) ([]Device, error) {
	root := lister.root()
	devicesPath := filepath.Join(root, usbDevicesPath)

	entries, err := os.ReadDir(devicesPath)
	if err != nil {
		return nil, fmt.Errorf("read local USB devices: %w", err)
	}

	devices := make([]Device, 0, len(entries))

	for _, entry := range entries {
		device, ok, err := readLocalDevice(root, devicesPath, entry.Name())
		if err != nil {
			return nil, err
		}

		if ok {
			devices = append(devices, device)
		}
	}

	sort.Slice(devices, func(left, right int) bool {
		return devices[left].BusID < devices[right].BusID
	})

	return devices, nil
}

func (lister LocalDeviceLister) root() string {
	if lister.SysfsRoot != "" {
		return lister.SysfsRoot
	}

	return defaultSysfsRoot
}

func readLocalDevice(root, devicesPath, name string) (Device, bool, error) {
	if !isUSBDeviceBusID(name) {
		return emptyDevice(), false, nil
	}

	devicePath := filepath.Join(devicesPath, name)
	if !fileExists(filepath.Join(devicePath, attrDriver)) {
		return emptyDevice(), false, nil
	}

	class, err := readHexUint8(filepath.Join(devicePath, attrBDeviceClass))
	if err != nil {
		return emptyDevice(), false, fmt.Errorf("read bDeviceClass for %s: %w", name, err)
	}

	if class == usbHubClass {
		return emptyDevice(), false, nil
	}

	device := Device{
		Path:                filepath.Join(root, usbDevicesPath, name),
		BusID:               name,
		Busnum:              readDecimalUint32(filepath.Join(devicePath, attrBusnum)),
		Devnum:              readDecimalUint32(filepath.Join(devicePath, attrDevnum)),
		Speed:               usbIPSpeed(readDecimalString(filepath.Join(devicePath, attrPathSpeed))),
		IDVendor:            readHexUint16Default(filepath.Join(devicePath, attrIDVendor)),
		IDProduct:           readHexUint16Default(filepath.Join(devicePath, attrIDProduct)),
		BCDDevice:           readHexUint16Default(filepath.Join(devicePath, attrBCDDevice)),
		BDeviceClass:        class,
		BDeviceSubClass:     readHexUint8Default(filepath.Join(devicePath, attrBDeviceSubClass)),
		BDeviceProtocol:     readHexUint8Default(filepath.Join(devicePath, attrBDeviceProtocol)),
		BConfigurationValue: readHexUint8Default(filepath.Join(devicePath, attrBConfigurationValue)),
		BNumConfigurations:  readDecimalUint8(filepath.Join(devicePath, attrBNumConfigurations)),
		Interfaces:          readLocalInterfaces(devicePath, name),
	}

	return device, true, nil
}

func emptyDevice() Device {
	return Device{
		Path:                "",
		BusID:               "",
		Busnum:              0,
		Devnum:              0,
		Speed:               0,
		IDVendor:            0,
		IDProduct:           0,
		BCDDevice:           0,
		BDeviceClass:        0,
		BDeviceSubClass:     0,
		BDeviceProtocol:     0,
		BConfigurationValue: 0,
		BNumConfigurations:  0,
		Interfaces:          nil,
	}
}

func isUSBDeviceBusID(name string) bool {
	return strings.Contains(name, "-") && !strings.Contains(name, ":")
}

func readLocalInterfaces(devicePath, busid string) []Interface {
	entries, err := os.ReadDir(devicePath)
	if err != nil {
		return nil
	}

	prefix := busid + ":"
	interfaces := make([]Interface, 0, len(entries))

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		interfacePath := filepath.Join(devicePath, entry.Name())
		interfaces = append(interfaces, Interface{
			Class:    readHexUint8Default(filepath.Join(interfacePath, "bInterfaceClass")),
			SubClass: readHexUint8Default(filepath.Join(interfacePath, "bInterfaceSubClass")),
			Protocol: readHexUint8Default(filepath.Join(interfacePath, "bInterfaceProtocol")),
		})
	}

	sort.Slice(interfaces, func(left, right int) bool {
		return interfaces[left].Class < interfaces[right].Class
	})

	return interfaces
}

func readDecimalString(path string) string {
	data, err := os.ReadFile(path) //nolint:gosec // Path is built from local sysfs root and fixed attributes.
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func readDecimalUint32(path string) uint32 {
	value, err := strconv.ParseUint(readDecimalString(path), 10, 32)
	if err != nil {
		return 0
	}

	return uint32(value)
}

func readDecimalUint8(path string) uint8 {
	value, err := strconv.ParseUint(readDecimalString(path), 10, 8)
	if err != nil {
		return 0
	}

	return uint8(value)
}

func readHexUint16Default(path string) uint16 {
	value, err := strconv.ParseUint(readDecimalString(path), 16, 16)
	if err != nil {
		return 0
	}

	return uint16(value)
}

func readHexUint8(path string) (uint8, error) {
	value, err := strconv.ParseUint(readDecimalString(path), 16, 8)
	if err != nil {
		return 0, fmt.Errorf("parse hex uint8: %w", err)
	}

	return uint8(value), nil
}

func readHexUint8Default(path string) uint8 {
	value, err := readHexUint8(path)
	if err != nil {
		return 0
	}

	return value
}

func usbIPSpeed(speed string) uint32 {
	switch speed {
	case "1.5":
		return usbIPSpeedLow
	case "12":
		return usbIPSpeedFull
	case "480":
		return usbIPSpeedHigh
	case "5000":
		return usbIPSpeedSuper
	default:
		return usbIPSpeedUnknown
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}
