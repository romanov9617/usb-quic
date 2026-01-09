#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/00_env.sh"

{
  echo "### date_utc"
  date -u
  echo "### uname"
  uname -a
  echo "### iface"
  echo "$IFACE"
  echo "### usbip_target"
  echo "${USBIP_SERVER_IP}:${USBIP_SERVER_PORT}"
  echo "### mountpoint_check"
  mountpoint -q "$MOUNTPOINT" && echo "OK" || echo "NOT_MOUNTED"
  echo "### mount_line"
  mount | grep "$MOUNTPOINT" || true
  echo "### lsblk"
  lsblk
  echo "### lsusb"
  lsusb || true
  echo "### fio_version"
  command -v fio && fio --version
  echo "### sudo_check"
  sudo -n true && echo "sudo_no_password_OK" || echo "sudo_password_may_be_required (run: sudo -v)"
  echo "### write_test_sudo"
  sudo sh -c "touch '$MOUNTPOINT/.fio_write_test' && rm -f '$MOUNTPOINT/.fio_write_test'" && echo "OK" || echo "FAIL"
} | tee "$OUTDIR/preflight.txt"

if ! mountpoint -q "$MOUNTPOINT"; then
  echo "ERROR: $MOUNTPOINT is not a mountpoint. Mount /dev/sdb1 to $MOUNTPOINT first." | tee -a "$OUTDIR/preflight.txt"
  exit 1
fi

command -v fio >/dev/null || {
  echo "ERROR: fio not found" | tee -a "$OUTDIR/preflight.txt"
  exit 1
}
