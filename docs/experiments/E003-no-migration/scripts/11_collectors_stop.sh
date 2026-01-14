#!/usr/bin/env bash
set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"

event "collectors_stop"

kill_pid "$(cat "${OUT_DIR}/pid_dmesg_follow" 2>/dev/null || true)"
kill_pid "$(cat "${OUT_DIR}/pid_iostat" 2>/dev/null || true)"
kill_pid "$(cat "${OUT_DIR}/pid_vmstat" 2>/dev/null || true)"
kill_pid "$(cat "${OUT_DIR}/pid_net" 2>/dev/null || true)"

event "collectors_stopped"
