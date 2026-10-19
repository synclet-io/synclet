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
#   SYNC=1              also build the e2e test connectors (images that run as root),
#                       serve them from a throwaway local registry container and run a
#                       sync through the API; the run fails unless the job completes
#   SYNC_SOURCE=<image:tag>, SYNC_DEST=<image:tag>
#                       with SYNC=1, use this connector image (pulled from its registry)
#                       instead of the test source or destination
#   SYNC_SOURCE_CONFIG=<json>, SYNC_DEST_CONFIG=<json>
#                       their configs (default {})
#   SYNC_STREAM_NAMESPACE  namespace of the synced "users" stream (default public; set it
#                       empty for sources without namespaces)
#   POST_INSTALL=<cmd>  run after the route and UI checks, before cleanup, with CTX and
#                       APP_NS exported; its failure fails the run
set -euo pipefail

CTX=orbstack
EG_VERSION=${EG_VERSION:-v1.4.2}
RUN=sct-$(openssl rand -hex 3)
APP_NS=$RUN
EG_NS=$RUN-eg
GWC=$RUN-eg
GW=$RUN-gateway
HOST=$RUN.local
ROOT=$(cd "$(dirname "$0")/.." && pwd)
CHART=$ROOT/deploy/helm/synclet
REGISTRY=$RUN-registry
REG_PORT=$((30000 + RANDOM % 10000))
REG=localhost:$REG_PORT
WORK=$(mktemp -d)
SNAP_KINDS=customresourcedefinitions,clusterroles,clusterrolebindings,mutatingwebhookconfigurations,validatingwebhookconfigurations,gatewayclasses.gateway.networking.k8s.io,namespaces,persistentvolumes
PF_PID=

k() { kubectl --context "$CTX" "$@"; }
h() { helm --kube-context "$CTX" "$@"; }
log() { printf '\n==> %s\n' "$*"; }
# shellcheck source-path=SCRIPTDIR source=cluster-snapshot.sh
source "$ROOT/scripts/cluster-snapshot.sh"

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
  if [[ ${SYNC:-} == 1 ]]; then
    docker rm -f "$REGISTRY" >/dev/null 2>&1
    docker rmi "$REG/synclet-test-source:$RUN" "$REG/synclet-test-dest:$RUN" >/dev/null 2>&1
  fi
  rm -rf "$WORK"
  exit "$rc"
}

snapshot >"$WORK/before"
trap cleanup EXIT

if [[ ${SYNC:-} == 1 ]]; then
  # Sync pods pull connector images with PullAlways, so they are served from a registry
  # the orbstack node reaches on localhost.
  log "test connectors in registry $REG"
  docker run -d --name "$REGISTRY" -p "127.0.0.1:$REG_PORT:5000" registry:2 >/dev/null
  for _ in $(seq 30); do curl -sf "http://$REG/v2/" >/dev/null && break; sleep 1; done
  for c in source:source destination:dest; do
    image=$REG/synclet-test-${c#*:}:$RUN
    docker build -q -t "$image" -f "$ROOT/tests/e2e/testconnector/${c%%:*}/Dockerfile" "$ROOT/tests/e2e/testconnector" >/dev/null
    docker push -q "$image" >/dev/null
    echo "$image runs as: $(docker inspect -f '{{if .Config.User}}{{.Config.User}}{{else}}root{{end}}' "$image")"
  done
fi

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

api() {
  local hdrs=(-H "Host: $HOST" -H 'Content-Type: application/json')
  [[ -n ${WS_ID:-} ]] && hdrs+=(-H "Workspace-Id: $WS_ID")
  curl -sS --fail-with-body -b "$WORK/cookies" -c "$WORK/cookies" "${hdrs[@]}" \
    "http://127.0.0.1:$port/synclet.publicapi.$1" -d "$2"
}

wait_task() {
  local status=
  for _ in $(seq 100); do
    status=$(api pipeline.v1.ConnectorTaskService/GetConnectorTaskResult "{\"taskId\":\"$1\"}" | jq -r .status)
    case $status in
      CONNECTOR_TASK_STATUS_COMPLETED) return 0 ;;
      CONNECTOR_TASK_STATUS_FAILED) api pipeline.v1.ConnectorTaskService/GetConnectorTaskResult "{\"taskId\":\"$1\"}" >&2; return 1 ;;
    esac
    sleep 3
  done
  echo "connector task $1 still $status" >&2
  return 1
}

if [[ ${SYNC:-} == 1 ]]; then
  log "sync with root-based connectors"
  user_id=$(api auth.v1.AuthService/Register "{\"email\":\"$RUN@test.local\",\"password\":\"$(openssl rand -hex 12)Aa1!\",\"name\":\"Chart test\"}" | jq -r .user.id)
  WS_ID=$(api workspace.v1.WorkspaceService/CreateWorkspace "{\"name\":\"$RUN\",\"ownerUserId\":\"$user_id\"}" | jq -r .workspace.id)
  src_image=$REG/synclet-test-source src_tag=$RUN src_config='{"streams":[{"name":"users","namespace":"public","record_count":100}]}'
  if [[ -n ${SYNC_SOURCE:-} ]]; then
    src_image=${SYNC_SOURCE%:*} src_tag=${SYNC_SOURCE##*:} src_config=${SYNC_SOURCE_CONFIG:-'{}'}
  fi
  src_connector=$(api registry.v1.ConnectorRegistryService/AddConnector "{\"dockerImage\":\"$src_image\",\"dockerTag\":\"$src_tag\",\"name\":\"test source\",\"connectorType\":\"CONNECTOR_TYPE_SOURCE\"}" | jq -r .id)
  dst_image=$REG/synclet-test-dest dst_tag=$RUN dst_config='{"output_dir":"/tmp/output"}'
  if [[ -n ${SYNC_DEST:-} ]]; then
    dst_image=${SYNC_DEST%:*} dst_tag=${SYNC_DEST##*:} dst_config=${SYNC_DEST_CONFIG:-'{}'}
  fi
  dst_connector=$(api registry.v1.ConnectorRegistryService/AddConnector "{\"dockerImage\":\"$dst_image\",\"dockerTag\":\"$dst_tag\",\"name\":\"test destination\",\"connectorType\":\"CONNECTOR_TYPE_DESTINATION\"}" | jq -r .id)
  source_id=$(api pipeline.v1.SourceService/CreateSource "{\"workspaceId\":\"$WS_ID\",\"name\":\"source\",\"managedConnectorId\":\"$src_connector\",\"config\":$src_config}" | jq -r .source.id)
  wait_task "$(api pipeline.v1.SourceService/DiscoverSourceSchema "{\"sourceId\":\"$source_id\"}" | jq -r .taskId)"
  dest_id=$(api pipeline.v1.DestinationService/CreateDestination "{\"workspaceId\":\"$WS_ID\",\"name\":\"destination\",\"managedConnectorId\":\"$dst_connector\",\"config\":$dst_config}" | jq -r .destination.id)
  conn_id=$(api pipeline.v1.ConnectionService/CreateConnection "{\"workspaceId\":\"$WS_ID\",\"name\":\"$RUN\",\"sourceId\":\"$source_id\",\"destinationId\":\"$dest_id\"}" | jq -r .connection.id)
  api pipeline.v1.ConnectionService/ConfigureStreams "{\"connectionId\":\"$conn_id\",\"streams\":[{\"streamName\":\"users\",\"namespace\":\"${SYNC_STREAM_NAMESPACE-public}\",\"syncMode\":\"SYNC_MODE_FULL_REFRESH\",\"destinationSyncMode\":\"DESTINATION_SYNC_MODE_OVERWRITE\"}]}" >/dev/null
  job_id=$(api pipeline.v1.JobService/TriggerSync "{\"connectionId\":\"$conn_id\"}" | jq -r .job.id)
  status=
  for _ in $(seq 100); do
    job=$(api pipeline.v1.JobService/GetJob "{\"jobId\":\"$job_id\"}")
    status=$(jq -r .job.status <<<"$job")
    [[ $status == JOB_STATUS_COMPLETED || $status == JOB_STATUS_FAILED || $status == JOB_STATUS_CANCELLED ]] && break
    sleep 3
  done
  echo "job $job_id: $status"
  [[ $status == JOB_STATUS_COMPLETED ]] || { jq . <<<"$job" >&2; k get pods -n "$APP_NS" >&2; exit 1; }
fi

if [[ -n ${POST_INSTALL:-} ]]; then
  log "post-install: $POST_INSTALL"
  CTX=$CTX APP_NS=$APP_NS $POST_INSTALL
fi

log "passed"
