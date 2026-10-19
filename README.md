<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/assets/synclet-logo.svg">
    <img src=".github/assets/synclet-logo-light.svg" alt="Synclet" width="450">
  </picture>
</p>

# Synclet

Run Airbyte connectors without Airbyte: one Go binary, one PostgreSQL. Synclet is a self-hosted control plane for connectors that implement the Airbyte Protocol, with schema discovery, incremental sync and monitoring.

<p align="center">
  <img src=".github/assets/demo-first-sync.gif" alt="Creating a connection from a Faker source to PostgreSQL in Synclet and running its first sync" width="880">
</p>

## Features

- **Airbyte Protocol connectors** -- Runs source and destination images that implement the Airbyte Protocol; see [Tested with](#tested-with) for the pairs and versions tested
- **Multiple Execution Modes** -- Docker, Kubernetes, or CLI
- **Stream Configuration** -- Field selection, namespace rewriting, primary key overrides
- **Secrets Management** -- AES-256-GCM encrypted configuration storage
- **OIDC Authentication** -- SSO with any OpenID Connect provider
- **Workspace Isolation** -- Multi-tenant with role-based access control (admin/editor/viewer)
- **Notification Channels** -- Slack, Email, Telegram alerts on sync events
- **Dashboard & Stats** -- Connection health, sync timeline, records/bytes metrics
- **Config Import/Export** -- YAML-based workspace backup and restore
- **Dark Mode** -- System-aware theme switching
- **Synclet connectors** -- MIT source and destination connectors written in Go on synclet-go-sdk, in [synclet-io/connectors](https://github.com/synclet-io/connectors); see [Writing your own connector](#writing-your-own-connector)

## Screenshots

The images come from the demo workspace that [`scripts/demo-seed`](scripts/demo-seed/README.md) creates on mock data.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/dashboard-dark.png">
  <img src=".github/assets/screenshots/dashboard-light.png" alt="Dashboard with sync totals, success rate, timeline and throughput">
</picture>

| | |
|---|---|
| <picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/connection-dark.png"><img src=".github/assets/screenshots/connection-light.png" alt="Connection page with status, schedule and replication stats"></picture> | <picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/topology-dark.png"><img src=".github/assets/screenshots/topology-light.png" alt="Topology of sources, connections and destinations coloured by health"></picture> |
| <picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/job-logs-dark.png"><img src=".github/assets/screenshots/job-logs-light.png" alt="Job page with its timeline and the combined connector logs"></picture> | <picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/audit-log-dark.png"><img src=".github/assets/screenshots/audit-log-light.png" alt="Audit log of connection changes with a before and after diff"></picture> |

## Architecture

Synclet is a **modular monolith** -- a single Go binary runs the API server, scheduler, and workers as goroutines. Each domain module (auth, workspace, pipeline, notify, secret, audit) is a bounded context with its own storage, service layer, and ConnectRPC handlers; `app` serves the frontend.

- **Backend**: Go 1.26, ConnectRPC (protobuf), PostgreSQL, GORM, Uber FX
- **Frontend**: Vue 3, TypeScript, TanStack Vue Query, Tailwind CSS
- **Connectors**: Docker containers running Airbyte protocol connectors
- **Orchestration**: Docker (default) or Kubernetes for distributed execution

## Quick Start

With Docker and Docker Compose installed:

```bash
git clone https://github.com/synclet-io/synclet && cd synclet && docker compose up -d
```

Open `http://localhost:8080` a minute later. The stack runs PostgreSQL, migrations and a standalone server that starts connectors through the host Docker daemon. Synclet (8080) and PostgreSQL (5437) listen on `127.0.0.1` only.

The secrets come from the committed `.env.compose`, whose values are public. Before Synclet is reachable from anywhere but your machine, put your own in an untracked `.env` next to `docker-compose.yml`, which overrides `.env.compose`, and keep a copy of `SECRET_ENCRYPTION_KEY`, since stored connector secrets cannot be decrypted without it:
```bash
echo "AUTH_JWT_SECRET=$(openssl rand -base64 32)" >> .env
echo "SECRET_ENCRYPTION_KEY=$(openssl rand -base64 32)" >> .env
docker compose up -d
```

To expose Synclet deliberately, change `127.0.0.1:8080:8080` under `synclet.ports` in `docker-compose.yml` to `8080:8080`, or put a TLS reverse proxy in front of it. Anyone who can reach it can start containers on the host through the mounted Docker socket.

### Development Setup

Building from source needs Go 1.26+, PostgreSQL 16+, Docker and Bun (for the frontend).

1. Clone the repository:
   ```bash
   git clone https://github.com/synclet-io/synclet && cd synclet
   ```

2. Configure environment:
   ```bash
   cp .env.dist .env
   go run . generate dotenv   # optional: append every other variable with its default
   ```

3. Start PostgreSQL and run database migrations:
   ```bash
   task db:up
   go run . migrate up
   ```

   `.env.dist` sends email to Mailpit. Start it to receive invites and password reset emails at `http://localhost:8025`:
   ```bash
   docker compose -f docker-compose.dev.yml up -d mailpit
   ```

4. Start backend (with hot reload):
   ```bash
   task dev
   ```

5. Start frontend:
   ```bash
   cd front && bun install && bun run dev
   ```

The API server runs on `http://localhost:8086` and the frontend dev server on `http://localhost:5173`.

### Production Deployment

Outside `ENVIRONMENT=development` the process refuses to start without `AUTH_JWT_SECRET` and `SECRET_ENCRYPTION_KEY`; generate each with `openssl rand -base64 32` and back up the encryption key, since stored connector secrets cannot be decrypted without it.

**Binary.** Each [GitHub release](https://github.com/synclet-io/synclet/releases) tagged `synclet-v<version>` carries `synclet-v<version>-<os>-<arch>.tar.gz` for `linux/amd64`, `linux/arm64`, `darwin/amd64` and `darwin/arm64`, plus `SHA256SUMS` over every archive. The binary serves the frontend, so it needs only PostgreSQL and Docker:
```bash
sha256sum -c SHA256SUMS --ignore-missing            # Linux
shasum -a 256 -c SHA256SUMS --ignore-missing        # macOS
tar -xzf synclet-v<version>-linux-amd64.tar.gz
cd synclet-v<version>-linux-amd64
./synclet migrate up
./synclet server --standalone
```

**Docker Compose:** `docker compose up -d` as in [Quick Start](#quick-start), with your own secrets in `.env`. The `synclet` service runs as root so it can reach the Docker socket and write the connector scratch directory `/tmp/synclet`, which Docker creates on the host.

**Docker:**
```bash
docker build -t synclet .
docker run --rm --env-file .env synclet migrate up
docker run -p 8080:8080 --env-file .env -v /var/run/docker.sock:/var/run/docker.sock synclet server --standalone
```

`server` alone serves the API only; `--standalone` also runs the scheduler, event consumers and an executor in the same process. Without it, run `synclet jobs` and `synclet docker-executor` (or `k8s-executor`) as separate processes.

**Helm (Kubernetes):**
```bash
helm install synclet deploy/helm/synclet/ -f values.yaml
```

#### Exposing the server: Ingress or Gateway API (Helm)

By default the chart creates an `Ingress` (`server.ingress.*`, class `nginx`). On clusters that use the Kubernetes Gateway API (Envoy Gateway, GKE Gateway, Istio, Cilium and others) it can create an `HTTPRoute` (`gateway.networking.k8s.io/v1`) attached to a `Gateway` you already run. Both route only to the public server Service; the internal executor API (`<fullname>-internal`) is never exposed.

| Value | Description | Default |
|-------|-------------|---------|
| `server.gateway.enabled` | Render the `HTTPRoute` | `false` |
| `server.gateway.parentRefs` | Gateways the route attaches to: `name` (required), `namespace`, `sectionName` (listener), `port` | `[]` |
| `server.gateway.hostnames` | Hostnames the route matches; empty matches every hostname of the listener | `[]` |
| `server.gateway.paths` | Path matches (`type`, `value`) | `PathPrefix /` |
| `server.gateway.tls` | The Gateway listener terminates TLS; makes the derived public URL `https` | `false` |
| `server.gateway.labels`, `server.gateway.annotations` | Extra metadata on the route | `{}` |
| `server.publicUrl` | URL users reach Synclet at; default for `frontend.url` (invite links) and `oidc.callbackBaseURL` | empty |

When `server.publicUrl` is empty it is derived from `server.ingress.hostname` if only the Ingress is enabled, or from the first `server.gateway.hostnames` entry if only the route is. With both enabled the render fails until `server.publicUrl` is set. `frontend.url` and `oidc.callbackBaseURL` still override it for their own setting.

A minimal `Gateway`, terminating TLS with a certificate Secret in its own namespace:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: shared-gateway
  namespace: gateway-system
spec:
  gatewayClassName: eg               # your implementation's GatewayClass
  listeners:
    - name: https
      protocol: HTTPS
      port: 443
      hostname: synclet.example.com
      tls:
        mode: Terminate
        certificateRefs:
          - name: synclet-example-com-tls
      allowedRoutes:
        namespaces:
          from: Selector
          selector:
            matchLabels:
              kubernetes.io/metadata.name: synclet   # the release namespace
```

```yaml
server:
  ingress:
    enabled: false
  gateway:
    enabled: true
    parentRefs:
      - name: shared-gateway
        namespace: gateway-system
        sectionName: https
    hostnames:
      - synclet.example.com
    tls: true
```

- **Cross-namespace attachment.** A listener accepts routes only from its own namespace unless `allowedRoutes.namespaces` says otherwise (`Same` by default, or `All`, or `Selector` as above). If the route stays `Accepted=False` with `NotAllowedByListeners`, that is the cause. Check with `kubectl get httproute <fullname> -o yaml`.
- **ReferenceGrant.** The route and its backend, the public Service, are in the release namespace, so attaching the route needs none. You need a `ReferenceGrant` in the Secret's namespace only when a listener's `certificateRefs` points at a Secret in a namespace other than the Gateway's (for example a certificate kept next to Synclet).
- **TLS.** TLS ends at the Gateway (`mode: Terminate`). The chart puts no certificate on the route, and cert-manager annotations go on the `Gateway`, not on the chart. Set `server.gateway.tls: true` (or `server.publicUrl` with `https://`) so invite links and OIDC callbacks use `https`. Keep `auth.secureCookies: true`.
- **Client IPs.** Login and register rate limits trust `X-Forwarded-For` only from `rateLimit.trustedProxies`. Behind a Gateway, set it to the pod CIDR of the data-plane proxies (for Envoy Gateway, the `envoy-*` pods in the controller namespace), not to the ingress controller's.

Removing a Gateway implementation: `helm uninstall` of the implementation does not remove everything it put in the cluster. Envoy Gateway, for example, leaves the `eg-gateway-helm-certgen:<namespace>` ClusterRole and ClusterRoleBinding, the `envoy-gateway-topology-injector.<namespace>` MutatingWebhookConfiguration and its CRDs, and a `GatewayClass` whose controller is gone keeps its `gateway-exists-finalizer.gateway.networking.k8s.io` finalizer and never finishes deleting. Remove them in this order:

```bash
kubectl delete gateway --all -n <gateway-namespace>      # before the controller, so it releases the GatewayClass
kubectl delete gatewayclass <class>
helm uninstall eg -n <gateway-namespace>
kubectl delete clusterrole,clusterrolebinding eg-gateway-helm-certgen:<gateway-namespace>
kubectl delete mutatingwebhookconfiguration envoy-gateway-topology-injector.<gateway-namespace>
kubectl delete namespace <gateway-namespace>
# A GatewayClass stuck in Terminating after the controller is gone:
kubectl patch gatewayclass <class> --type merge -p '{"metadata":{"finalizers":null}}'
# CRDs, only when nothing else in the cluster uses them (Gateway API CRDs are shared):
kubectl get crd -o name | grep -E '\.gateway\.envoyproxy\.io$' | xargs kubectl delete
```

`task helm:test:orbstack` runs this cycle on the `orbstack` context: it installs Envoy Gateway and the chart in throwaway namespaces, checks that the UI answers through the route, removes everything, and fails if CRDs, cluster roles and bindings, webhooks, GatewayClasses, namespaces or PersistentVolumes differ from before the run. `GARAGE=1` adds the bundled Garage archive, `EXTRA_VALUES=<file>` passes more chart values, and `CHART_PROBES=1` keeps the chart's `/live` and `/ready` probes (by default the test points them at `/health` on the public port, which images up to 0.4.0 serve).

Migrating from Ingress to a Gateway without downtime:

1. Enable the route alongside the Ingress and set `server.publicUrl` to the URL users will end up on. Both objects route to the same Service, either with the old hostname on the Ingress and a new one on the route, or with the same hostname on both:
   ```yaml
   server:
     publicUrl: https://synclet.example.com
     ingress:
       enabled: true
       hostname: synclet.example.com
     gateway:
       enabled: true
       parentRefs: [{name: shared-gateway, namespace: gateway-system, sectionName: https}]
       hostnames: [synclet.example.com]
       tls: true
   ```
2. Check the route through the Gateway's address before any DNS change, e.g. `curl --resolve synclet.example.com:443:<gateway-ip> https://synclet.example.com/`. Make sure the Gateway serves a valid certificate for the hostname.
3. Switch DNS to the Gateway's address and wait for the old TTL to run out.
4. Disable the Ingress (`server.ingress.enabled: false`). `server.publicUrl` can stay or be removed, since the route's first hostname now gives the same URL.

#### Server probes (Helm)

The server's startup, liveness and readiness probes call `/live`, `/live` and `/ready` on the `internal` container port. Each probe takes `path` and `port` (`server.startupProbe.*`, `server.livenessProbe.*`, `server.readinessProbe.*`); `port` is a container port name (`public`, `internal`) or a number.

Older images serve only `/health` on the public port. Point the probes there instead of disabling them:

```yaml
server:
  startupProbe:
    path: /health
    port: public
  livenessProbe:
    path: /health
    port: public
  readinessProbe:
    path: /health
    port: public
```

## Configuration Reference

### Environment Variables

`synclet generate dotenv` appends every variable the binary reads, with its default, to `.env` (or the first `--dotenv` file). The most common ones:

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | `development` allows generated secrets and `migrate down` without `--yes` | `development` |
| `LOG_LEVEL` | Log level | `info` |
| `DB_DSN` | PostgreSQL connection string | required |
| `AUTH_JWT_SECRET` | Secret for signing JWT tokens | required outside development |
| `SECRET_ENCRYPTION_KEY` | Base64-encoded 32-byte AES key for connector secrets | required outside development |
| `SECRET_ENCRYPTION_KEY_PREVIOUS` | Previous key, kept during key rotation | empty |
| `PUBLIC_HTTP_SERVER_ADDR` | Public API and UI listen address | `0.0.0.0:8080` |
| `INTERNAL_HTTP_SERVER_ADDR` | Executor API and the `/live`, `/ready` probes | `0.0.0.0:8081` |
| `PUBLIC_HTTP_SERVER_CORS_ALLOWED_ORIGINS` | Comma-separated hostnames whose browser origins may call the API | empty |
| `AUTH_REGISTRATION_ENABLED` | Allow email/password sign-up | `true` |
| `AUTH_SECURE_COOKIES` | Mark auth cookies HTTPS-only | `false` |
| `AUTH_PASSWORD_RESET_COOLDOWN` | Minimum time between two password reset emails for one user | `2m` |
| `AUTH_OIDC_PROVIDERS` | Comma-separated OIDC provider slugs, each configured by `AUTH_OIDC_<SLUG>_*` | empty |
| `AUTH_OIDC_CALLBACK_BASE_URL` | Base URL for OIDC callbacks | empty |
| `WORKSPACE_MODE` | `single` shared workspace or `multi` | `single` |
| `WORKSPACE_FRONTEND_URL` | Base URL used in invite and password reset links | `http://localhost:5173` |
| `NOTIFY_SMTP_HOST` | SMTP server; empty disables email | empty |
| `NOTIFY_SMTP_PORT` | SMTP port | `587` |
| `NOTIFY_SMTP_TLS` | `implicit` (TLS from the first byte), `starttls` or `none` (plain text, for local catchers such as Mailpit) | `implicit` |
| `NOTIFY_SMTP_USER` | SMTP username | empty |
| `NOTIFY_SMTP_PASSWORD` | SMTP password | empty |
| `NOTIFY_SMTP_FROM` | Sender address | `noreply@synclet.io` |
| `PIPELINE_MAX_CONCURRENT_JOBS` | Concurrent sync jobs | `10` |
| `PIPELINE_IDLE_TIMEOUT` | Cancel a sync that receives no records for this long | `10m` |
| `PIPELINE_JOB_SCHEDULER_INTERVAL` | How often due connections are scheduled | `30s` |
| `PIPELINE_LOG_MAX_LINES_PER_ATTEMPT` | Stored log lines per job attempt; `0` disables the cap | `100000` |
| `PIPELINE_LOG_ARCHIVE_BACKEND` | Where logs of finished jobs go: `none`, `filesystem`, `s3` or `gcs` | `none` |
| `PIPELINE_LOG_ARCHIVE_DIR` | Base directory for the `filesystem` backend | empty |
| `PIPELINE_LOG_ARCHIVE_S3_*` | `ENDPOINT`, `REGION`, `BUCKET`, `PREFIX`, `ACCESS_KEY_ID`, `SECRET_ACCESS_KEY`, `USE_PATH_STYLE` for the `s3` backend | empty |
| `PIPELINE_LOG_ARCHIVE_GCS_*` | `BUCKET`, `PREFIX`, `CREDENTIALS_FILE` for the `gcs` backend | empty |
| `DOCKER_EXECUTOR_MAX_SYNC_DURATION` | Maximum sync run time on the Docker executor | `24h` |
| `DOCKER_EXECUTOR_TEMP_DIR_ROOT` | Scratch directory shared with the host Docker daemon | empty |
| `DOCKER_EXECUTOR_LOCAL_ROOT` | Host directory mounted at `/local` in every connector container, where file-writing destinations leave their files; shared by all jobs and never cleaned (see [Files written by destinations](#files-written-by-destinations)). Empty gives each job its own `/local`, removed with the job. The compose file sets `/tmp/synclet_local` | empty |
| `DOCKER_EXECUTOR_OWNER_ID` | Owner label that scopes orphan container cleanup to this executor. Executors sharing one Docker daemon need distinct values; two standalone servers on one machine must set it explicitly. A containerised executor needs a stable hostname (the compose file sets `hostname: synclet`) or an explicit value, since a container's default hostname changes on recreation. Containers without the label (from earlier versions) are left alone; remove them by hand once | hostname |
| `K8S_EXECUTOR_NAMESPACE` | Namespace for connector pods | empty |
| `K8S_EXECUTOR_LOCAL_VOLUME_CLAIM` | Existing PersistentVolumeClaim mounted at `/local` in the connector containers (Helm: `k8s.localVolume`). Empty mounts a per-pod `emptyDir` | empty |
| `K8S_EXECUTOR_OWNER_ID` | Owner label on the Jobs and Secrets the Kubernetes executor creates; its reconciler handles only those carrying it, so executors sharing one namespace need distinct values. The default is the pod name: the Helm chart runs the executor as a StatefulSet (stable names `<release>-executor-0`, `-1`, ...), so a restarted replica reclaims its own leftovers, and in standalone mode, where the server runs it in a Deployment, the chart sets the release fullname (`k8s.ownerId` overrides both). Jobs without the label (from earlier versions) are left alone; remove them by hand once | hostname |
| `EXECUTOR_API_URL` | Internal server URL used by distributed executors | empty |
| `EXECUTOR_API_TOKEN` | Token distributed executors present to the internal server | empty |
| `WATERMILL_TRANSPORT` | Event bus transport | `sql` |

### Authentication

Users sign in with email and password, or with an OIDC provider when `AUTH_OIDC_PROVIDERS` is set.

**Password reset.** With SMTP configured (`NOTIFY_SMTP_HOST`), the login page shows a "Forgot password?" link. The user enters their email and receives a reset link that expires in 1 hour and works once; requesting another link invalidates the previous one. Requests for the same account within `AUTH_PASSWORD_RESET_COOLDOWN` (2 minutes by default) send nothing. Resetting the password signs the user out of every session. An account that only signed in through OIDC can set a password the same way.

To check the email path locally, start Mailpit (see Development Setup) with the SMTP settings from `.env.dist`, click "Forgot password?" on the login page and request a link for your account. The email shows up at `http://localhost:8025`; its link opens the reset form on `WORKSPACE_FRONTEND_URL`, where the new password replaces the old one.

**Without SMTP** the "Forgot password?" link is hidden. The operator who runs the Synclet deployment issues the link instead and hands it to the user:

```bash
synclet users reset-link --email jane@example.com
```

The command connects to the database (like `migrate`, no running server needed), prints the reset URL to stdout and exits non-zero when no user has that email. The link has the same 1-hour, single-use limits; the cooldown does not apply to this command.

### Job Log Storage

While a job runs, its log lines are stored in Postgres and the UI tails them from there. Each attempt keeps at most `PIPELINE_LOG_MAX_LINES_PER_ATTEMPT` lines; later lines are dropped and counted in a single `[synclet] log truncated: N further lines dropped` line.

With `PIPELINE_LOG_ARCHIVE_BACKEND=none` (the default) logs stay in Postgres until job retention deletes the job. With `filesystem`, `s3` or `gcs`, a finished job's logs are moved out of Postgres into one gzipped NDJSON object per attempt, keyed `logs/<workspace>/<connection>/<job>/<attempt>.ndjson.gz`, and read back from there; retention and workspace deletion remove the objects too.

- `filesystem` writes under `PIPELINE_LOG_ARCHIVE_DIR`. Every server replica must see the same directory, so use a shared volume.
- `s3` works with AWS S3, MinIO, Cloudflare R2 and Google Cloud Storage (HMAC keys via its S3 interoperability endpoint). Set `PIPELINE_LOG_ARCHIVE_S3_ENDPOINT` for anything but AWS, and `PIPELINE_LOG_ARCHIVE_S3_USE_PATH_STYLE=true` for MinIO. Without static keys the default AWS credential chain is used (environment, IRSA, instance profile).
- `gcs` uses the native Google Cloud Storage API. With `PIPELINE_LOG_ARCHIVE_GCS_CREDENTIALS_FILE` it authenticates with that service account key; without it, Application Default Credentials (GKE Workload Identity, `GOOGLE_APPLICATION_CREDENTIALS`, or `gcloud auth application-default login` locally). The server logs which credential source it uses at startup. Credentials are resolved on first use, so missing ADC shows up as archive errors, not a startup failure.

Archiving health is exported on the metrics endpoint: `synclet_log_archive_failures_total{backend}` counts job attempts whose logs failed to archive, `synclet_log_archive_bytes_total` the size of the objects written, and `synclet_log_lines_dropped_total` the lines dropped by `PIPELINE_LOG_MAX_LINES_PER_ATTEMPT`.

#### GCS with GKE Workload Identity (Helm)

No key file is needed. The server and jobs pods run as the chart's Kubernetes service account; bind it to a Google service account that can write the bucket:

```bash
gcloud storage buckets add-iam-policy-binding gs://<bucket> \
  --member "serviceAccount:<gsa>@<project>.iam.gserviceaccount.com" \
  --role roles/storage.objectAdmin

gcloud iam service-accounts add-iam-policy-binding <gsa>@<project>.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:<project>.svc.id.goog[<namespace>/<ksa>]"
```

`<ksa>` is the chart's service account (`<release>-synclet` unless `serviceAccount.name` is set). Then:

```yaml
serviceAccount:
  annotations:
    iam.gke.io/gcp-service-account: <gsa>@<project>.iam.gserviceaccount.com
logArchive:
  backend: gcs
  gcs:
    bucket: <bucket>
```

Outside Workload Identity, store a service account key in a Secret and set `logArchive.gcs.credentialsSecret` (key `key.json` by default); it is mounted only into the server and jobs pods.

The executor deployment (distributed mode) and connector pods run under accounts of their own, so they get no bucket access; see below.

#### Service accounts (Helm)

The chart's `serviceAccount` is meant for the server and jobs pods, which archive job logs and so may hold bucket permissions through Workload Identity. Two other workloads run under accounts of their own:

- **Executor** (`executor.serviceAccount`, distributed mode). The executor launches connector pods, so it needs the connector-runner Role, but it never touches the log archive: it sends log lines to the server through the executor API and only the server archives them. With `create: true` (the default) the chart renders `<release>-synclet-executor` and binds it to the connector-runner Role and nothing else. Set `create: false` with a `name` to use an existing account (the chart still binds it), or leave both unset to keep the executor on the chart's account.
- **Connector pods** (`pipeline.serviceAccount`). Every pod the executor creates (sync, check, spec, discover) runs in `k8s.namespace` under the account in `PIPELINE_RUNTIME_SERVICE_ACCOUNT_NAME`, or that namespace's default account when it is empty. With `create: true` the chart renders `<release>-synclet-connector` in `k8s.namespace` with the given annotations and sets that variable to its name. A source or destination overrides it with `service_account_name` in its runtime config; a sync pod takes the source's override, else the destination's, else the global default. Check, spec and discover pods use the global default. The account gets no RBAC.

Connector images are third-party code, so a connector pod gets no Kubernetes API token unless `pipeline.serviceAccount` names or creates an account for it, `pipeline.resources.serviceAccountName` names an existing one, or a runtime config sets `service_account_name`: a pod without an account name runs with `automountServiceAccountToken: false`, and its coordinator container still reaches the control plane through `EXECUTOR_API_URL` with the per-job token. A connector account exists to carry cloud identity annotations (IRSA on EKS, Workload Identity on GKE), not cluster rights; a pod running under one keeps the cluster's default token mount, so do not bind it to any Role. The chart itself renders no ClusterRole or ClusterRoleBinding: its only Role is the namespaced connector-runner Role in `k8s.namespace`.

Both accounts take `annotations`, which is how Workload Identity binds a cloud identity to a Kubernetes account. Some connectors reach Google Cloud or AWS that way:

```yaml
executor:
  serviceAccount:
    annotations: {}   # usually none: the executor needs no cloud access
pipeline:
  serviceAccount:
    create: true
    annotations:
      # GKE Workload Identity
      iam.gke.io/gcp-service-account: <gsa>@<project>.iam.gserviceaccount.com
      # EKS IRSA
      # eks.amazonaws.com/role-arn: arn:aws:iam::<account>:role/<role>
```

On GKE, grant `roles/iam.workloadIdentityUser` to `serviceAccount:<project>.svc.id.goog[<k8s.namespace>/<release>-synclet-connector]` as in the log-archive example above. On EKS, the role's trust policy must allow `system:serviceaccount:<k8s.namespace>:<release>-synclet-connector`.

#### Bundled Garage (Helm)

The chart can run [Garage](https://garagehq.deuxfleurs.fr), a small S3-compatible store, next to Synclet so archiving works without an external bucket:

```yaml
logArchive:
  garage:
    enabled: true
    bucket: synclet-logs        # default
    persistence:
      size: 10Gi                # default; size it for your log volume and retention
      storageClass: ""
```

This selects the `s3` backend and points Synclet at the `<fullname>-garage` Service on port 3900 (region `garage`, path-style). Leave `logArchive.backend` at `none`: combining Garage with another backend fails at render time.

Garage runs as a single-node StatefulSet (replication factor 1) with its metadata and data on one ReadWriteOnce volume. On startup it assigns its own layout, creates the access key and the bucket if they are missing and grants the key read/write on the bucket, so no bootstrap job is involved. Its RPC secret, admin token and S3 key are generated once into the `<fullname>-garage` Secret, which is kept on `helm uninstall` together with the volume: Garage refuses to start if the stored key and the Secret disagree. Delete both to start over, or bring your own Secret with `logArchive.garage.existingSecret` (keys `rpc-secret` as 64 hex chars, `admin-token`, `access-key-id`, `secret-access-key`). The S3 (3900) and admin (3903) ports are exposed only through a ClusterIP Service.

Garage is a single replica on one volume: backing up that volume is up to you, and losing it loses the archived logs. For serious production use a managed bucket (S3, R2, GCS) instead.

Garage is AGPL-3.0. The chart runs the unmodified upstream image as a separate service that Synclet talks to over the S3 API, which doesn't affect Synclet's own license.

Switching back to `none` leaves already archived logs where they are, but they can no longer be read.

### Connector Configuration

Synclet uses the Airbyte connector protocol. Connectors are Docker images that implement the Airbyte source or destination specification.

1. Add a connector repository (Settings > Repositories)
2. Browse and select connectors from the repository
3. Configure source/destination instances with connection parameters
4. Create connections between sources and destinations

#### Files written by destinations

Destinations that write files (`destination-duckdb`, `destination-csv`, `destination-local-json`, `destination-sqlite`) write under `/local` inside the connector container, as on the Airbyte platform. Where the files end up depends on the executor:

- **Docker**: set `DOCKER_EXECUTOR_LOCAL_ROOT` to a host directory and a destination path of `/local/x.duckdb` lands at `<root>/x.duckdb` on the host. The directory is shared by all jobs and never cleaned; give each connection its own file name in the destination config. The directory must be writable by the connector images' user: a root Synclet creates is made `1777` (world-writable and sticky, like `/tmp`), and an existing one keeps its permissions, with a warning at startup when it is not writable by others. With `docker compose` the root is `/tmp/synclet_local` on the host, made `1777` by the `local-root` service. Unset, each job gets its own `/local`, removed with the job, so the files are lost.
- **Kubernetes**: set the chart value `k8s.localVolume` to an existing PersistentVolumeClaim and the files land on that volume. Connector pods can run on several nodes at once, so the claim needs `ReadWriteMany` unless they all run on one node, and it must be writable by the connector image's user. Unset, each pod gets an `emptyDir`, removed with the pod.

### Writing your own connector

A connector is a Go program on [synclet-go-sdk](https://github.com/synclet-io/synclet-go-sdk), shipped as a Docker image that speaks the Airbyte Protocol. Two guides take you from an empty directory to a released image:

- [Writing a connector](https://synclet-io.github.io/synclet-site/docs/connector-development/writing-a-connector/): spec, check, discover, read, state and the conformance suite, with source-postgres as the example
- [Publishing a connector](https://synclet-io.github.io/synclet-site/docs/connector-development/publishing-a-connector/): the release tag, the GHCR image, the registry, and the template for your own connectors repository

Show what you built in [Show and tell](https://github.com/synclet-io/synclet/discussions/categories/show-and-tell).

## Native Connectors

Synclet's own Go connectors, built on [synclet-go-sdk](https://github.com/synclet-io/synclet-go-sdk), live in [synclet-io/connectors](https://github.com/synclet-io/connectors) and ship as Docker images:

| Connector | Type | Directory |
|-----------|------|-----------|
| Google Sheets | Source | [`source-google-sheets`](https://github.com/synclet-io/connectors/tree/main/connectors/source-google-sheets) |
| Google Sheets | Destination | [`destination-google-sheets`](https://github.com/synclet-io/connectors/tree/main/connectors/destination-google-sheets) |
| MySQL | Source | [`source-mysql`](https://github.com/synclet-io/connectors/tree/main/connectors/source-mysql) |
| BigQuery | Destination | [`destination-bigquery`](https://github.com/synclet-io/connectors/tree/main/connectors/destination-bigquery) |

Native connectors implement the same Airbyte Protocol and can be used interchangeably with their Docker-based counterparts.

## Tested with

These pairs were run end to end through Synclet 0.6.0 (`main` at `b7d946b`) on 2026-10-19: Docker executor, PostgreSQL 16, two runs per pair so the second run starts from the saved state. A connector that is not listed here may work; it has not been tested.

| Source | Destination | Streams and mode | Result |
|---|---|---|---|
| `airbyte/source-faker:6.2.38` | `airbyte/destination-postgres:3.0.23` | `users`, incremental, append | Passed: 100 records; the second run read 0 new records |
| `airbyte/source-postgres:3.8.7` | `airbyte/destination-postgres:3.0.23` | `public.cities`, incremental on `id`, append | Passed: 250 records; the second run re-read the row at the cursor |
| `ghcr.io/synclet-io/connectors/source-postgres:0.2.0` | `airbyte/destination-postgres:3.0.23` | `public.cities`, incremental on `id`, append | Passed: 250 records; the second run re-read the row at the cursor |
| `airbyte/source-faker:6.2.38` | `airbyte/destination-postgres:3.0.23` | `users`, full refresh, overwrite | Passed: 100 records; the second run re-read all 100 and kept 100 rows |
| `airbyte/source-faker:6.2.38` | `airbyte/destination-dev-null:0.7.24` | `users`, full refresh, overwrite | Passed: 100 records; the second run re-read all 100 |

The matrix is produced by [`tests/e2e/tested_with_test.go`](tests/e2e/tested_with_test.go), an opt-in end-to-end test. To regenerate it, run `task e2e:tested-with` with `E2E_DATABASE_URL` pointing at a PostgreSQL that connector containers can reach through `TESTED_WITH_PG_HOST` (default `host.docker.internal`); the test logs one `TESTED-WITH` line per run.

Known issues and the pairs that fail are listed on the [Connectors](https://synclet-io.github.io/synclet-site/docs/concepts/connectors/#tested-with) page. CI runs the end-to-end suite with Synclet's own protocol test connectors.

Connector images keep their own licenses: most Airbyte connectors are under ELv2 since 2025-07-22 and 51 remain MIT. See [Connector licenses](https://synclet-io.github.io/synclet-site/docs/reference/connector-licenses/).

## API

ConnectRPC API on port 8080. Proto definitions in `proto/`.

Services:
- `SourceService` -- CRUD for source instances, connection testing, schema discovery
- `DestinationService` -- CRUD for destination instances, connection testing
- `ConnectionService` -- CRUD for connections, stream configuration, state management, config import/export
- `JobService` -- Sync triggering, job monitoring, log retrieval
- `StatsService` -- Workspace and connection statistics
- `ConnectorRegistryService` -- Repository and connector management

## Project Structure

```
synclet/
├── main.go               # Entry point
├── cmd/                  # CLI commands (cobra)
├── app/                  # DI wiring (Uber FX modules)
├── modules/              # Domain modules
│   ├── app/              # Embedded frontend handler
│   ├── audit/            # Audit log
│   ├── auth/             # Authentication, users, API keys, OIDC
│   ├── notify/           # Notification channels, rules, webhooks
│   ├── pipeline/         # Sources, destinations, connections, jobs, executors
│   ├── secret/           # Encrypted secret storage
│   └── workspace/        # Workspaces, members, invites
├── pkg/                  # Shared libraries (coordinator, protocol, eventbus, migrations, ...)
├── proto/                # Protobuf definitions
├── gen/                  # Generated Go protobuf code
├── front/                # Frontend (Vue 3 + TypeScript)
└── deploy/helm/          # Helm chart
```

User documentation lives in the separate docs site repository (`synclet_site`).

## Development

| Command | Description |
|---------|-------------|
| `task dev` | Run server with hot reload |
| `task test` | Run all tests |
| `task build` | Build binary to `./bin/synclet` |
| `task proto` | Regenerate protobuf code |
| `go test ./...` | Run Go unit tests |
| `cd front && bun run dev` | Frontend dev server |
| `cd front && bun run build` | Frontend production build |

`task e2e` runs the end-to-end tests against PostgreSQL (`E2E_DATABASE_URL`, default `postgres://postgres:postgres@localhost:5432/synclet_test`) and Docker. `TestE2E_PasswordResetEmail` also needs Mailpit from `docker-compose.dev.yml`; `E2E_MAILPIT_URL` (default `http://localhost:8025`) and `E2E_MAILPIT_SMTP_PORT` (default `1025`) point it at another instance.

### CLI Reference

Every command accepts `--dotenv <file>` (repeatable) to load environment files first.

| Command | Description |
|---------|-------------|
| `synclet server` | Public API, UI and internal executor API |
| `synclet server --standalone` | Also run background jobs, event consumers and an executor (Kubernetes inside a pod, Docker otherwise) |
| `synclet jobs` | Background jobs and event consumers only (distributed mode) |
| `synclet docker-executor` | Docker sync executor, talking to the server at `EXECUTOR_API_URL` |
| `synclet k8s-executor` | Kubernetes sync executor, talking to the server at `EXECUTOR_API_URL` |
| `synclet migrate up [--module <name>]` | Apply pending migrations |
| `synclet migrate down --module <name>` | Roll back the latest migration of one module |
| `synclet migrate down --all` | Roll back the latest migration of every module |
| `synclet migrate status [--module <name>]` | Show applied and pending migrations |
| `synclet migrate create <module> <name>` | Create an empty migration file |
| `synclet generate dotenv` | Append every missing configuration variable to `.env` |
| `synclet users reset-link --email <email>` | Print a single-use password reset link for a user |

`migrate down` also needs `--yes` when `ENVIRONMENT` is anything other than `development`. `synclet --help` and `synclet <command> --help` list every flag.

`server` and `jobs` start through `app.RunServer` and `app.RunJobs`, one fx graph with the database; `docker-executor` and `k8s-executor` start through `app.RunDockerExecutor` and `app.RunK8sExecutor`, whose graphs hold no database connection, storage or use cases and reach the server only through the executor API (`EXECUTOR_API_URL`, `EXECUTOR_API_TOKEN`); control-plane variables such as `DB_DSN` are dropped from an executor's environment with a warning. `TestExecutorGraph` (`app`) and `TestExecutorDependencies` (`modules/pipeline/pipelineexec`) guard that boundary.

## Roadmap

See [ROADMAP.md](ROADMAP.md).

## License

Synclet is MIT licensed, see [LICENSE](LICENSE). Connectors are separate images with their own licenses, see [Connector licenses](https://synclet-io.github.io/synclet-site/docs/reference/connector-licenses/).

Airbyte is a trademark of Airbyte, Inc. Synclet is not affiliated with, endorsed by or sponsored by Airbyte, Inc.
