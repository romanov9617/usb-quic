#!/usr/bin/env bash
set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"

event "injection_start" "mode=${INJ_MODE};len=${INJ_LEN_SEC}s"

case "${INJ_MODE}" in
link_down)
  log "inject: link down ${CLIENT_IFACE}"
  ip link set dev "${CLIENT_IFACE}" down
  ;;
route_blackhole)
  # Blackhole to usbip host (client side). Needs iproute2.
  log "inject: add blackhole route to ${TARGET_HOST}"
  ip route add blackhole "${TARGET_HOST}" || true
  ;;
vpn_flip)
  log "inject: vpn iface down ${VPN_IFACE}"
  ip link set dev "${VPN_IFACE}" down
  ;;
*)
  die "unknown INJ_MODE=${INJ_MODE}"
  ;;
esac

event "injection_applied" "mode=${INJ_MODE}"
