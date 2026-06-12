package usbip

import (
	"os"
	"path/filepath"
	"testing"
)

//nolint:cyclop // The assertions intentionally cover filtering and field mapping in one sysfs fixture.
func TestListLocalDevices(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createLocalUSBDevice(t, root, "1-1", "00", true)
	createLocalUSBDevice(t, root, "1-0", "09", true)
	createLocalUSBDevice(t, root, "1-2", "00", false)

	devices, err := LocalDeviceLister{SysfsRoot: root}.ListLocalDevices(t.Context())
	if err != nil {
		t.Fatalf("list local devices: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("len(devices)=%d, want 1", len(devices))
	}

	device := devices[0]
	if device.BusID != "1-1" || device.IDVendor != 0x2fe3 || device.IDProduct != 0x0001 {
		t.Fatalf("device=%+v", device)
	}

	if device.BDeviceClass != 0 || device.BDeviceSubClass != 0 || device.BDeviceProtocol != 0 {
		t.Fatalf(
			"device class=%02x/%02x/%02x, want 00/00/00",
			device.BDeviceClass,
			device.BDeviceSubClass,
			device.BDeviceProtocol,
		)
	}

	if len(device.Interfaces) != 1 {
		t.Fatalf("len(interfaces)=%d, want 1", len(device.Interfaces))
	}

	if got := device.Interfaces[0]; got.Class != 0x08 || got.SubClass != 0x06 || got.Protocol != 0x50 {
		t.Fatalf("interface=%+v, want 08/06/50", got)
	}
}

func createLocalUSBDevice(t *testing.T, root, busid, class string, withDriver bool) {
	t.Helper()

	devicePath := filepath.Join(root, usbDevicesPath, busid)

	err := os.MkdirAll(devicePath, 0o700)
	if err != nil {
		t.Fatalf("create device path: %v", err)
	}

	writeAttr(t, devicePath, attrBDeviceClass, class)
	writeAttr(t, devicePath, attrBDeviceSubClass, "00")
	writeAttr(t, devicePath, attrBDeviceProtocol, "00")
	writeAttr(t, devicePath, attrBConfigurationValue, "01")
	writeAttr(t, devicePath, attrBNumConfigurations, "1")
	writeAttr(t, devicePath, attrBCDDevice, "0100")
	writeAttr(t, devicePath, attrBusnum, "1")
	writeAttr(t, devicePath, attrDevnum, "2")
	writeAttr(t, devicePath, attrIDProduct, "0001")
	writeAttr(t, devicePath, attrIDVendor, "2fe3")
	writeAttr(t, devicePath, attrPathSpeed, "480")

	interfacePath := filepath.Join(devicePath, busid+":1.0")

	err = os.MkdirAll(interfacePath, 0o700)
	if err != nil {
		t.Fatalf("create interface path: %v", err)
	}

	writeAttr(t, interfacePath, "bInterfaceClass", "08")
	writeAttr(t, interfacePath, "bInterfaceSubClass", "06")
	writeAttr(t, interfacePath, "bInterfaceProtocol", "50")

	if withDriver {
		writeAttr(t, devicePath, attrDriver, "")
	}
}

func writeAttr(t *testing.T, dir, name, value string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(value+"\n"), 0o600)
	if err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
