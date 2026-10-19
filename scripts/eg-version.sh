# shellcheck shell=bash
# Sourced by helm-orbstack-test.sh.

# eg_version reads `helm list -A -o json` on stdin and prints the highest version of a
# gateway-helm release outside the sct-* namespaces a crashed run may have left, or v1.4.2.
eg_version() {
  jq -r '[.[] | select((.chart | startswith("gateway-helm-")) and (.namespace | startswith("sct-") | not))
    | .chart | ltrimstr("gateway-helm-")]
    | sort_by(ltrimstr("v") | split("-")[0] | split(".") | map(tonumber? // 0)) | last // "v1.4.2"'
}
