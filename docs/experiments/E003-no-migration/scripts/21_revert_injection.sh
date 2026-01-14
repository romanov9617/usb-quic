#!/usr/bin/env bash
set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"

event "injection_revert_start" "mode=${INJ_MODE}"

case "${INJ_MODE}" in
link_down)
  log "revert: link up ${CLIENT_IFACE}"
  ip link set dev "${CLIENT_IFACE}" up
  ;;
route_blackhole)
  log "revert: del blackhole route to ${TARGET_HOST}"
  ip route del blackhole "${TARGET_HOST}" || true
  ;;
vpn_flip)
  log "revert: vpn iface up ${VPN_IFACE}"
  ip link set dev "${VPN_IFACE}" up
  ;;
*)
  die "unknown INJ_MODE=${INJ_MODE}"
  ;;
esac

event "injection_reverted" "mode=${INJ_MODE}"
