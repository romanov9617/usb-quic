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

	platformDevicesPath = "devices/platform"
	vhciDevicePrefix    = "vhci_hcd"

	statusAttribute = "status"
	attachAttribute = "attach"
	detachAttribute = "detach"

	vdevStatusNull = 4
	usbSpeedSuper  = 5

	minStatusFields = 5
	devidBusShift   = 16
)

var (
	errNoVHCIController = errors.New("vhci_hcd controller not found")
	errNoFreePort       = errors.New("no free vhci_hcd port")
)

// Controller controls the Linux vhci_hcd driver through sysfs.
type Controller struct {
	SysfsRoot string
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

type controller struct {
	path string
}

type portStatus struct {
	hub    string
	port   int
	status int
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

		port, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		statusValue, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		ports = append(ports, portStatus{
			hub:    fields[0],
			port:   port,
			status: statusValue,
		})
	}

	return ports
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
