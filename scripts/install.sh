#!/bin/sh
set -eu

prefix="/usr/local"
replace_usbip=false

usage() {
	echo "usage: sudo ./install.sh [--prefix PATH] [--replace-usbip]"
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--prefix)
		prefix="${2:?--prefix requires a path}"
		shift 2
		;;
	--replace-usbip)
		replace_usbip=true
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		usage >&2
		exit 2
		;;
	esac
done

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
bin_dir="${prefix}/bin"

install -d "${bin_dir}"
install -m 0755 "${script_dir}/usb-quic" "${bin_dir}/usb-quic"
install -m 0755 "${script_dir}/usb-quicd" "${bin_dir}/usb-quicd"

services_installed=false
if [ "${prefix}" = "/usr/local" ] && [ -d /etc/systemd/system ]; then
	install -d /etc/usb-quic
	install -m 0644 "${script_dir}/config/client.env.example" /etc/usb-quic/client.env.example
	install -m 0644 "${script_dir}/config/server.env.example" /etc/usb-quic/server.env.example
	install -m 0644 "${script_dir}/systemd/usb-quic-client.service" /etc/systemd/system/usb-quic-client.service
	install -m 0644 "${script_dir}/systemd/usb-quic-server.service" /etc/systemd/system/usb-quic-server.service

	if command -v systemctl >/dev/null 2>&1; then
		systemctl daemon-reload
	fi
	services_installed=true
fi

if [ "${replace_usbip}" = true ]; then
	usbip_link="${bin_dir}/usbip"
	if [ -e "${usbip_link}" ] || [ -L "${usbip_link}" ]; then
		if [ ! -L "${usbip_link}" ] || [ "$(readlink "${usbip_link}")" != "${bin_dir}/usb-quic" ]; then
			echo "refusing to replace existing ${usbip_link}" >&2
			exit 1
		fi
	fi

	ln -sfn "${bin_dir}/usb-quic" "${usbip_link}"
fi

echo "Installed usb-quic and usb-quicd into ${bin_dir}."
if [ "${services_installed}" = true ]; then
	echo "Services were installed but not enabled. Configure /etc/usb-quic before starting them."
fi
