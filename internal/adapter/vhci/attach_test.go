package vhci

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAttachDeviceWritesVHCIAttribute(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	controllerPath := filepath.Join(root, platformDevicesPath, "vhci_hcd.0")

	err := os.MkdirAll(controllerPath, 0o700)
	if err != nil {
		t.Fatalf("create controller path: %v", err)
	}

	status := `hub port sta spd dev sockfd local_busid
hs 000 004 000 00000000 000 0-0
hs 001 006 002 00010002 123 1-1
ss 008 004 005 00000000 000 0-0
`

	err = os.WriteFile(filepath.Join(controllerPath, statusAttribute), []byte(status), 0o600)
	if err != nil {
		t.Fatalf("write status: %v", err)
	}

	err = os.WriteFile(filepath.Join(controllerPath, attachAttribute), nil, 0o600)
	if err != nil {
		t.Fatalf("write attach: %v", err)
	}

	port, err := Controller{SysfsRoot: root, RunRoot: ""}.AttachDevice(42, 1, 2, 2)
	if err != nil {
		t.Fatalf("attach device: %v", err)
	}

	if port != 0 {
		t.Fatalf("port=%d, want 0", port)
	}

	attachPath := filepath.Join(controllerPath, attachAttribute)

	got, err := os.ReadFile(attachPath) //nolint:gosec // Test reads a file created under t.TempDir.
	if err != nil {
		t.Fatalf("read attach: %v", err)
	}

	want := "0 42 65538 2"
	if string(got) != want {
		t.Fatalf("attach payload=%q, want %q", string(got), want)
	}
}

func TestAttachDevicePrefersSuperSpeedHub(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	controllerPath := filepath.Join(root, platformDevicesPath, "vhci_hcd.0")

	err := os.MkdirAll(controllerPath, 0o700)
	if err != nil {
		t.Fatalf("create controller path: %v", err)
	}

	status := `hub port sta spd dev sockfd local_busid
hs 000 004 000 00000000 000 0-0
ss 008 004 005 00000000 000 0-0
`

	err = os.WriteFile(filepath.Join(controllerPath, statusAttribute), []byte(status), 0o600)
	if err != nil {
		t.Fatalf("write status: %v", err)
	}

	err = os.WriteFile(filepath.Join(controllerPath, attachAttribute), nil, 0o600)
	if err != nil {
		t.Fatalf("write attach: %v", err)
	}

	port, err := Controller{SysfsRoot: root, RunRoot: ""}.AttachDevice(42, 1, 2, 5)
	if err != nil {
		t.Fatalf("attach device: %v", err)
	}

	if port != 8 {
		t.Fatalf("port=%d, want 8", port)
	}
}

func TestAttachDeviceReturnsNoController(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	err := os.MkdirAll(filepath.Join(root, platformDevicesPath), 0o700)
	if err != nil {
		t.Fatalf("create platform path: %v", err)
	}

	_, err = Controller{SysfsRoot: root, RunRoot: ""}.AttachDevice(42, 1, 2, 2)
	if !errors.Is(err, errNoVHCIController) {
		t.Fatalf("err=%v, want %v", err, errNoVHCIController)
	}
}

func TestDetachDeviceWritesVHCIAttribute(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	controllerPath := filepath.Join(root, platformDevicesPath, "vhci_hcd.0")

	err := os.MkdirAll(controllerPath, 0o700)
	if err != nil {
		t.Fatalf("create controller path: %v", err)
	}

	status := "hub port sta spd dev sockfd local_busid\n"

	err = os.WriteFile(filepath.Join(controllerPath, statusAttribute), []byte(status), 0o600)
	if err != nil {
		t.Fatalf("write status: %v", err)
	}

	err = os.WriteFile(filepath.Join(controllerPath, detachAttribute), nil, 0o600)
	if err != nil {
		t.Fatalf("write detach: %v", err)
	}

	err = Controller{SysfsRoot: root, RunRoot: ""}.DetachDevice(3)
	if err != nil {
		t.Fatalf("detach device: %v", err)
	}

	detachPath := filepath.Join(controllerPath, detachAttribute)

	got, err := os.ReadFile(detachPath) //nolint:gosec // Test reads a file created under t.TempDir.
	if err != nil {
		t.Fatalf("read detach: %v", err)
	}

	want := "3"
	if string(got) != want {
		t.Fatalf("detach payload=%q, want %q", string(got), want)
	}
}

//nolint:cyclop // Table-shaped sysfs fixture keeps the usbip status behavior visible.
func TestListImportedDevices(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runRoot := t.TempDir()

	controllerPath := filepath.Join(root, platformDevicesPath, "vhci_hcd.0")

	err := os.MkdirAll(controllerPath, 0o700)
	if err != nil {
		t.Fatalf("create controller path: %v", err)
	}

	status := `hub port sta spd dev      sockfd local_busid
hs  0000 004 000 00000000 000000 0-0
hs  0003 006 003 00010002 000042 1-1
ss  0008 005 005 00030004 000043 2-1
`

	err = os.WriteFile(filepath.Join(controllerPath, statusAttribute), []byte(status), 0o600)
	if err != nil {
		t.Fatalf("write status: %v", err)
	}

	runPath := filepath.Join(runRoot, vhciRunDir)

	err = os.MkdirAll(runPath, 0o700)
	if err != nil {
		t.Fatalf("create run path: %v", err)
	}

	err = os.WriteFile(filepath.Join(runPath, "port3"), []byte("127.0.0.1 3240 1-1"), 0o600)
	if err != nil {
		t.Fatalf("write port record: %v", err)
	}

	usbPath := filepath.Join(root, usbDevicesPath, "1-1")

	err = os.MkdirAll(usbPath, 0o700)
	if err != nil {
		t.Fatalf("create usb path: %v", err)
	}

	err = os.WriteFile(filepath.Join(usbPath, "idVendor"), []byte("2fe3\n"), 0o600)
	if err != nil {
		t.Fatalf("write vendor: %v", err)
	}

	err = os.WriteFile(filepath.Join(usbPath, "idProduct"), []byte("0001\n"), 0o600)
	if err != nil {
		t.Fatalf("write product: %v", err)
	}

	devices, err := Controller{SysfsRoot: root, RunRoot: runRoot}.ListImportedDevices()
	if err != nil {
		t.Fatalf("list imported devices: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("len(devices)=%d, want 1", len(devices))
	}

	got := devices[0]
	if got.Port != 3 || got.Status != 6 || got.Speed != 3 || got.LocalBusID != "1-1" {
		t.Fatalf("device status=%+v", got)
	}

	if got.RemoteHost != "127.0.0.1" || got.RemotePort != "3240" || got.RemoteBusID != "1-1" {
		t.Fatalf("device remote=%+v", got)
	}

	if got.RemoteBus != 1 || got.RemoteDev != 2 {
		t.Fatalf("remote bus/dev=%03d/%03d, want 001/002", got.RemoteBus, got.RemoteDev)
	}

	if got.IDVendor != 0x2fe3 || got.IDProduct != 0x0001 {
		t.Fatalf("ids=%04x:%04x, want 2fe3:0001", got.IDVendor, got.IDProduct)
	}
}

func TestRecordConnectionWritesUSBIPPortRecord(t *testing.T) {
	t.Parallel()

	runRoot := t.TempDir()

	err := Controller{SysfsRoot: "", RunRoot: runRoot}.RecordConnection(4, "example.test", "3240", "1-1")
	if err != nil {
		t.Fatalf("record connection: %v", err)
	}

	portRecordPath := filepath.Join(runRoot, vhciRunDir, "port4")

	got, err := os.ReadFile(portRecordPath) //nolint:gosec // Test reads a file created under t.TempDir.
	if err != nil {
		t.Fatalf("read port record: %v", err)
	}

	want := "example.test 3240 1-1"
	if string(got) != want {
		t.Fatalf("record=%q, want %q", string(got), want)
	}
}
