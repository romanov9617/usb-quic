---
id: E001
type: artifacts
---

# Artifacts (E001)

## Файлы
- **PCAP**: `docs/_assets/pcap/E001-usbip-no-encryption-30k.pcap`
  - Размер: ~30–50 МБ
  - Фильтр: `tcp port 3240`

- **Скриншоты**:
  - `docs/_assets/img/E001-wireshark-usbip-usb-urb.png`
  - `docs/_assets/img/E001-wireshark-usb-mass-storage.png`

- **Логи**:
  - `docs/_assets/logs/E001-tcpdump.log`

## Контроль качества
- [x] PCAP открывается в Wireshark
- [x] Присутствует TCP/3240
- [x] Декодирование USB/IP включается автоматически
- [x] Видны URB Submit / Response
- [x] Присутствуют сигнатуры USB Mass Storage

## Примечания по приватности
- Захват ограничен по времени
- Личные файлы и имена не анализируются
- PCAP используется исключительно в исследовательских целях
