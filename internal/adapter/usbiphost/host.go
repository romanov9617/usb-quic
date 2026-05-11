// Package usbiphost contains Linux usbip-host sysfs integration.
package usbiphost

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultSysfsRoot = "/sys"

	usbDevicesPath   = "bus/usb/devices"
	usbDriversPath   = "bus/usb/drivers"
	driversProbePath = "bus/usb/drivers_probe"
	usbipHostDriver  = "usbip-host"

	bDeviceClassAttribute = "bDeviceClass"
	bindAttribute         = "bind"
	driverLink            = "driver"
	matchBusIDAttribute   = "match_busid"
	unbindAttribute       = "unbind"

	usbClassHub = 0x09
)

var (
	// ErrDeviceNotFound marks a missing USB device sysfs node.
	ErrDeviceNotFound = errors.New("usb device not found")

	// ErrDeviceIsHub marks attempts to export a USB hub.
	ErrDeviceIsHub = errors.New("usb hubs cannot be bound to usbip-host")

	// ErrDeviceNotBoundToUSBIPHost marks unbind calls for non-exported devices.
	ErrDeviceNotBoundToUSBIPHost = errors.New("usb device is not bound to usbip-host")
)

// Controller controls the Linux usbip-host driver through sysfs.
type Controller struct {
	SysfsRoot string
}

// BindDevice makes a local USB device exportable through usbip-host.
func (controller Controller) BindDevice(busid string) error {
	devicePath := controller.devicePath(busid)
	if !fileExists(devicePath) {
		return fmt.Errorf("%w: %s", ErrDeviceNotFound, busid)
	}

	if controller.isHub(devicePath) {
		return fmt.Errorf("%w: %s", ErrDeviceIsHub, busid)
	}

	hostPath := controller.driverPath(usbipHostDriver)

	err := writeSysfsAttribute(filepath.Join(hostPath, matchBusIDAttribute), "add "+busid)
	if err != nil {
		return fmt.Errorf("add usbip-host match_busid %s: %w", busid, err)
	}

	driver, err := currentDriver(devicePath)
	if err != nil {
		return fmt.Errorf("read current USB driver for %s: %w", busid, err)
	}

	if driver != "" && driver != usbipHostDriver {
		err = writeSysfsAttribute(filepath.Join(controller.driverPath(driver), unbindAttribute), busid)
		if err != nil {
			return fmt.Errorf("unbind %s from %s: %w", busid, driver, err)
		}
	}

	if driver == usbipHostDriver {
		return nil
	}

	err = writeSysfsAttribute(filepath.Join(hostPath, bindAttribute), busid)
	if err != nil {
		_ = writeSysfsAttribute(filepath.Join(hostPath, matchBusIDAttribute), "del "+busid)

		return fmt.Errorf("bind %s to usbip-host: %w", busid, err)
	}

	return nil
}

// UnbindDevice stops exporting a local USB device through usbip-host.
func (controller Controller) UnbindDevice(busid string) error {
	devicePath := controller.devicePath(busid)
	if !fileExists(devicePath) {
		return fmt.Errorf("%w: %s", ErrDeviceNotFound, busid)
	}

	driver, err := currentDriver(devicePath)
	if err != nil {
		return fmt.Errorf("read current USB driver for %s: %w", busid, err)
	}

	if driver != usbipHostDriver {
		return fmt.Errorf("%w: %s", ErrDeviceNotBoundToUSBIPHost, busid)
	}

	hostPath := controller.driverPath(usbipHostDriver)

	err = writeSysfsAttribute(filepath.Join(hostPath, unbindAttribute), busid)
	if err != nil {
		return fmt.Errorf("unbind %s from usbip-host: %w", busid, err)
	}

	err = writeSysfsAttribute(filepath.Join(hostPath, matchBusIDAttribute), "del "+busid)
	if err != nil {
		return fmt.Errorf("delete usbip-host match_busid %s: %w", busid, err)
	}

	err = writeSysfsAttribute(filepath.Join(controller.root(), driversProbePath), busid)
	if err != nil {
		return fmt.Errorf("reprobe USB device %s: %w", busid, err)
	}

	return nil
}

func (controller Controller) root() string {
	if controller.SysfsRoot != "" {
		return controller.SysfsRoot
	}

	return defaultSysfsRoot
}

func (controller Controller) devicePath(busid string) string {
	return filepath.Join(controller.root(), usbDevicesPath, busid)
}

func (controller Controller) driverPath(driver string) string {
	return filepath.Join(controller.root(), usbDriversPath, driver)
}

func (controller Controller) isHub(devicePath string) bool {
	//nolint:gosec // Path is built from a local USB sysfs device path and a fixed attribute name.
	data, err := os.ReadFile(filepath.Join(devicePath, bDeviceClassAttribute))
	if err != nil {
		return false
	}

	class, err := strconv.ParseUint(strings.TrimSpace(string(data)), 16, 8)
	if err != nil {
		return false
	}

	return class == usbClassHub
}

func currentDriver(devicePath string) (string, error) {
	target, err := os.Readlink(filepath.Join(devicePath, driverLink))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}

		return "", fmt.Errorf("read driver symlink: %w", err)
	}

	return filepath.Base(target), nil
}

func writeSysfsAttribute(path, value string) error {
	err := os.WriteFile(path, []byte(value), 0)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}
