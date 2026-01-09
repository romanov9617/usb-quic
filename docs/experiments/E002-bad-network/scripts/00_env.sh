#!/usr/bin/env bash
set -euo pipefail

: "${USBIP_SERVER_IP:?set USBIP_SERVER_IP}"
: "${USBIP_SERVER_PORT:=3240}"
: "${MOUNTPOINT:=/mnt/usb}"

# fio knobs
: "${TESTFILE:=${MOUNTPOINT}/fio_testfile.bin}"
: "${SIZE:=256m}"
: "${RUNTIME:=15}"
: "${IODEPTH:=1}"
: "${NUMJOBS:=1}"
: "${FIO_TIMEOUT:=120}"

# Determine interface used to reach USBIP server
IFACE="${IFACE:-$(ip route get "$USBIP_SERVER_IP" | sed -n 's/.* dev \([^ ]\+\).*/\1/p')}"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUTDIR="${OUTDIR:-results/${STAMP}}"

mkdir -p "$OUTDIR"

echo "OUTDIR=$OUTDIR"
echo "IFACE=$IFACE"
echo "MOUNTPOINT=$MOUNTPOINT"
echo "TESTFILE=$TESTFILE"
echo "SIZE=$SIZE RUNTIME=$RUNTIME IODEPTH=$IODEPTH NUMJOBS=$NUMJOBS FIO_TIMEOUT=$FIO_TIMEOUT"
