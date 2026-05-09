# usb-quic

Languages: English | [Русский](README.ru.md)

`usb-quic` is a userspace USB/IP-over-QUIC proxy project.

The product goal is not only to tunnel USB/IP traffic. The CLI must become a
practical replacement for the legacy `usbip` user workflow, suitable for use
through a shell alias, wrapper, or drop-in binary strategy.

Current status: CLI compatibility scaffolds plus an early daemon TCP forwarding
prototype. QUIC transport and TLS are not implemented in the current codebase.

## Compatibility Goal

The project targets compatibility with stock Linux USB/IP components:

- `usbip`
- `usbipd`
- `usbip-host`
- `vhci-hcd`

The edge protocol must remain plain USB/IP over local TCP. QUIC is used only
between the client-side and server-side proxies.

Target topology:

```text
[ Linux client ]
  usbip / vhci-hcd
        |
        | TCP 127.0.0.1:3240
        v
  client-side proxy
        |
        | QUIC
        v
  server-side proxy
        |
        | TCP 127.0.0.1:3240
        v
  usbipd -> usbip-host -> physical USB device
```

Core invariant:

```text
1 TCP USB/IP session = 1 QUIC stream
```

## CLI Compatibility Contract

The CLI should follow the observable API of the legacy `usbip` command.

At minimum, common user workflows should remain familiar:

- `usbip list -r HOST`
- `usbip attach -r HOST -b BUSID`
- `usbip detach -p PORT`
- `usbip port`
- `usbip list -l`
- `usbip bind -b BUSID`
- `usbip unbind -b BUSID`

Any compatibility gap should be treated as either:

- a product defect to fix, or
- an explicit limitation documented here.

## Current CLI API

The current command tree is implemented with Cobra in:

- `internal/adapter/delivery/cli/root.go`
- `internal/adapter/delivery/cli/commands.go`

The operator-facing binary is assembled from `cmd/usb-quic`.

Current root usage:

```text
usage: usb-quic [--debug] [--log] [--tcp-port PORT] [version]
             [help] <command> <args>

  attach     Attach a remote USB device
  detach     Detach a remote USB device
  list       List exportable or local USB devices
  bind       Bind device to usbip-host.ko
  unbind     Unbind device from usbip-host.ko
  port       Show imported USB devices
```

### Global Flags

| Flag | Current status | Notes |
| --- | --- | --- |
| `--debug` | parsed | Present for `usbip` shape compatibility. |
| `--log` | parsed | Present for `usbip` shape compatibility. |
| `--tcp-port PORT` | parsed and used | Defaults to USB/IP port `3240`. |

## Command Surface

| Command | Current flags | Current behavior | Compatibility status |
| --- | --- | --- | --- |
| `version` | none | Prints build-time version, or `dev` in local builds. | Partial: output format does not match `usbip (usbip-utils 2.0)`. |
| `help` | none | Prints root help. | Mostly compatible at root level. |
| `attach` | `-r, --remote HOST`; `-b, --busid BUSID`; `-d, --device DEVID` | Returns `ErrNotImplemented`. Flags are parsed but not implemented. | Interface shape matches the observed legacy flags; behavior is not implemented. |
| `detach` | `-p, --port PORT` | Returns `ErrNotImplemented`. | Not implemented. |
| `list` | `-p, --parsable`; `-r, --remote HOST`; `-l, --local`; `-d, --device` | Returns `ErrNotImplemented`. Flags are parsed but not implemented. | Interface shape matches the observed legacy flags; behavior is not implemented. |
| `bind` | `-b, --busid BUSID` | Returns `ErrNotImplemented`. | Not implemented. |
| `unbind` | `-b, --busid BUSID` | Returns `ErrNotImplemented`. | Not implemented. |
| `port` | none | Returns `ErrNotImplemented`. | Not implemented. |

## Current Daemon API

The daemon entrypoint is separate from the `usb-quic` operator CLI, as
in the original USB/IP userspace tools. It is assembled from `cmd/daemon` and
implemented in `internal/adapter/delivery/daemon`.

Current daemon usage:

```text
usage: usbipd [options]

	-4, --ipv4
		Bind to IPv4. Default is both.

	-6, --ipv6
		Bind to IPv6. Default is both.

	-e, --device
		Run in device mode.
		Rather than drive an attached device, create
		a virtual UDC to bind gadgets to.

	-D, --daemon
		Run as a daemon process.

	-d, --debug
		Print debugging information.

	-PFILE, --pid FILE
		Write process id to FILE.
		If no FILE specified, use /var/run/usbipd.pid

	-tPORT, --tcp-port PORT
		Listen on TCP/IP port PORT.

	-h, --help
		Print this help.

	-v, --version
		Show version.
```

All daemon flags are parsed. The daemon runtime can listen for local TCP
sessions and forward each accepted session through the current stream transport
adapter. The adapter is TCP-backed today; QUIC transport and TLS are still not
implemented.

## Observed Legacy USB/IP API

The local system `usbip` command reports this root usage:

```text
usage: usbip [--debug] [--log] [--tcp-port PORT] [version]
             [help] <command> <args>

  attach     Attach a remote USB device
  detach     Detach a remote USB device
  list       List exportable or local USB devices
  bind       Bind device to usbip-host.ko
  unbind     Unbind device from usbip-host.ko
  port       Show imported USB devices
```

Observed subcommand usage:

```text
usage: usbip attach <args>
    -r, --remote=<host>      The machine with exported USB devices
    -b, --busid=<busid>      Busid of the device on <host>
    -d, --device=<devid>     Id of the virtual UDC on <host>
```

```text
usage: usbip list [-p|--parsable] <args>
    -p, --parsable         Parsable list format
    -r, --remote=<host>    List the exportable USB devices on <host>
    -l, --local            List the local USB devices
    -d, --device           List the local USB gadgets bound to usbip-vudc
```

```text
usage: usbip detach <args>
    -p, --port=<port>    vhci_hcd port the device is on
```

```text
usage: usbip bind <args>
    -b, --busid=<busid>    Bind usbip-host.ko to device on <busid>
```

```text
usage: usbip unbind <args>
    -b, --busid=<busid>    Unbind usbip-host.ko from device on <busid>
```

## Current Compatibility Gaps

- `attach` does not require or use `--busid` / `--device` yet.
- `attach` is only a CLI stub and does not perform legacy attach behavior.
- `list -r` is only a CLI stub and does not print remote exportable devices.
- `list -l`, `list -p`, `list -d`, `detach`, `bind`, `unbind`, and `port` are not implemented.
- The daemon command starts an early TCP forwarding service, not the final QUIC proxy yet.
- QUIC transport and TLS configuration are not implemented in the current codebase.

## Implementation Direction

The current command structure separates:

- the `usb-quic` operator command surface, and
- the `usbipd`-like daemon entrypoint.

The user-facing command API should preserve legacy command semantics. The proxy
startup and QUIC transport details should be hidden behind the compatible
workflow where possible. Daemon behavior belongs in the separate daemon
entrypoint, not in the operator-facing `usb-quic` command.
