#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/00_env.sh"

{
  echo "### date_utc"
  date -u
  echo "### ip_br"
  ip -br a
  echo "### route"
  ip route
  echo "### tc_qdisc"
  sudo tc -s qdisc show dev "$IFACE" || true
  echo "### ss_usbip"
  ss -ti "( dport = :$USBIP_SERVER_PORT or sport = :$USBIP_SERVER_PORT )" || true
  echo "### nstat"
  nstat -az 2>/dev/null || true
} | tee "$OUTDIR/sys_net.txt"
