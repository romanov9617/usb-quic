// Package vhci contains Linux vhci_hcd sysfs integration.
package vhci

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
	defaultRunRoot   = "/var/run"
	recordDirMode    = 0o750
	recordFileMode   = 0o600

	platformDevicesPath = "devices/platform"
	usbDevicesPath      = "bus/usb/devices"
	vhciDevicePrefix    = "vhci_hcd"
	vhciRunDir          = "vhci_hcd"

	statusAttribute = "status"
	attachAttribute = "attach"
	detachAttribute = "detach"

	vdevStatusNull        = 4
	vdevStatusNotAssigned = 5
	usbSpeedSuper         = 5

	minStatusFields        = 5
	connectionRecordFields = 3
	statusHubField         = 0
	statusPortField        = 1
	statusStateField       = 2
	statusSpeedField       = 3
	statusDeviceField      = 4
	statusLocalBusIDField  = 6
	devidBusShift          = 16
	devidMask              = 0xffff
)

var (
	errNoVHCIController = errors.New("vhci_hcd controller not found")
	errNoFreePort       = errors.New("no free vhci_hcd port")
)

// Controller controls the Linux vhci_hcd driver through sysfs.
type Controller struct {
	SysfsRoot string
	RunRoot   string
}

// ImportedDevice describes a USB/IP device imported into local vhci_hcd.
type ImportedDevice struct {
	Hub         string
	Port        int
	Status      int
	Speed       int
	LocalBusID  string
	RemoteHost  string
	RemotePort  string
	RemoteBusID string
	RemoteBus   int
	RemoteDev   int
	IDVendor    uint16
	IDProduct   uint16
}

// AttachDevice passes sockfd to vhci_hcd and returns the selected local port.
func (vhci Controller) AttachDevice(sockfd uintptr, busnum, devnum, speed uint32) (int, error) {
	controller, err := vhci.findController()
	if err != nil {
		return 0, err
	}

	port, err := controller.freePort(speed)
	if err != nil {
		return 0, err
	}

	devid := busnum<<devidBusShift | devnum
	value := fmt.Sprintf("%d %d %d %d", port, sockfd, devid, speed)

	err = os.WriteFile(filepath.Join(controller.path, attachAttribute), []byte(value), 0)
	if err != nil {
		return 0, fmt.Errorf("write vhci_hcd attach: %w", err)
	}

	return port, nil
}

// DetachDevice detaches the imported USB/IP device from a local vhci_hcd port.
func (vhci Controller) DetachDevice(port int) error {
	controller, err := vhci.findController()
	if err != nil {
		return err
	}

	value := strconv.Itoa(port)

	err = os.WriteFile(filepath.Join(controller.path, detachAttribute), []byte(value), 0)
	if err != nil {
		return fmt.Errorf("write vhci_hcd detach: %w", err)
	}

	return nil
}

// RecordConnection stores the remote endpoint metadata used by usbip port.
func (vhci Controller) RecordConnection(port int, host, service, busid string) error {
	runDir := filepath.Join(vhci.runRoot(), vhciRunDir)

	err := os.MkdirAll(runDir, recordDirMode)
	if err != nil {
		return fmt.Errorf("create vhci_hcd run directory: %w", err)
	}

	value := fmt.Sprintf("%s %s %s", host, service, busid)

	err = os.WriteFile(filepath.Join(runDir, fmt.Sprintf("port%d", port)), []byte(value), recordFileMode)
	if err != nil {
		return fmt.Errorf("write vhci_hcd port record: %w", err)
	}

	return nil
}

// ListImportedDevices returns USB/IP devices imported into local vhci_hcd.
func (vhci Controller) ListImportedDevices() ([]ImportedDevice, error) {
	controller, err := vhci.findController()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(controller.path, statusAttribute))
	if err != nil {
		return nil, fmt.Errorf("read vhci_hcd status: %w", err)
	}

	statuses := parseStatus(string(data))
	devices := make([]ImportedDevice, 0, len(statuses))

	for _, status := range statuses {
		if status.status == vdevStatusNull || status.status == vdevStatusNotAssigned {
			continue
		}

		device := ImportedDevice{
			Hub:         status.hub,
			Port:        status.port,
			Status:      status.status,
			Speed:       status.speed,
			LocalBusID:  status.localBusID,
			RemoteHost:  "",
			RemotePort:  "",
			RemoteBusID: "",
			RemoteBus:   remoteBus(status.devid),
			RemoteDev:   remoteDev(status.devid),
			IDVendor:    0,
			IDProduct:   0,
		}

		device.RemoteHost, device.RemotePort, device.RemoteBusID = vhci.readConnectionRecord(status.port)
		device.IDVendor, device.IDProduct = vhci.readUSBIDs(status.localBusID)

		devices = append(devices, device)
	}

	return devices, nil
}

type controller struct {
	path string
}

type portStatus struct {
	hub        string
	port       int
	status     int
	speed      int
	devid      uint32
	localBusID string
}

func (vhci Controller) findController() (controller, error) {
	root := vhci.SysfsRoot
	if root == "" {
		root = defaultSysfsRoot
	}

	platformPath := filepath.Join(root, platformDevicesPath)

	entries, err := os.ReadDir(platformPath)
	if err != nil {
		return controller{}, fmt.Errorf("read vhci_hcd platform devices: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), vhciDevicePrefix) {
			continue
		}

		path := filepath.Join(platformPath, entry.Name())
		if fileExists(filepath.Join(path, statusAttribute)) {
			return controller{path: path}, nil
		}
	}

	return controller{}, errNoVHCIController
}

func (vhci Controller) runRoot() string {
	if vhci.RunRoot != "" {
		return vhci.RunRoot
	}

	return defaultRunRoot
}

func (controller controller) freePort(speed uint32) (int, error) {
	data, err := os.ReadFile(filepath.Join(controller.path, statusAttribute))
	if err != nil {
		return 0, fmt.Errorf("read vhci_hcd status: %w", err)
	}

	ports := parseStatus(string(data))
	if len(ports) == 0 {
		return 0, errNoFreePort
	}

	hub := preferredHub(speed)
	for _, port := range ports {
		if port.status == vdevStatusNull && port.hub == hub {
			return port.port, nil
		}
	}

	for _, port := range ports {
		if port.status == vdevStatusNull {
			return port.port, nil
		}
	}

	return 0, errNoFreePort
}

func parseStatus(status string) []portStatus {
	lines := strings.Split(status, "\n")
	ports := make([]portStatus, 0, len(lines))

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < minStatusFields {
			continue
		}

		port, err := strconv.Atoi(fields[statusPortField])
		if err != nil {
			continue
		}

		statusValue, err := strconv.Atoi(fields[statusStateField])
		if err != nil {
			continue
		}

		speed, err := strconv.Atoi(fields[statusSpeedField])
		if err != nil {
			continue
		}

		devid, err := strconv.ParseUint(fields[statusDeviceField], 16, 32)
		if err != nil {
			continue
		}

		localBusID := ""
		if len(fields) > statusLocalBusIDField {
			localBusID = fields[statusLocalBusIDField]
		}

		ports = append(ports, portStatus{
			hub:        fields[statusHubField],
			port:       port,
			status:     statusValue,
			speed:      speed,
			devid:      uint32(devid),
			localBusID: localBusID,
		})
	}

	return ports
}

func (vhci Controller) readConnectionRecord(port int) (string, string, string) {
	data, err := os.ReadFile(filepath.Join(vhci.runRoot(), vhciRunDir, fmt.Sprintf("port%d", port)))
	if err != nil {
		return "", "", ""
	}

	fields := strings.Fields(string(data))
	if len(fields) < connectionRecordFields {
		return "", "", ""
	}

	return fields[0], fields[1], fields[2]
}

func (vhci Controller) readUSBIDs(busid string) (uint16, uint16) {
	if busid == "" || busid == "0-0" {
		return 0, 0
	}

	root := vhci.SysfsRoot
	if root == "" {
		root = defaultSysfsRoot
	}

	devicePath := filepath.Join(root, usbDevicesPath, busid)

	return readHexUint16(filepath.Join(devicePath, "idVendor")), readHexUint16(filepath.Join(devicePath, "idProduct"))
}

func readHexUint16(path string) uint16 {
	//nolint:gosec // Path is built from the local vhci_hcd status busid and fixed sysfs attribute names.
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 16, 16)
	if err != nil {
		return 0
	}

	return uint16(value)
}

func remoteBus(devid uint32) int {
	return int((devid >> devidBusShift) & devidMask)
}

func remoteDev(devid uint32) int {
	return int(devid & devidMask)
}

func preferredHub(speed uint32) string {
	if speed >= usbSpeedSuper {
		return "ss"
	}

	return "hs"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}
