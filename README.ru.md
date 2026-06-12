# usb-quic

Языки: [English](README.md) | Русский

`usb-quic` - экспериментальный Linux-прокси USB/IP поверх QUIC и частично
совместимая замена CLI для распространённых сценариев `usbip`.

Проект предоставляет:

- `usb-quic`: операторский CLI, совместимый с основными командами `usbip`;
- `usb-quicd`: foreground proxy service для клиентской и серверной машин;
- release-архивы для Linux `amd64` и `arm64`;
- опциональную drop-in ссылку `/usr/local/bin/usbip`;
- опциональные systemd-сервисы клиента и сервера.

## Статус

Проект пока не готов к production. QUIC transport требует
`--insecure-dev-tls`: сервер создаёт временный self-signed сертификат, а клиент
не проверяет сертификат.

Автоматическими тестами проверяются:

- обработка USB/IP `DEVLIST` и `IMPORT`;
- адаптеры `vhci_hcd`, `usbip-host` и sysfs на fixtures;
- TCP и QUIC stream forwarding;
- CLI-команды `list`, `attach`, `detach`, `bind`, `unbind` и `port`.

Не гарантируются:

- production security и работа во враждебной сети;
- восстановление прерванной USB/IP-сессии;
- обнаружение gadgets через `usbip list -d`;
- совместимость с каждым ядром и USB-устройством;
- unattended end-to-end работа на реальном оборудовании.

Поэтому для первого публичного релиза рекомендуется версия `v0.1.0`, а не
`v1.0.0`.

## Установка

Скачайте архив для своей архитектуры из GitHub Releases, проверьте и распакуйте:

```bash
sha256sum -c SHA256SUMS --ignore-missing
tar -xzf usb-quic_v0.1.0_linux_amd64.tar.gz
cd usb-quic_v0.1.0_linux_amd64
sudo ./install.sh
```

Будут установлены:

- `/usr/local/bin/usb-quic`
- `/usr/local/bin/usb-quicd`
- выключенные `usb-quic-client.service` и `usb-quic-server.service`
- примеры конфигурации в `/etc/usb-quic`

Installer не запускает сервис автоматически: адрес сервера и security policy
невозможно безопасно выбрать без участия пользователя.

### Замена команды `usbip`

Чтобы создать `/usr/local/bin/usbip` как ссылку на `usb-quic`:

```bash
sudo ./install.sh --replace-usbip
```

Это надёжнее shell alias и работает с `sudo`. Штатный бинарник остаётся по
исходному пути, обычно `/usr/sbin/usbip`.

## Настройка Сервисов

Следующая конфигурация использует небезопасный development TLS.

### Серверная машина

Штатный `usbipd` должен быть установлен, запущен и экспортировать нужные
устройства:

```bash
sudo cp /etc/usb-quic/server.env.example /etc/usb-quic/server.env
sudo systemctl enable --now usb-quic-server.service
sudo systemctl status usb-quic-server.service
```

Пример слушает QUIC на UDP-порту `4242` и передаёт streams штатному `usbipd` на
`127.0.0.1:3240`. При необходимости разрешите UDP-порт `4242` в firewall.

### Клиентская машина

Перед включением сервиса замените `SERVER_ADDRESS`:

```bash
sudo cp /etc/usb-quic/client.env.example /etc/usb-quic/client.env
sudo editor /etc/usb-quic/client.env
sudo systemctl enable --now usb-quic-client.service
sudo systemctl status usb-quic-client.service
```

Клиентский пример открывает proxy только на `127.0.0.1:3240`:

```bash
usb-quic list -r 127.0.0.1
```

Kernel-операции `attach`, `detach`, `bind` и `unbind` могут требовать root-прав
и соответствующих kernel modules.

## CLI

Основные команды:

```bash
usb-quic list -l
usb-quic list -r HOST
sudo usb-quic attach -r HOST -b BUSID
sudo usb-quic detach -p PORT
usb-quic port
sudo usb-quic bind -b BUSID
sudo usb-quic unbind -b BUSID
```

Имена USB vendor/product загружаются из системной базы `usb.ids`, если она
доступна.

Параметры proxy service выводятся командой `usb-quicd --help`. Запуск
`usb-quicd` без явного `--transport` завершается ошибкой вместо выбора
потенциально небезопасной сетевой конфигурации.

## Разработка

```bash
make build
make test
```

Локальные бинарники создаются как `dist/usb-quic` и `dist/usb-quicd`.

Процесс релиза и требования к `v1.0.0` описаны в
[docs/RELEASING.md](docs/RELEASING.md).

## Удаление

Из распакованного release-архива:

```bash
sudo ./uninstall.sh
```

Конфигурация в `/etc/usb-quic` сохраняется.
