#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/00_env.sh"

: "${DELAY:=0ms}"
: "${JITTER:=0ms}"
: "${LOSS:=0%}"
: "${LIMIT:=1000}"

echo "Applying netem on $IFACE: delay=$DELAY jitter=$JITTER loss=$LOSS limit=$LIMIT" | tee "$OUTDIR/netem_apply.txt"

sudo tc qdisc del dev "$IFACE" root 2>/dev/null || true
sudo tc qdisc add dev "$IFACE" root handle 1: netem delay "$DELAY" "$JITTER" loss "$LOSS" limit "$LIMIT"

sudo tc -s qdisc show dev "$IFACE" | tee "$OUTDIR/tc_qdisc_after.txt"
