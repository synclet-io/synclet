#!/usr/bin/env bash
# Runs eg_version from eg-version.sh on fixture `helm list -A -o json` output.
set -euo pipefail

# shellcheck source-path=SCRIPTDIR source=eg-version.sh
source "$(dirname "$0")/eg-version.sh"

fail() { echo "FAIL: $*" >&2; exit 1; }

check() {
  local name=$1 json=$2 want=$3 got
  got=$(eg_version <<<"$json") || fail "$name: eg_version failed"
  [[ $got == "$want" ]] || fail "$name: want $want, got $got"
}

check "left-over release in an sct namespace" '[
  {"name":"eg","namespace":"sct-old-eg","chart":"gateway-helm-v1.3.0"},
  {"name":"eg","namespace":"envoy-gateway-system","chart":"gateway-helm-v1.6.0"}
]' v1.6.0
check "highest of several" '[
  {"name":"a","namespace":"gw-a","chart":"gateway-helm-v1.10.1"},
  {"name":"b","namespace":"gw-b","chart":"gateway-helm-v1.9.3"},
  {"name":"pg","namespace":"db","chart":"postgresql-15.0.0"}
]' v1.10.1
check "only sct releases" '[{"name":"eg","namespace":"sct-abc123-eg","chart":"gateway-helm-v1.5.0"}]' v1.4.2
check "no releases" '[]' v1.4.2

echo "ok"
