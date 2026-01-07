---
id: E001
type: steps
---
---
## Предварительные условия
- Две машины в одной L2-сети (host + VM)
- Доступ root на обеих системах
- USB-флешка физически подключена к серверу
- Wireshark установлен на сервере

## Шаги

### 1. Подготовка USB/IP сервера (CachyOS)
```bash
sudo pacman -S usbip
sudo modprobe usbip-core usbip-host
sudo usbipd -D
usbip list -l
sudo usbip bind -b <BUSID>
```
### 2. Подготовка USB/IP клиента (Ubuntu 24 VM)
```bash
sudo apt update
sudo apt install -y usbip
sudo modprobe vhci_hcd
usbip list -r <SERVER_IP>
sudo usbip attach -r <SERVER_IP> -b <BUSID>
```

Проверка:
```bash
lsusb
lsblk
```
### 3. Захват сетевого трафика
На сервере определить интерфейс:

```bash
ip route get <CLIENT_IP>
```
Запустить захват:

```bash
sudo timeout 20s tcpdump \
  -i wlan0 \
  -s 0 \
  -w E001-usbip-no-encryption-20s.pcap \
  tcp port 3240
```

### 4. Генерация USB-нагрузки
На клиенте:
```bash
sudo dd if=/dev/sdX of=/dev/null bs=1M count=200 status=progress
```

## Ожидаемый результат (чек-лист)
 - [x] Устройство подключено по USB/IP
 - [x] TCP-сессия на порту 3240
 - [x] PCAP содержит полезную нагрузку
 - [x] Wireshark декодирует USB/IP и USB URB