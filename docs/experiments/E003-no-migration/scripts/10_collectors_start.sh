#!/usr/bin/env bash
set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"

need_cmd dmesg
need_cmd iostat || true
need_cmd vmstat || true

event "collectors_start"

# dmesg follow
(dmesg --follow --ctime) >"${OUT_DIR}/dmesg_follow.log" 2>&1 &
echo $! >"${OUT_DIR}/pid_dmesg_follow"

# iostat (если есть)
if command -v iostat >/dev/null 2>&1; then
  (iostat -x 1) >"${OUT_DIR}/iostat.log" 2>&1 &
  echo $! >"${OUT_DIR}/pid_iostat"
fi

# vmstat (если есть)
if command -v vmstat >/dev/null 2>&1; then
  (vmstat 1) >"${OUT_DIR}/vmstat.log" 2>&1 &
  echo $! >"${OUT_DIR}/pid_vmstat"
fi

# net counters snapshot loop
(
  while true; do
    ts="$(ts_iso)"
    ip -s link show dev "${CLIENT_IFACE}" | sed "s/^/${ts} /"
    sleep 1
  done
) >"${OUT_DIR}/net_link_counters.log" 2>&1 &
echo $! >"${OUT_DIR}/pid_net"

event "collectors_started"
