#!/usr/bin/env bash
# Fails when any package listed in the thresholds file reports line coverage
# below its minimum. Each non-comment line of the file is "<package> <min%>".
set -euo pipefail

cd "$(dirname "$0")/.."

thresholds_file="${1:-scripts/coverage-thresholds.txt}"

packages=()
while read -r pkg min; do
  [[ -z "$pkg" || "$pkg" == \#* ]] && continue
  packages+=("$pkg")
done < "$thresholds_file"

if [[ ${#packages[@]} -eq 0 ]]; then
  echo "no packages listed in $thresholds_file" >&2
  exit 1
fi

report="$(go test -count=1 -cover "${packages[@]}" 2>&1)" || {
  echo "$report"
  echo "go test failed" >&2
  exit 1
}

failed=0
printf '%-72s %8s %8s  %s\n' PACKAGE COVERAGE MIN STATUS

while read -r pkg min; do
  [[ -z "$pkg" || "$pkg" == \#* ]] && continue

  rel="${pkg#./}"
  line="$(printf '%s\n' "$report" | grep -E "(^|/)${rel}[[:space:]]" | head -1 || true)"
  pct="$(printf '%s\n' "$line" | grep -oE 'coverage: [0-9.]+%' | grep -oE '[0-9.]+' || true)"

  if [[ -z "$pct" ]]; then
    printf '%-72s %8s %7s%%  %s\n' "$rel" "n/a" "$min" "NO COVERAGE LINE"
    failed=1
    continue
  fi

  if awk -v p="$pct" -v m="$min" 'BEGIN { exit !(p + 0 < m + 0) }'; then
    status="BELOW"
    failed=1
  else
    status="ok"
  fi

  printf '%-72s %7s%% %7s%%  %s\n' "$rel" "$pct" "$min" "$status"
done < "$thresholds_file"

exit "$failed"
