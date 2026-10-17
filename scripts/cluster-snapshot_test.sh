#!/usr/bin/env bash
# Runs snapshot from cluster-snapshot.sh against a fake kubectl: a cluster
# without Gateway API CRDs, and one whose API server cannot be reached.
set -euo pipefail

# shellcheck source-path=SCRIPTDIR source=cluster-snapshot.sh
source "$(dirname "$0")/cluster-snapshot.sh"

SNAP_KINDS=customresourcedefinitions,gatewayclasses.gateway.networking.k8s.io,namespaces
FAKE_MODE=no-gateway-crds

k() {
  [[ $1 == get && $3 == -o && $4 == name ]] || { echo "unexpected kubectl call: $*" >&2; return 2; }
  if [[ $FAKE_MODE == unreachable ]]; then
    echo 'The connection to the server 127.0.0.1:6443 was refused' >&2
    return 1
  fi
  local kind out=()
  IFS=, read -ra kinds <<<"$2"
  for kind in "${kinds[@]}"; do
    case $kind in
      customresourcedefinitions) out+=(customresourcedefinition.apiextensions.k8s.io/widgets.example.com) ;;
      namespaces) out+=(namespace/default namespace/kube-system) ;;
      gatewayclasses.gateway.networking.k8s.io)
        echo 'error: the server doesn'"'"'t have a resource type "gatewayclasses"' >&2
        return 1
        ;;
    esac
  done
  printf '%s\n' "${out[@]}"
}

fail() { echo "FAIL: $*" >&2; exit 1; }

want='customresourcedefinition.apiextensions.k8s.io/widgets.example.com
namespace/default
namespace/kube-system'
got=$(snapshot) || fail "snapshot failed on a cluster without Gateway API CRDs"
[[ $got == "$want" ]] || fail "snapshot without Gateway API CRDs: want
$want
got
$got"

FAKE_MODE=unreachable
if got=$(snapshot 2>/dev/null); then
  fail "snapshot succeeded with an unreachable API server, output: '$got'"
fi

echo "ok"
