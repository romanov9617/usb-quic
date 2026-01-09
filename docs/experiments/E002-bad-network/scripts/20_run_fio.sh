#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/00_env.sh"

mkdir -p "$OUTDIR/fio"

# Fixed, safe order (warm-up + then random)
JOBS=(fio/seq_write.fio fio/seq_read.fio fio/rand_read_4k.fio fio/rand_write_4k.fio)
TOTAL_JOBS="${#JOBS[@]}"
JOB_IDX=0

job_bar() {
  local cur="$1" total="$2" width="${3:-20}"
  local filled=$((cur * width / total))
  local empty=$((width - filled))
  printf "[%0.s#" $(seq 1 "$filled" 2>/dev/null || true)
  printf "%0.s." $(seq 1 "$empty" 2>/dev/null || true)
  printf "] %d/%d" "$cur" "$total"
}

fmt_time() {
  local s="${1:-0}"
  if ((s < 0)); then s=0; fi
  local h=$((s / 3600))
  local m=$(((s % 3600) / 60))
  local sec=$((s % 60))
  if ((h > 0)); then
    printf "%dh%02dm%02ds" "$h" "$m" "$sec"
  else
    printf "%dm%02ds" "$m" "$sec"
  fi
}

now_epoch() { date +%s; }

{
  echo "testfile=$TESTFILE"
  echo "size=$SIZE runtime=$RUNTIME iodepth=$IODEPTH numjobs=$NUMJOBS timeout=$FIO_TIMEOUT"
  echo "jobs=${JOBS[*]}"
} | tee "$OUTDIR/fio/meta.txt"

START_ALL="$(now_epoch)"

for job in "${JOBS[@]}"; do
  JOB_IDX=$((JOB_IDX + 1))
  name="$(basename "$job" .fio)"
  out_json="$OUTDIR/fio/${name}.json"

  echo
  echo "------------------------------------------------------------"
  echo "FIO job $(job_bar "$JOB_IDX" "$TOTAL_JOBS") : $name"
  echo "Params: runtime=${RUNTIME}s iodepth=${IODEPTH} numjobs=${NUMJOBS} size=${SIZE} timeout=${FIO_TIMEOUT}s"
  echo "Output: $out_json"
  echo "------------------------------------------------------------"

  echo "cmd: sudo fio $job --filename=$TESTFILE --size=$SIZE --runtime=$RUNTIME --iodepth=$IODEPTH --numjobs=$NUMJOBS" |
    tee "$OUTDIR/fio/${name}.cmd.txt"

  JOB_START="$(now_epoch)"

  # status-interval prints live progress each second
  # timeout prevents endless waits (won't always break D-state, but helps in most hangs)
  if ! timeout --signal=INT "${FIO_TIMEOUT}s" sudo fio "$job" \
    --filename="$TESTFILE" \
    --size="$SIZE" \
    --runtime="$RUNTIME" \
    --iodepth="$IODEPTH" \
    --numjobs="$NUMJOBS" \
    --status-interval=1 \
    --output-format=json \
    --output="$out_json"; then
    echo "fio_failed_or_timed_out: $name" | tee -a "$OUTDIR/fio/errors.txt"
    ps -eo pid,stat,wchan:40,cmd | grep -E 'fio|usbip|vhci' | grep -v grep |
      tee "$OUTDIR/fio/${name}.ps.txt" || true
    dmesg -T | tail -n 120 | tee "$OUTDIR/fio/${name}.dmesg_tail.txt" || true
    exit 1
  fi

  JOB_END="$(now_epoch)"
  JOB_DUR=$((JOB_END - JOB_START))
  ALL_DUR=$((JOB_END - START_ALL))

  # Estimated remaining time within this profile:
  # conservative: each remaining job might take min(runtime, timeout)
  if ((RUNTIME <= FIO_TIMEOUT)); then
    PER_JOB_EST="$RUNTIME"
  else
    PER_JOB_EST="$FIO_TIMEOUT"
  fi
  OVERHEAD_PER_JOB=2
  JOBS_LEFT=$((TOTAL_JOBS - JOB_IDX))
  EST_LEFT=$((JOBS_LEFT * (PER_JOB_EST + OVERHEAD_PER_JOB)))

  echo "Job finished in $(fmt_time "$JOB_DUR"). Elapsed in this profile: $(fmt_time "$ALL_DUR")."
  echo "ETA within this profile: ~$(fmt_time "$EST_LEFT") remaining (heuristic)."
done
