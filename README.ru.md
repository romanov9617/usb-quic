# usb-quic

Языки: [English](README.md) | Русский

`usb-quic` - проект userspace-прокси для USB/IP поверх QUIC.

Цель продукта - не только туннелировать USB/IP-трафик. CLI должен стать
практической заменой legacy workflow команды `usbip`, пригодной для
использования через shell alias, wrapper или стратегию drop-in binary.

Текущий статус: CLI-каркас совместимости и ранний daemon TCP forwarding
prototype. QUIC transport и TLS в текущем коде не реализованы.

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
| `attach` | `-r, --remote HOST`; `-b, --busid BUSID`; `-d, --device DEVID` | Возвращает `ErrNotImplemented`. Флаги парсятся, но не реализованы. | Форма интерфейса совпадает с наблюдаемыми legacy flags; поведение не реализовано. |
| `detach` | `-p, --port PORT` | Возвращает `ErrNotImplemented`. | Не реализовано. |
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

- `attach` пока не требует и не использует `--busid` / `--device`.
- `attach` является только CLI-заглушкой и не выполняет legacy attach behavior.
- `list -r` выводит remote exportable devices, но имена vendor/product пока являются placeholder-текстом до добавления USB ID database.
- `list -l`, `list -p`, `list -d`, `detach`, `bind`, `unbind` и `port` не реализованы.
- Daemon command запускает ранний TCP forwarding service, но еще не финальный QUIC proxy.
- QUIC transport и TLS configuration в текущем коде не реализованы.

## Направление Реализации

Текущая структура команд разделяет:

- operator command surface `usb-quic`, и
- `usbipd`-подобный daemon entrypoint.

Пользовательский command API должен сохранять legacy-семантику команд. Запуск
proxy и детали QUIC transport должны быть скрыты за совместимым workflow там,
где это возможно. Daemon behavior должен находиться в отдельном daemon
entrypoint, а не в operator-facing команде `usb-quic`.

## Ручная проверка TCP-to-QUIC transport

Daemon entrypoint содержит скрытые `usb-quic` flags для локальной проверки
текущего transport path без изменения usbipd-like help output. Smoke test
использует ephemeral self-signed certificate и отключает client-side
certificate verification, поэтому это не production TLS configuration.

Запустите в трех отдельных терминалах:

```bash
python3 - <<'PY'
import socket

sock = socket.socket()
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.bind(("127.0.0.1", 19000))
sock.listen()
print("echo upstream listening on 127.0.0.1:19000", flush=True)

while True:
    conn, addr = sock.accept()
    print(f"echo accepted {addr}", flush=True)
    with conn:
        while data := conn.recv(65536):
            conn.sendall(data)
PY
```

```bash
GOCACHE=/tmp/usb-quic-go-build-cache go run ./cmd/daemon \
  --debug \
  --usb-quic-transport quic-server \
  --usb-quic-quic-listen 127.0.0.1:14242 \
  --usb-quic-upstream 127.0.0.1:19000 \
  --usb-quic-dev-insecure-tls
```

```bash
GOCACHE=/tmp/usb-quic-go-build-cache go run ./cmd/daemon \
  --debug \
  --tcp-port 13241 \
  --usb-quic-transport quic-client \
  --usb-quic-quic-addr 127.0.0.1:14242 \
  --usb-quic-dev-insecure-tls
```

Затем отправьте данные в TCP entrypoint:

```bash
printf 'hello over quic\n' | timeout 3 nc 127.0.0.1 13241
```

Ожидаемый вывод:

```text
hello over quic
```

В логах daemon должны быть события TCP client accept, QUIC server listen и
tunnel start/stop на обеих сторонах.
