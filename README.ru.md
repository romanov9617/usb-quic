# usb-quic

Языки: [English](README.md) | Русский

`usb-quic` - проект userspace-прокси для USB/IP поверх QUIC.

Цель продукта - не только туннелировать USB/IP-трафик. CLI должен стать
практической заменой legacy workflow команды `usbip`, пригодной для
использования через shell alias, wrapper или стратегию drop-in binary.

Текущий статус: CLI-каркас совместимости, TCP forwarding и experimental QUIC
client/server transport. Production TLS configuration еще предстоит добавить.

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

Operator-facing binary собирается из `cmd/usb-quic`.

Текущий root usage:

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

### Глобальные Флаги

| Флаг | Текущий статус | Примечания |
| --- | --- | --- |
| `--debug` | парсится | Присутствует для совместимости с формой `usbip`. |
| `--log` | парсится | Присутствует для совместимости с формой `usbip`. |
| `--tcp-port PORT` | парсится и используется | По умолчанию USB/IP port `3240`. |

## Поверхность Команд

| Команда | Текущие флаги | Текущее поведение | Статус совместимости |
| --- | --- | --- | --- |
| `version` | нет | Печатает build-time version или `dev` в локальных сборках. | Частично: формат вывода не совпадает с `usbip (usbip-utils 2.0)`. |
| `help` | нет | Печатает root help. | В основном совместимо на root-уровне. |
| `attach` | `-r, --remote HOST`; `-b, --busid BUSID`; `-d, --device DEVID` | Отправляет `OP_REQ_IMPORT` на `HOST:--tcp-port`, получает `OP_REP_IMPORT` и подключает импортированный socket к локальному `vhci_hcd` через sysfs. | Начальная Linux-реализация; требует `vhci_hcd` и прав на запись в sysfs attribute `attach`. |
| `detach` | `-p, --port PORT` | Отключает локальный `vhci_hcd` port через kernel sysfs attribute `detach`. | Начальная Linux-реализация; требует прав на запись в sysfs attribute `detach`. |
| `list` | `-p, --parsable`; `-r, --remote HOST`; `-l, --local`; `-d, --device` | `list -r HOST` отправляет `OP_REQ_DEVLIST` на `HOST:--tcp-port` и печатает экспортируемые устройства. Остальные режимы `list` возвращают `ErrNotImplemented`. | Remote list частично реализован; local, parsable и device modes не реализованы. |
| `bind` | `-b, --busid BUSID` | Возвращает `ErrNotImplemented`. | Не реализовано. |
| `unbind` | `-b, --busid BUSID` | Возвращает `ErrNotImplemented`. | Не реализовано. |
| `port` | нет | Возвращает `ErrNotImplemented`. | Не реализовано. |

## Текущий Daemon API

Daemon entrypoint отделен от operator CLI `usb-quic`, как в
оригинальных USB/IP userspace tools. Он собирается из `cmd/daemon` и реализован
в `internal/adapter/delivery/daemon`.

Текущий daemon usage:

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

Все daemon flags парсятся. Daemon runtime уже может слушать локальные TCP
sessions и прокидывать каждую принятую session через текущий stream transport
adapter. Сейчас adapter основан на TCP; QUIC transport и TLS еще не
реализованы.

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

- `attach` сейчас поддерживает основной remote import path через `vhci_hcd`;
  `--device` принимается как legacy-compatible alias для import id.
- `list -r` выводит remote exportable devices, но имена vendor/product пока являются placeholder-текстом до добавления USB ID database.
- `list -l`, `list -p`, `list -d`, `bind`, `unbind` и `port` не реализованы.
- Daemon command поддерживает TCP forwarding и experimental QUIC client/server
  transport modes.
- QUIC smoke tests сейчас используют ephemeral self-signed certificate с
  `--usb-quic-dev-insecure-tls`; production TLS configuration еще предстоит
  добавить.

## Направление Реализации

Текущая структура команд разделяет:

- operator command surface `usb-quic`, и
- `usbipd`-подобный daemon entrypoint.

Пользовательский command API должен сохранять legacy-семантику команд. Запуск
proxy и детали QUIC transport должны быть скрыты за совместимым workflow там,
где это возможно. Daemon behavior должен находиться в отдельном daemon
entrypoint, а не в operator-facing команде `usb-quic`.

## Ручная проверка `usb-quic list -r` с настоящим `usbipd`

Daemon entrypoint содержит скрытые `usb-quic` flags для локальной проверки
текущего transport path без изменения usbipd-like help output. Smoke test
использует ephemeral self-signed certificate и отключает client-side
certificate verification, поэтому это не production TLS configuration.

Сначала соберите binaries:

```bash
make build
```

Эта проверка проходит первый USB/IP command path end to end с настоящим
`usbipd`:

```text
usb-quic list -r -> local TCP -> QUIC stream -> TCP upstream
usb-quic attach -r -> local TCP -> QUIC stream -> TCP upstream -> vhci_hcd
```

На машине, к которой подключено USB-устройство:

```bash
sudo modprobe usbip-host
sudo usbipd -D
```

В другом терминале на этой же машине посмотрите локальные USB devices и
экспортируйте нужное:

```bash
usbip list -l
sudo usbip bind -b <BUSID>
```

Запустите QUIC server с upstream на настоящий порт `usbipd`:

```bash
./dist/daemon \
  --debug \
  --usb-quic-transport quic-server \
  --usb-quic-quic-listen 127.0.0.1:14242 \
  --usb-quic-upstream 127.0.0.1:3240 \
  --usb-quic-dev-insecure-tls
```

Запустите client-side TCP entrypoint:

```bash
./dist/daemon \
  --debug \
  --tcp-port 13241 \
  --usb-quic-transport quic-client \
  --usb-quic-quic-addr 127.0.0.1:14242 \
  --usb-quic-dev-insecure-tls
```

Затем запросите exported devices через tunnel:

```bash
./dist/usb-quic \
  --tcp-port 13241 \
  list -r 127.0.0.1
```

В выводе должны появиться устройства, exported настоящим `usbipd`.

Чтобы импортировать одно из этих устройств через тот же tunnel, загрузите
`vhci_hcd` на client-машине и запустите attach с правами, достаточными для
kernel sysfs attach API:

```bash
sudo modprobe vhci-hcd
sudo ./dist/usb-quic \
  --tcp-port 13241 \
  attach -r 127.0.0.1 -b <BUSID>
```

Затем отключите импортированное устройство от выбранного локального vhci port:

```bash
sudo ./dist/usb-quic detach -p <PORT>
```
