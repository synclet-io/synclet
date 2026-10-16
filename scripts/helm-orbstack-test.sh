#!/usr/bin/env bash
# Installs Envoy Gateway and the Synclet chart on the orbstack context in throwaway
# namespaces, checks the UI answers through an HTTPRoute, then removes everything,
# cluster-scoped objects included, and fails if the cluster differs from before.
#
# Env:
#   GARAGE=1            also enable the bundled Garage log archive and wait for it
#   EXTRA_VALUES=<file> extra values file for the Synclet release
#   CHART_PROBES=1      keep the chart's /live and /ready probes; by default they call
#                       /health on the public port, which images up to 0.4.0 serve
#   EG_VERSION          Envoy Gateway chart version (default v1.4.2)
set -euo pipefail

CTX=orbstack
EG_VERSION=${EG_VERSION:-v1.4.2}
RUN=sct-$(openssl rand -hex 3)
APP_NS=$RUN
EG_NS=$RUN-eg
GWC=$RUN-eg
GW=$RUN-gateway
HOST=$RUN.local
CHART=$(cd "$(dirname "$0")/../deploy/helm/synclet" && pwd)
WORK=$(mktemp -d)
SNAP_KINDS=customresourcedefinitions,clusterroles,clusterrolebindings,mutatingwebhookconfigurations,validatingwebhookconfigurations,gatewayclasses.gateway.networking.k8s.io,namespaces,persistentvolumes
PF_PID=

k() { kubectl --context "$CTX" "$@"; }
h() { helm --kube-context "$CTX" "$@"; }
log() { printf '\n==> %s\n' "$*"; }
snapshot() { k get "$SNAP_KINDS" -o name 2>/dev/null | sort; }

[[ $(kubectl config get-contexts -o name) == *"$CTX"* ]] || { echo "context $CTX not found" >&2; exit 1; }

cleanup() {
  local rc=$?
  set +e
  [[ -n $PF_PID ]] && kill "$PF_PID" 2>/dev/null
  log "cleanup ($RUN)"
  h uninstall synclet -n "$APP_NS" --wait --timeout 3m >/dev/null 2>&1
  k delete gateway "$GW" -n "$EG_NS" --ignore-not-found --wait --timeout 60s
  k delete gatewayclass "$GWC" --ignore-not-found --wait --timeout 60s ||
    { k patch gatewayclass "$GWC" --type merge -p '{"metadata":{"finalizers":null}}'; k delete gatewayclass "$GWC" --ignore-not-found; }
  h uninstall eg -n "$EG_NS" --wait --timeout 3m >/dev/null 2>&1
  k delete ns "$APP_NS" "$EG_NS" --ignore-not-found --wait=false

  # Helm leaves CRDs and objects the Envoy Gateway controller and certgen created itself.
  snapshot >"$WORK/after"
  comm -13 "$WORK/before" "$WORK/after" | grep -E "$RUN|\.gateway\.envoyproxy\.io$|\.gateway\.networking\.(x-)?k8s\.io$" |
    grep -v '^namespace/' | while read -r obj; do k delete "$obj" --ignore-not-found --wait --timeout 60s; done
  for _ in $(seq 150); do
    snapshot >"$WORK/after"
    cmp -s "$WORK/before" "$WORK/after" && break
    sleep 2
  done
  if cmp -s "$WORK/before" "$WORK/after"; then
    log "cluster matches the pre-test snapshot"
  else
    log "cluster differs from the pre-test snapshot (< removed, > left over):"
    diff "$WORK/before" "$WORK/after"
    rc=1
  fi
  rm -rf "$WORK"
  exit "$rc"
}

snapshot >"$WORK/before"
trap cleanup EXIT

log "Envoy Gateway $EG_VERSION in $EG_NS"
h install eg oci://docker.io/envoyproxy/gateway-helm --version "$EG_VERSION" -n "$EG_NS" --create-namespace --wait --timeout 5m

k create ns "$APP_NS"
k apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: $GWC
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: $GW
  namespace: $EG_NS
spec:
  gatewayClassName: $GWC
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: Selector
          selector:
            matchLabels:
              kubernetes.io/metadata.name: $APP_NS
EOF
k wait gateway "$GW" -n "$EG_NS" --for=condition=Accepted --timeout 3m
for _ in $(seq 60); do
  k get deploy -n "$EG_NS" -l "gateway.envoyproxy.io/owning-gateway-name=$GW" -o name | grep -q . && break
  sleep 2
done
k wait deploy -n "$EG_NS" -l "gateway.envoyproxy.io/owning-gateway-name=$GW" --for=condition=Available --timeout 3m

cat >"$WORK/values.yaml" <<EOF
server:
  ingress:
    enabled: false
  gateway:
    enabled: true
    parentRefs:
      - name: $GW
        namespace: $EG_NS
        sectionName: http
    hostnames:
      - $HOST
auth:
  secureCookies: false
  jwtSecret: "$(openssl rand -base64 32)"
encryption:
  key: "$(openssl rand -base64 32)"
internalApi:
  secret: "$(openssl rand -hex 32)"
postgresql:
  auth:
    password: "$(openssl rand -hex 16)"
  primary:
    persistence:
      size: 1Gi
EOF
args=(-f "$WORK/values.yaml")
[[ ${CHART_PROBES:-} == 1 ]] || for p in startupProbe livenessProbe readinessProbe; do
  args+=(--set "server.$p.path=/health" --set "server.$p.port=public")
done
[[ ${GARAGE:-} == 1 ]] && args+=(--set logArchive.garage.enabled=true)
[[ -n ${EXTRA_VALUES:-} ]] && args+=(-f "$EXTRA_VALUES")

log "Synclet chart in $APP_NS"
h install synclet "$CHART" -n "$APP_NS" "${args[@]}" --wait --timeout 8m
k get pods -n "$APP_NS"

log "route status"
conds=
for _ in $(seq 30); do
  conds=$(k get httproute synclet -n "$APP_NS" -o jsonpath='{range .status.parents[*].conditions[*]}{.type}={.status} {end}')
  [[ $conds == *Accepted=True* && $conds == *ResolvedRefs=True* ]] && break
  sleep 2
done
echo "httproute/synclet: $conds"
[[ $conds == *Accepted=True* && $conds == *ResolvedRefs=True* ]] || { echo "route not accepted" >&2; exit 1; }

log "UI through the Gateway"
svc=$(k get svc -n "$EG_NS" -l "gateway.envoyproxy.io/owning-gateway-name=$GW" -o jsonpath='{.items[0].metadata.name}')
port=$((20000 + RANDOM % 20000))
k port-forward -n "$EG_NS" "svc/$svc" "$port:80" >"$WORK/pf.log" 2>&1 &
PF_PID=$!
code=000
for _ in $(seq 30); do
  code=$(curl -s -o "$WORK/ui.html" -w '%{http_code}' -H "Host: $HOST" "http://127.0.0.1:$port/" || true)
  [[ $code == 200 ]] && break
  sleep 2
done
echo "GET / -> $code"
if [[ $code != 200 ]] || ! grep -q '<title>Synclet</title>' "$WORK/ui.html"; then
  echo "UI did not answer through the route" >&2
  exit 1
fi

log "passed"
