#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT}/lib/common.sh"

EXP_ENV="${1:?usage: 40_run_one.sh profiles/experiment.env profiles/loads/X.env profiles/injections/Y.env profiles/tcp/Z.env}"
LOAD_ENV="${2:?}"
INJ_ENV="${3:?}"
TCP_ENV="${4:?}"

# Load profiles
source "${ROOT}/${EXP_ENV}"
source "${ROOT}/${LOAD_ENV}"
source "${ROOT}/${INJ_ENV}"
source "${ROOT}/${TCP_ENV}"

RUN_ID="$(date +%Y%m%d_%H%M%S)"
CASE_ID="${LOAD_NAME}__${INJ_NAME}__${TCP_NAME}"
OUT_DIR="${RESULTS_ROOT}/${RUN_ID}/${CASE_ID}"
mkdir_p "${OUT_DIR}"

RUN_LOG="${OUT_DIR}/run.log"
EVENTS_CSV="${OUT_DIR}/events.csv"
echo "ts,event,details" >"${EVENTS_CSV}"

export OUT_DIR RUN_LOG EVENTS_CSV
export MOUNTPOINT DEV CLIENT_IFACE TARGET_HOST
export LOAD_NAME FIO_RW FIO_BS FIO_IODEPTH FIO_NUMJOBS FIO_RWMIXREAD FIO_SIZE
export INJ_NAME INJ_MODE INJ_LEN_SEC VPN_IFACE
export RUNTIME_SEC INJECT_AT_SEC
export TCP_NAME TCP_KEEPALIVE_ENABLE TCP_KEEPIDLE TCP_KEEPINTVL TCP_KEEPCNT

log "run_one: case=${CASE_ID}"
event "run_start" "case=${CASE_ID}"

# Optional sysctl keepalive tuning (best-effort)
if [[ "${TCP_KEEPALIVE_ENABLE:-0}" == "1" ]]; then
  event "tcp_tune_start"
  sysctl -w net.ipv4.tcp_keepalive_time="${TCP_KEEPIDLE:-10}" >/dev/null 2>&1 || true
  sysctl -w net.ipv4.tcp_keepalive_intvl="${TCP_KEEPINTVL:-2}" >/dev/null 2>&1 || true
  sysctl -w net.ipv4.tcp_keepalive_probes="${TCP_KEEPCNT:-3}" >/dev/null 2>&1 || true
  event "tcp_tune_applied" "time=${TCP_KEEPIDLE:-};intvl=${TCP_KEEPINTVL:-};cnt=${TCP_KEEPCNT:-}"
fi

# Preflight
bash "${ROOT}/scripts/00_preflight.sh"

# Collectors start
bash "${ROOT}/scripts/10_collectors_start.sh"

# Start fio in background so we can inject mid-run
event "fio_bg_start"
set +e
bash "${ROOT}/scripts/30_run_fio.sh" &
FIO_PID=$!
set -e
echo "${FIO_PID}" >"${OUT_DIR}/pid_fio"

# Wait until injection moment
sleep "${INJECT_AT_SEC}"
event "inject_wait_done" "at=${INJECT_AT_SEC}s"

# Apply injection, keep it for len, revert
bash "${ROOT}/scripts/20_apply_injection.sh"
sleep "${INJ_LEN_SEC}"
bash "${ROOT}/scripts/21_revert_injection.sh"

# Wait fio to finish
event "fio_wait"
set +e
wait "${FIO_PID}"
FIO_WAIT_RC=$?
set -e
echo "${FIO_WAIT_RC}" > "${OUT_DIR}/fio_wait_rc"
event "fio_wait_done" "rc=${FIO_WAIT_RC}"

# Stop collectors
bash "${ROOT}/scripts/11_collectors_stop.sh"

# Final state snapshots
dump_env_state
dump_usbip_state
dmesg --ctime | tail -n 300 >"${OUT_DIR}/dmesg_tail.log" 2>&1 || true

# Summary.json (минимальный)
cat >"${OUT_DIR}/summary.json" <<EOF
{
  "run_id": "${RUN_ID}",
  "case_id": "${CASE_ID}",
  "profiles": {
    "load": "${LOAD_ENV}",
    "injection": "${INJ_ENV}",
    "tcp": "${TCP_ENV}"
  },
  "params": {
    "runtime_sec": ${RUNTIME_SEC},
    "inject_at_sec": ${INJECT_AT_SEC},
    "inj_len_sec": ${INJ_LEN_SEC}
  },
  "result": {
    "fio_wait_rc": ${FIO_WAIT_RC}
  }
}
EOF

event "run_end" "fio_wait_rc=${FIO_WAIT_RC}"
log "run_one: done out=${OUT_DIR}"
