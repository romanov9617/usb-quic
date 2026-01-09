#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-profiles}"

# Accept either a single profile file or a directory with *.env
if [[ -f "$TARGET" ]]; then
  PROFILES=("$TARGET")
elif [[ -d "$TARGET" ]]; then
  mapfile -t PROFILES < <(ls -1 "$TARGET"/*.env 2>/dev/null || true)
else
  echo "ERROR: $TARGET is not a file or directory"
  exit 1
fi

if [[ "${#PROFILES[@]}" -eq 0 ]]; then
  echo "ERROR: no .env profiles found in $TARGET"
  exit 1
fi

bar() {
  # bar <current> <total> [width]
  local cur="$1" total="$2" width="${3:-30}"
  local filled=$((cur * width / total))
  local empty=$((width - filled))
  printf "[%0.s#" $(seq 1 "$filled" 2>/dev/null || true)
  printf "%0.s." $(seq 1 "$empty" 2>/dev/null || true)
  printf "] %d/%d" "$cur" "$total"
}

fmt_time() {
  # fmt_time <seconds>
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

TOTAL="${#PROFILES[@]}"
IDX=0
START_ALL="$(now_epoch)"

# IMPORTANT: must match JOBS list in scripts/20_run_fio.sh
JOBS_PER_PROFILE=4

echo "=== E002 USB/IP netem + fio matrix ==="
echo "Profiles: $TOTAL  | Jobs per profile: $JOBS_PER_PROFILE"
echo

for p in "${PROFILES[@]}"; do
  IDX=$((IDX + 1))
  START_PROFILE="$(now_epoch)"

  echo
  echo "============================================================"
  echo "Profile $(bar "$IDX" "$TOTAL") : $p"
  echo "Started: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "============================================================"

  # Load profile vars for ETA calculation too
  set -a
  source "$p"
  set +a

  # Estimate remaining time based on current profile settings:
  # Each job runs for ~min(RUNTIME, FIO_TIMEOUT) seconds.
  RUNTIME_SEC="${RUNTIME:-15}"
  TIMEOUT_SEC="${FIO_TIMEOUT:-120}"

  # choose a conservative per-job time: min(runtime, timeout)
  if ((RUNTIME_SEC <= TIMEOUT_SEC)); then
    PER_JOB_EST="$RUNTIME_SEC"
  else
    PER_JOB_EST="$TIMEOUT_SEC"
  fi

  # Add small overhead buffer per job (process start, fsync, logging)
  OVERHEAD_PER_JOB=2

  PROFILES_LEFT=$((TOTAL - IDX))
  EST_LEFT=$(((PROFILES_LEFT * JOBS_PER_PROFILE * (PER_JOB_EST + OVERHEAD_PER_JOB))))

  echo "Config: delay=${DELAY:-0ms} loss=${LOSS:-0%} jitter=${JITTER:-0ms} limit=${LIMIT:-1000}"
  echo "fio: runtime=${RUNTIME_SEC}s iodepth=${IODEPTH:-1} numjobs=${NUMJOBS:-1} size=${SIZE:-256m} timeout=${TIMEOUT_SEC}s"
  echo "ETA after this profile: ~$(fmt_time "$EST_LEFT") remaining (heuristic)"
  echo

  export OUTDIR="results/$(date -u +%Y%m%dT%H%M%SZ)_$(basename "$p" .env)"
  mkdir -p "$OUTDIR"
  cp "$p" "$OUTDIR/profile.env"

  ./scripts/10_netem_apply.sh
  ./scripts/05_preflight.sh
  ./scripts/30_collect_sys.sh
  ./scripts/20_run_fio.sh
  ./scripts/30_collect_sys.sh
  ./scripts/11_netem_clear.sh

  END_PROFILE="$(now_epoch)"
  DUR_PROFILE=$((END_PROFILE - START_PROFILE))
  DUR_ALL=$((END_PROFILE - START_ALL))

  echo
  echo "Profile done in $(fmt_time "$DUR_PROFILE"). Total elapsed: $(fmt_time "$DUR_ALL"). Output: $OUTDIR"
done

echo
echo "=== Matrix completed. Total time: $(fmt_time $(($(now_epoch) - START_ALL))) ==="
