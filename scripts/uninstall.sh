#!/bin/sh
set -eu

prefix="${PREFIX:-/usr/local}"
bin_dir="${prefix}/bin"
usbip_link="${bin_dir}/usbip"

if [ "${prefix}" = "/usr/local" ] && command -v systemctl >/dev/null 2>&1; then
	systemctl disable --now usb-quic-client.service usb-quic-server.service 2>/dev/null || true
fi

if [ -L "${usbip_link}" ] && [ "$(readlink "${usbip_link}")" = "${bin_dir}/usb-quic" ]; then
	rm -f "${usbip_link}"
fi

rm -f "${bin_dir}/usb-quic" "${bin_dir}/usb-quicd"
if [ "${prefix}" = "/usr/local" ]; then
	rm -f /etc/systemd/system/usb-quic-client.service /etc/systemd/system/usb-quic-server.service
fi

if [ "${prefix}" = "/usr/local" ] && command -v systemctl >/dev/null 2>&1; then
	systemctl daemon-reload
fi

echo "Removed usb-quic binaries and services. Configuration under /etc/usb-quic was preserved."
