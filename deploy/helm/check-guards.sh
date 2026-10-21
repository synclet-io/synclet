#!/usr/bin/env bash
# Renders the chart with each guarded value combination and passes only when every
# `helm template` run fails with its guard's message and the default values render.
# Usage: check-guards.sh [chart-dir]
set -uo pipefail

chart="${1:-$(dirname "$0")/synclet}"
failures=0

render() {
  helm template t "$chart" --namespace synclet "$@" 2>&1
}

expect_ok() {
  local name="$1"
  shift
  local out
  if out=$(render "$@"); then
    echo "ok   $name"
  else
    echo "FAIL $name: render failed"
    echo "$out"
    failures=$((failures + 1))
  fi
}

expect_fail() {
  local name="$1" msg="$2"
  shift 2
  local out
  if out=$(render "$@"); then
    echo "FAIL $name: render succeeded, expected: $msg"
    failures=$((failures + 1))
  elif grep -qF -- "$msg" <<<"$out"; then
    echo "ok   $name"
  else
    echo "FAIL $name: expected: $msg"
    echo "$out"
    failures=$((failures + 1))
  fi
}

expect_ok "default values"

expect_fail "executor.strategy after the rename" \
  "executor.strategy was renamed to executor.updateStrategy" \
  --set mode=distributed --set executor.strategy.rollingUpdate.maxSurge=1

expect_fail "ingress and gateway without publicUrl" \
  "server.ingress.enabled and server.gateway.enabled are both true: set server.publicUrl to the URL users reach Synclet at" \
  --set server.ingress.enabled=true --set server.ingress.hostname=synclet.example.com \
  --set server.gateway.enabled=true --set 'server.gateway.parentRefs[0].name=gw' \
  --set 'server.gateway.hostnames[0]=synclet.example.com'

expect_fail "gateway without parentRefs" \
  "server.gateway.enabled needs server.gateway.parentRefs (the Gateway the HTTPRoute attaches to)" \
  --set server.gateway.enabled=true --set 'server.gateway.hostnames[0]=synclet.example.com'

expect_fail "garage with another archive backend" \
  'logArchive.garage.enabled selects the s3 backend and wires it to Garage; leave logArchive.backend at none (got "s3")' \
  --set logArchive.garage.enabled=true --set logArchive.backend=s3 --set logArchive.s3.bucket=logs

expect_fail "s3 without bucket" \
  "logArchive.backend=s3 needs logArchive.s3.bucket" \
  --set logArchive.backend=s3

expect_fail "gcs without bucket" \
  "logArchive.backend=gcs needs logArchive.gcs.bucket" \
  --set logArchive.backend=gcs

expect_fail "filesystem without existingClaim" \
  "logArchive.backend=filesystem needs logArchive.persistence.existingClaim (a ReadWriteMany PVC shared by the server and jobs pods)" \
  --set logArchive.backend=filesystem

expect_fail "vso without appSecrets.path" \
  "vso.appSecrets.path is required when vso.enabled=true" \
  --set vso.enabled=true

expect_fail "vso database without path" \
  "vso.database.path is required when vso.database.enabled=true" \
  --set vso.enabled=true --set vso.appSecrets.path=synclet/app --set vso.database.enabled=true

expect_fail "mcp separate without hostname" \
  "mcp.ingress.hostname is required when mcp.ingress.mode is separate" \
  --set mcp.enabled=true --set mcp.ingress.mode=separate

expect_fail "mcp unknown ingress mode" \
  "mcp.ingress.mode must be shared or separate" \
  --set mcp.enabled=true --set mcp.ingress.mode=public

if [ "$failures" -gt 0 ]; then
  echo "$failures check(s) failed"
  exit 1
fi
echo "all chart guard checks passed"
