# usb-quic

Languages: English | [Русский](README.ru.md)

`usb-quic` is a userspace USB/IP-over-QUIC proxy project.

The product goal is not only to tunnel USB/IP traffic. The CLI must become a
practical replacement for the legacy `usbip` user workflow, suitable for use
through a shell alias, wrapper, or drop-in binary strategy.

Current status: CLI compatibility scaffold. QUIC transport and TLS are not
implemented in the current codebase.

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

Current root usage:

```text
usage: usb-quic [-v] [--debug] [--log] [--tcp-port PORT] [version] [help] <command> <args>

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
| `-v`, `--verbose` | parsed and used | Enables debug logging. This is an extension beyond the observed legacy `usbip` API. |

## Command Surface

| Command | Current flags | Current behavior | Compatibility status |
| --- | --- | --- | --- |
| `version` | none | Prints build-time version, or `dev` in local builds. | Partial: output format does not match `usbip (usbip-utils 2.0)`. |
| `help` | none | Prints root help. | Mostly compatible at root level. |
| `attach` | `-r, --remote HOST`; `-b, --busid BUSID` | Returns `ErrNotImplemented`. `busid` is parsed but not used. | Not implemented. |
| `detach` | `-p, --port PORT` | Returns `ErrNotImplemented`. | Not implemented. |
| `list` | `-l, --local`; `-r, --remote HOST`; `-p, --parsable` | Returns `ErrNotImplemented`. `--local`, `--remote`, and `--parsable` are parsed but not implemented. | Not implemented. |
| `bind` | `-b, --busid BUSID` | Returns `ErrNotImplemented`. | Not implemented. |
| `unbind` | `-b, --busid BUSID` | Returns `ErrNotImplemented`. | Not implemented. |
| `port` | none | Returns `ErrNotImplemented`. | Not implemented. |

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

- `attach` is missing `-d, --device`.
- `list` is missing `-d, --device`.
- `attach` does not require or use `--busid`.
- `attach` is only a CLI stub and does not perform legacy attach behavior.
- `list -r` is only a CLI stub and does not print remote exportable devices.
- `list -l`, `list -p`, `detach`, `bind`, `unbind`, and `port` are not implemented.
- Subcommand help currently uses the root help template instead of legacy-style subcommand usage.
- The CLI has separate client and server roles, while legacy `usbip` exposes one operator-facing command.
- Server role returns `listen: command is not implemented yet` when invoked without a command.
- QUIC transport and TLS configuration are not implemented in the current codebase.

## Implementation Direction

The next CLI design step is to separate:

- the `usbip`-compatible command surface, and
- proxy daemon or service control behavior.

The user-facing command API should preserve legacy command semantics. The proxy
startup and QUIC transport details should be hidden behind the compatible
workflow where possible, or exposed through clearly separate service commands
when they cannot be hidden.
