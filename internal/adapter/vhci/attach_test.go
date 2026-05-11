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

	port, err := Controller{SysfsRoot: root}.AttachDevice(42, 1, 2, 2)
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

	port, err := Controller{SysfsRoot: root}.AttachDevice(42, 1, 2, 5)
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

	_, err = Controller{SysfsRoot: root}.AttachDevice(42, 1, 2, 2)
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

	err = Controller{SysfsRoot: root}.DetachDevice(3)
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
