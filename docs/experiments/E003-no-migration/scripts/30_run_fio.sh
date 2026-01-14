#!/usr/bin/env bash
set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/common.sh"

need_cmd fio

event "fio_start" "rw=${FIO_RW};bs=${FIO_BS};qd=${FIO_IODEPTH};jobs=${FIO_NUMJOBS};size=${FIO_SIZE}"

FIO_ARGS=(
  --name="usbip_migration_${LOAD_NAME}"
  --filename="${MOUNTPOINT}/fio_test.bin"
  --rw="${FIO_RW}"
  --bs="${FIO_BS}"
  --direct=1
  --iodepth="${FIO_IODEPTH}"
  --numjobs="${FIO_NUMJOBS}"
  --time_based=1
  --runtime="${RUNTIME_SEC}"
  --group_reporting=1
  --size="${FIO_SIZE}"
  --output-format=json
  --output="${OUT_DIR}/fio.json"
)

if [[ -n "${FIO_RWMIXREAD:-}" ]]; then
  FIO_ARGS+=( --rwmixread="${FIO_RWMIXREAD}" )
fi

# сохраняем команду, чтобы всегда можно было воспроизвести
printf "%s\n" "fio ${FIO_ARGS[*]}" > "${OUT_DIR}/fio.cmd"
log "fio: starting (see fio.cmd)"

set +e
# всё (stdout+stderr) пишем в fio.log
fio "${FIO_ARGS[@]}" 2>&1 | tee "${OUT_DIR}/fio.log"
FIO_RC=${PIPESTATUS[0]}
set -e

echo "${FIO_RC}" > "${OUT_DIR}/fio_exit_code"
event "fio_end" "rc=${FIO_RC}"

exit "${FIO_RC}"

