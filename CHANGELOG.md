# Changelog

All notable user-visible changes are documented here.

## Unreleased

## v0.1.0

First preview release.

### Included

- `usb-quic` CLI with common `usbip`-compatible workflows.
- `usb-quicd` client/server proxy service.
- TCP and experimental QUIC stream forwarding.
- Linux `amd64` and `arm64` release archives.
- Installer, optional `usbip` replacement symlink, and disabled-by-default
  systemd services.
- USB vendor and product name lookup through the system `usb.ids` database.

### Known limitations

- QUIC requires development-only insecure TLS.
- Real-hardware and cross-distribution compatibility are not guaranteed.
- Interrupted USB/IP sessions are not recovered.
- `usbip list -d` gadget discovery is not implemented.
