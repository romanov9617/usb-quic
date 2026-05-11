package usbiphost

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const testBusID = "1-1"

func TestBindDeviceRebindsDeviceToUSBIPHost(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createDevice(t, root, "00", "usbhid")
	createDriver(t, root, "usbhid")
	hostPath := createDriver(t, root, usbipHostDriver)

	err := Controller{SysfsRoot: root}.BindDevice(testBusID)
	if err != nil {
		t.Fatalf("bind device: %v", err)
	}

	assertFile(t, filepath.Join(hostPath, matchBusIDAttribute), "add "+testBusID)
	assertFile(t, filepath.Join(driverPath(root, "usbhid"), unbindAttribute), testBusID)
	assertFile(t, filepath.Join(hostPath, bindAttribute), testBusID)
}

func TestBindDeviceRejectsHub(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createDevice(t, root, "09", "hub")
	createDriver(t, root, usbipHostDriver)

	err := Controller{SysfsRoot: root}.BindDevice(testBusID)
	if !errors.Is(err, ErrDeviceIsHub) {
		t.Fatalf("err=%v, want %v", err, ErrDeviceIsHub)
	}
}

func TestBindDeviceReturnsDeviceNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createDriver(t, root, usbipHostDriver)

	err := Controller{SysfsRoot: root}.BindDevice(testBusID)
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("err=%v, want %v", err, ErrDeviceNotFound)
	}
}

func TestUnbindDeviceRemovesUSBIPHostBinding(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createDevice(t, root, "00", usbipHostDriver)
	hostPath := createDriver(t, root, usbipHostDriver)
	probePath := createDriversProbe(t, root)

	err := Controller{SysfsRoot: root}.UnbindDevice(testBusID)
	if err != nil {
		t.Fatalf("unbind device: %v", err)
	}

	assertFile(t, filepath.Join(hostPath, unbindAttribute), testBusID)
	assertFile(t, filepath.Join(hostPath, matchBusIDAttribute), "del "+testBusID)
	assertFile(t, probePath, testBusID)
}

func TestUnbindDeviceRequiresUSBIPHostDriver(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createDevice(t, root, "00", "usbhid")
	createDriver(t, root, usbipHostDriver)

	err := Controller{SysfsRoot: root}.UnbindDevice(testBusID)
	if !errors.Is(err, ErrDeviceNotBoundToUSBIPHost) {
		t.Fatalf("err=%v, want %v", err, ErrDeviceNotBoundToUSBIPHost)
	}
}

func createDevice(t *testing.T, root, class, driver string) {
	t.Helper()

	devicePath := filepath.Join(root, usbDevicesPath, testBusID)

	err := os.MkdirAll(devicePath, 0o700)
	if err != nil {
		t.Fatalf("create device path: %v", err)
	}

	err = os.WriteFile(filepath.Join(devicePath, bDeviceClassAttribute), []byte(class+"\n"), 0o600)
	if err != nil {
		t.Fatalf("write device class: %v", err)
	}

	if driver == "" {
		return
	}

	createDriver(t, root, driver)

	err = os.Symlink(driverPath(root, driver), filepath.Join(devicePath, driverLink))
	if err != nil {
		t.Fatalf("create driver symlink: %v", err)
	}
}

func createDriver(t *testing.T, root, driver string) string {
	t.Helper()

	path := driverPath(root, driver)

	err := os.MkdirAll(path, 0o700)
	if err != nil {
		t.Fatalf("create driver path: %v", err)
	}

	for _, attribute := range []string{bindAttribute, unbindAttribute, matchBusIDAttribute} {
		err = os.WriteFile(filepath.Join(path, attribute), nil, 0o600)
		if err != nil {
			t.Fatalf("write %s: %v", attribute, err)
		}
	}

	return path
}

func createDriversProbe(t *testing.T, root string) string {
	t.Helper()

	path := filepath.Join(root, driversProbePath)

	err := os.MkdirAll(filepath.Dir(path), 0o700)
	if err != nil {
		t.Fatalf("create drivers probe dir: %v", err)
	}

	err = os.WriteFile(path, nil, 0o600)
	if err != nil {
		t.Fatalf("write drivers probe: %v", err)
	}

	return path
}

func driverPath(root, driver string) string {
	return filepath.Join(root, usbDriversPath, driver)
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path) //nolint:gosec // Test reads a file created under t.TempDir.
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	if string(got) != want {
		t.Fatalf("%s=%q, want %q", path, string(got), want)
	}
}
