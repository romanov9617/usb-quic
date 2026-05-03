# usb-quic

Языки: [English](README.md) | Русский

`usb-quic` - проект userspace-прокси для USB/IP поверх QUIC.

Цель продукта - не только туннелировать USB/IP-трафик. CLI должен стать
практической заменой legacy workflow команды `usbip`, пригодной для
использования через shell alias, wrapper или стратегию drop-in binary.

Текущий статус: ранний прототип.

## Цель Совместимости

Проект ориентирован на совместимость со штатными Linux USB/IP-компонентами:

- `usbip`
- `usbipd`
- `usbip-host`
- `vhci-hcd`

На краях протокол должен оставаться обычным USB/IP поверх локального TCP. QUIC
используется только между client-side и server-side прокси.

Целевая топология:

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

Ключевой инвариант:

```text
1 TCP USB/IP session = 1 QUIC stream
```

## Контракт CLI-Совместимости

CLI должен следовать наблюдаемому API legacy-команды `usbip`.

Как минимум, распространенные пользовательские сценарии должны оставаться
привычными:

- `usbip list -r HOST`
- `usbip attach -r HOST -b BUSID`
- `usbip detach -p PORT`
- `usbip port`
- `usbip list -l`
- `usbip bind -b BUSID`
- `usbip unbind -b BUSID`

Любой разрыв совместимости следует считать:

- дефектом продукта, который нужно исправить, или
- явным ограничением, задокументированным здесь.

## Текущий CLI API

Текущее дерево команд реализовано через Cobra в:

- `internal/adapter/delivery/cli/root.go`
- `internal/adapter/delivery/cli/commands.go`

Текущий root usage:

```text
usage: usb-quic [-v] [--debug] [--log] [--tcp-port PORT] [version] [help] <command> <args>

  attach     Attach a remote USB device
  detach     Detach a remote USB device
  list       List exportable or local USB devices
  bind       Bind device to usbip-host.ko
  unbind     Unbind device from usbip-host.ko
  port       Show imported USB devices
```

### Глобальные Флаги

| Флаг | Текущий статус | Примечания |
| --- | --- | --- |
| `--debug` | парсится | Присутствует для совместимости с формой `usbip`. |
| `--log` | парсится | Присутствует для совместимости с формой `usbip`. |
| `--tcp-port PORT` | парсится и используется | По умолчанию USB/IP port `3240`. |
| `-v`, `--verbose` | парсится и используется | Включает debug logging. Это расширение относительно наблюдаемого legacy API `usbip`. |

## Поверхность Команд

| Команда | Текущие флаги | Текущее поведение | Статус совместимости |
| --- | --- | --- | --- |
| `version` | нет | Печатает build-time version или `dev` в локальных сборках. | Частично: формат вывода не совпадает с `usbip (usbip-utils 2.0)`. |
| `help` | нет | Печатает root help. | В основном совместимо на root-уровне. |
| `attach` | `-r, --remote HOST`; `-b, --busid BUSID` | В client role, если задан `--remote`, запускает TCP-to-QUIC proxying. `busid` парсится, но не используется. | Пока не Liskov-compatible. Команда не выполняет legacy attach semantics как одноразовая пользовательская команда. |
| `detach` | `-p, --port PORT` | Возвращает `ErrNotImplemented`. | Не реализовано. |
| `list` | `-l, --local`; `-r, --remote HOST`; `-p, --parsable` | В client role, если задан `--remote`, запускает TCP-to-QUIC proxying. `--local` и `--parsable` парсятся, но не реализованы. | Пока не Liskov-compatible. Команда не возвращает legacy device list. |
| `bind` | `-b, --busid BUSID` | Возвращает `ErrNotImplemented`. | Не реализовано. |
| `unbind` | `-b, --busid BUSID` | Возвращает `ErrNotImplemented`. | Не реализовано. |
| `port` | нет | Возвращает `ErrNotImplemented`. | Не реализовано. |

## Наблюдаемый Legacy USB/IP API

Локальная системная команда `usbip` сообщает такой root usage:

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

Наблюдаемый usage подкоманд:

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

## Текущие Разрывы Совместимости

- В `attach` отсутствует `-d, --device`.
- В `list` отсутствует `-d, --device`.
- `attach` не требует и не использует `--busid`.
- `attach` сейчас запускает proxying вместо поведения legacy attach command.
- `list -r` сейчас запускает proxying вместо вывода remote exportable devices.
- `list -l`, `list -p`, `detach`, `bind`, `unbind` и `port` не реализованы.
- Help подкоманд сейчас использует root help template вместо legacy-style subcommand usage.
- CLI имеет отдельные client и server роли, тогда как legacy `usbip` предоставляет одну operator-facing команду.
- Server role запускает QUIC-to-TCP listener при вызове без команды. Это полезно для прототипа, но не является частью legacy `usbip` command API.

## Направление Реализации

Следующий шаг дизайна CLI - разделить:

- `usbip`-совместимую пользовательскую поверхность команд, и
- поведение proxy daemon или service control.

Пользовательский command API должен сохранять legacy-семантику команд. Запуск
proxy и детали QUIC transport должны быть скрыты за совместимым workflow там,
где это возможно, или вынесены в явно отдельные service commands там, где это
невозможно скрыть.
