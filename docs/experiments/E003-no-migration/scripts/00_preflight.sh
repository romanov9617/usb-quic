#!/usr/bin/env bash
set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"

need_cmd usbip
need_cmd fio
need_cmd ip
need_cmd mount
need_cmd umount

log "preflight: start"
event "preflight_start"

# mountpoint sanity
mkdir -p "${MOUNTPOINT}"

# ensure device exists
[[ -b "${DEV}" ]] || die "block device not found: ${DEV}"

# remount fresh
if mount | grep -qE " ${MOUNTPOINT} "; then
  log "preflight: umount ${MOUNTPOINT}"
  umount "${MOUNTPOINT}" || umount -l "${MOUNTPOINT}" || true
fi

log "preflight: mount ${DEV} -> ${MOUNTPOINT}"
mount "${DEV}" "${MOUNTPOINT}"

# sanity write-test
log "preflight: write-test"
sh -c "echo test > ${MOUNTPOINT}/.write_test && sync && rm -f ${MOUNTPOINT}/.write_test"

# usbip state snapshot
dump_env_state
dump_usbip_state

event "preflight_ok"
log "preflight: ok"
