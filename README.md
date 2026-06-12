# usb-quic

Languages: English | [Русский](README.ru.md)

`usb-quic` is an experimental Linux USB/IP-over-QUIC proxy and a mostly
compatible replacement for common `usbip` CLI workflows.

The project currently provides:

- `usb-quic`: operator CLI compatible with common `usbip` commands;
- `usb-quicd`: foreground proxy service for client and server machines;
- release archives for Linux `amd64` and `arm64`;
- optional drop-in `/usr/local/bin/usbip` symlink;
- optional systemd client and server services.

## Status

This project is not production-ready yet. QUIC transport currently requires
`--insecure-dev-tls`, which uses an ephemeral self-signed server certificate
and disables certificate verification on the client.

Verified by automated tests:

- USB/IP `DEVLIST` and `IMPORT` protocol handling;
- local `vhci_hcd`, `usbip-host`, and sysfs adapters using fixtures;
- TCP and QUIC stream forwarding;
- CLI behavior for `list`, `attach`, `detach`, `bind`, `unbind`, and `port`.

Not guaranteed:

- production security or hostile-network use;
- recovery of an interrupted USB/IP session;
- `usbip list -d` gadget discovery;
- compatibility with every kernel and USB device;
- unattended end-to-end operation on real hardware.

For that reason, the recommended first public release is `v0.1.0`, not
`v1.0.0`.

## Installation

Download the archive for your architecture from GitHub Releases, verify it,
extract it, and run:

```bash
sha256sum -c SHA256SUMS --ignore-missing
tar -xzf usb-quic_v0.1.0_linux_amd64.tar.gz
cd usb-quic_v0.1.0_linux_amd64
sudo ./install.sh
```

This installs:

- `/usr/local/bin/usb-quic`
- `/usr/local/bin/usb-quicd`
- disabled `usb-quic-client.service` and `usb-quic-server.service`
- configuration examples under `/etc/usb-quic`

The installer does not start a service because the server address and security
policy cannot be chosen safely automatically.

### Replace the `usbip` command

To create `/usr/local/bin/usbip` as a symlink to `usb-quic`:

```bash
sudo ./install.sh --replace-usbip
```

This works more reliably than a shell alias, including with `sudo`. The stock
binary remains available at its original path, commonly `/usr/sbin/usbip`.

## Service Setup

The following setup uses development-only insecure TLS.

### Server machine

The stock `usbipd` must be installed, running, and exporting the required USB
devices. Configure and start the server-side proxy:

```bash
sudo cp /etc/usb-quic/server.env.example /etc/usb-quic/server.env
sudo systemctl enable --now usb-quic-server.service
sudo systemctl status usb-quic-server.service
```

The example listens for QUIC on UDP port `4242` and forwards streams to stock
`usbipd` at `127.0.0.1:3240`. Allow UDP port `4242` in the firewall if needed.

### Client machine

Edit `SERVER_ADDRESS` before enabling the client service:

```bash
sudo cp /etc/usb-quic/client.env.example /etc/usb-quic/client.env
sudo editor /etc/usb-quic/client.env
sudo systemctl enable --now usb-quic-client.service
sudo systemctl status usb-quic-client.service
```

The client example exposes the proxy only on `127.0.0.1:3240`. Query it with:

```bash
usb-quic list -r 127.0.0.1
```

Kernel operations such as `attach`, `detach`, `bind`, and `unbind` may require
root privileges and the appropriate kernel modules.

## CLI

Common commands:

```bash
usb-quic list -l
usb-quic list -r HOST
sudo usb-quic attach -r HOST -b BUSID
sudo usb-quic detach -p PORT
usb-quic port
sudo usb-quic bind -b BUSID
sudo usb-quic unbind -b BUSID
```

USB vendor and product names are loaded from a system `usb.ids` database when
available.

Run `usb-quicd --help` for proxy service options. Running `usb-quicd` without
an explicit `--transport` fails instead of selecting a potentially unsafe
network configuration.

## Development

```bash
make build
make test
```

Local binaries are written to `dist/usb-quic` and `dist/usb-quicd`.

Release procedure and `v1.0.0` requirements are documented in
[docs/RELEASING.md](docs/RELEASING.md).

## Uninstall

From an extracted release archive:

```bash
sudo ./uninstall.sh
```

Configuration under `/etc/usb-quic` is preserved.
