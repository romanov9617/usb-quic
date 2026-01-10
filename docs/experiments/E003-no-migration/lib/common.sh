#!/usr/bin/env bash
set -euo pipefail

ts_iso() { date -Is; }

log() { echo "[$(ts_iso)] $*" | tee -a "${RUN_LOG:-/dev/null}"; }

die() {
  log "ERROR: $*"
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing dependency: $1"
}

# events.csv: ts,event,details
event() {
  local name="${1:?event name}"
  local details="${2:-}"
  echo "$(ts_iso),${name},${details}" >>"${EVENTS_CSV:?EVENTS_CSV not set}"
}

mkdir_p() { mkdir -p "$1"; }

# best-effort kill background pid
kill_pid() {
  local pid="${1:-}"
  [[ -n "${pid}" ]] || return 0
  if kill -0 "${pid}" 2>/dev/null; then
    kill "${pid}" 2>/dev/null || true
    # give it a moment
    sleep 0.2
    kill -9 "${pid}" 2>/dev/null || true
  fi
}

# Dump usbip state (client side)
dump_usbip_state() {
  usbip port >"${OUT_DIR}/usbip_port.log" 2>&1 || true
}

dump_env_state() {
  {
    echo "=== uname ==="
    uname -a
    echo
    echo "=== ip addr ==="
    ip -br a
    echo
    echo "=== ip route ==="
    ip r
    echo
    echo "=== mount ==="
    mount | grep -E " ${MOUNTPOINT} " || true
    echo
    echo "=== lsusb ==="
    lsusb || true
    echo
    echo "=== usbip port ==="
    usbip port || true
  } >"${OUT_DIR}/env_state.log" 2>&1 || true
}
