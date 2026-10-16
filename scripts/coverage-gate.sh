#!/usr/bin/env bash
# Fails when any package listed in the thresholds file reports statement coverage
# below its minimum. Each non-comment line of the file is "<package> <min%>".
# Coverage is read from a Go cover profile (second argument); without one the
# listed packages are tested to produce it.
set -euo pipefail

cd "$(dirname "$0")/.."

thresholds_file="${1:-scripts/coverage-thresholds.txt}"
profile="${2:-}"

packages=()
while read -r pkg min; do
  [[ -z "$pkg" || "$pkg" == \#* ]] && continue
  packages+=("$pkg")
done < "$thresholds_file"

if [[ ${#packages[@]} -eq 0 ]]; then
  echo "no packages listed in $thresholds_file" >&2
  exit 1
fi

if [[ -z "$profile" ]]; then
  profile="$(mktemp)"
  trap 'rm -f "$profile"' EXIT

  go test -count=1 -covermode=atomic -coverprofile="$profile" "${packages[@]}" >/dev/null || {
    echo "go test failed" >&2
    exit 1
  }
fi

if [[ ! -f "$profile" ]]; then
  echo "cover profile $profile not found" >&2
  exit 1
fi

module="$(go list -m)"

# "<import path> <percent>" per package; a block listed more than once counts as
# covered when any listing has a non-zero count.
coverage="$(awk '
  NR == 1 && /^mode:/ { next }
  {
    split($1, loc, ":")
    file = loc[1]
    pkg = file
    sub(/\/[^\/]*$/, "", pkg)
    if (!($1 in stmts)) { stmts[$1] = $2; block_pkg[$1] = pkg }
    if ($3 > 0) { hit[$1] = 1 }
  }
  END {
    for (b in stmts) {
      total[block_pkg[b]] += stmts[b]
      if (b in hit) { covered[block_pkg[b]] += stmts[b] }
    }
    for (p in total) {
      printf "%s %.1f\n", p, (total[p] ? 100 * covered[p] / total[p] : 0)
    }
  }
' "$profile")"

failed=0
printf '%-72s %8s %8s  %s\n' PACKAGE COVERAGE MIN STATUS

while read -r pkg min; do
  [[ -z "$pkg" || "$pkg" == \#* ]] && continue

  rel="${pkg#./}"
  pct="$(printf '%s\n' "$coverage" | awk -v p="$module/$rel" '$1 == p { print $2 }')"

  if [[ -z "$pct" ]]; then
    printf '%-72s %8s %7s%%  %s\n' "$rel" "n/a" "$min" "NOT IN PROFILE"
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
