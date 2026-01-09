#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/00_env.sh"

echo "Clearing qdisc on $IFACE" | tee "$OUTDIR/netem_clear.txt"
sudo tc qdisc del dev "$IFACE" root 2>/dev/null || true
sudo tc -s qdisc show dev "$IFACE" | tee "$OUTDIR/tc_qdisc_cleared.txt"
