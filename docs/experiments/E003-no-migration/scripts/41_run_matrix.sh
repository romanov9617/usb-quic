#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

EXP_ENV="${1:-profiles/experiment.env}"
LOADS_GLOB="${2:-profiles/loads/*.env}"
INJS_GLOB="${3:-profiles/injections/*.env}"
TCPS_GLOB="${4:-profiles/tcp/*.env}"

shopt -s nullglob
LOADS=(${ROOT}/${LOADS_GLOB})
INJS=(${ROOT}/${INJS_GLOB})
TCPS=(${ROOT}/${TCPS_GLOB})

[[ ${#LOADS[@]} -gt 0 ]] || {
  echo "No loads found: ${LOADS_GLOB}"
  exit 1
}
[[ ${#INJS[@]} -gt 0 ]] || {
  echo "No injections found: ${INJS_GLOB}"
  exit 1
}
[[ ${#TCPS[@]} -gt 0 ]] || {
  echo "No tcp profiles found: ${TCPS_GLOB}"
  exit 1
}

echo "Matrix sizes: loads=${#LOADS[@]} injections=${#INJS[@]} tcp=${#TCPS[@]}"
echo "Total runs: $((${#LOADS[@]} * ${#INJS[@]} * ${#TCPS[@]}))"

for l in "${LOADS[@]}"; do
  for i in "${INJS[@]}"; do
    for t in "${TCPS[@]}"; do
      echo "=== RUN: load=$(basename "$l") inj=$(basename "$i") tcp=$(basename "$t") ==="
      sudo -v
      bash "${ROOT}/scripts/40_run_one.sh" "${EXP_ENV}" "${l#${ROOT}/}" "${i#${ROOT}/}" "${t#${ROOT}/}"
      echo
    done
  done
done
