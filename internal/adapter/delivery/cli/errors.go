package cli

import "errors"

// ErrNotImplemented marks commands whose USB/IP behavior is not wired yet.
var ErrNotImplemented = errors.New("command is not implemented yet")

// ErrAttachRemoteRequired marks attach calls without a remote host.
var ErrAttachRemoteRequired = errors.New("attach: remote host is required")

// ErrAttachBusIDRequired marks attach calls without a busid.
var ErrAttachBusIDRequired = errors.New("attach: busid is required")

// ErrDetachPortRequired marks detach calls without a vhci port.
var ErrDetachPortRequired = errors.New("detach: port is required")

// ErrDeviceBusIDRequired marks local device commands without a busid.
var ErrDeviceBusIDRequired = errors.New("busid is required")

// ErrUnexpectedConnectionType marks non-TCP attach dials.
var ErrUnexpectedConnectionType = errors.New("unexpected usbip connection type")
