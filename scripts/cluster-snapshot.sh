# shellcheck shell=bash
# Sourced by helm-orbstack-test.sh; expects SNAP_KINDS and a k function that runs kubectl.

# snapshot prints the sorted names of all objects of the kinds in SNAP_KINDS. A kind
# the server does not serve, such as gatewayclasses on a cluster without Gateway API
# CRDs, lists nothing; any other kubectl error fails the snapshot.
snapshot() {
  local kind names err errfile
  errfile=$(mktemp)
  for kind in ${SNAP_KINDS//,/ }; do
    if ! names=$(k get "$kind" -o name 2>"$errfile"); then
      err=$(<"$errfile")
      [[ $err == *"doesn't have a resource type"* ]] && continue
      rm -f "$errfile"
      echo "snapshot of $kind: $err" >&2
      return 1
    fi
    [[ -z $names ]] || printf '%s\n' "$names"
  done | sort
  local rc=${PIPESTATUS[0]}
  rm -f "$errfile"
  return "$rc"
}
