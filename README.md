<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/assets/synclet-logo.svg">
    <img src=".github/assets/synclet-logo-light.svg" alt="Synclet" width="450">
  </picture>
</p>

# Synclet

Data synchronization platform built on the Airbyte connector protocol. Connect any source to any destination with automatic schema detection, incremental sync, and real-time monitoring.

## Features

- **Airbyte Protocol Compatible** -- Use any Airbyte source/destination connector
- **Multiple Execution Modes** -- Docker, Kubernetes, or CLI
- **Stream Configuration** -- Field selection, namespace rewriting, primary key overrides
- **Secrets Management** -- AES-256-GCM encrypted configuration storage
- **OIDC Authentication** -- SSO with any OpenID Connect provider
- **Workspace Isolation** -- Multi-tenant with role-based access control (admin/editor/viewer)
- **Notification Channels** -- Slack, Email, Telegram alerts on sync events
- **Dashboard & Stats** -- Connection health, sync timeline, records/bytes metrics
- **Config Import/Export** -- YAML-based workspace backup and restore
- **Dark Mode** -- System-aware theme switching
- **Native Connectors** -- Go-native high-performance connectors alongside Docker-based Airbyte connectors

## Architecture

Synclet is a **modular monolith** -- a single Go binary runs the API server, scheduler, and workers as goroutines. Each domain module (auth, workspace, pipeline, notify, secret, audit) is a bounded context with its own storage, service layer, and ConnectRPC handlers; `app` serves the frontend.

- **Backend**: Go 1.25, ConnectRPC (protobuf), PostgreSQL, GORM, Uber FX
- **Frontend**: Vue 3, TypeScript, TanStack Vue Query, Tailwind CSS
- **Connectors**: Docker containers running Airbyte protocol connectors
- **Orchestration**: Docker (default) or Kubernetes for distributed execution

## Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 16+
- Docker
- Bun (for frontend)

### Development Setup

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

**Docker Compose** (PostgreSQL, migrations and a standalone server with the Docker executor):
```bash
echo "AUTH_JWT_SECRET=$(openssl rand -base64 32)" >> .env
echo "SECRET_ENCRYPTION_KEY=$(openssl rand -base64 32)" >> .env
mkdir -p /tmp/synclet && chmod 1777 /tmp/synclet
docker compose up -d
```

The image runs as UID 65532. On Linux, add `DOCKER_GID=$(stat -c %g /var/run/docker.sock)` to `.env` so the container can reach the Docker socket.

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
| `AUTH_OIDC_PROVIDERS` | Comma-separated OIDC provider slugs, each configured by `AUTH_OIDC_<SLUG>_*` | empty |
| `AUTH_OIDC_CALLBACK_BASE_URL` | Base URL for OIDC callbacks | empty |
| `WORKSPACE_MODE` | `single` shared workspace or `multi` | `single` |
| `WORKSPACE_FRONTEND_URL` | Base URL used in invite links | `http://localhost:5173` |
| `NOTIFY_SMTP_HOST` | SMTP server; empty disables email | empty |
| `NOTIFY_SMTP_PORT` | SMTP port | `587` |
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
| `K8S_EXECUTOR_NAMESPACE` | Namespace for connector pods | empty |
| `EXECUTOR_API_URL` | Internal server URL used by distributed executors | empty |
| `EXECUTOR_API_TOKEN` | Token distributed executors present to the internal server | empty |
| `WATERMILL_TRANSPORT` | Event bus transport | `sql` |

### Job Log Storage

While a job runs, its log lines are stored in Postgres and the UI tails them from there. Each attempt keeps at most `PIPELINE_LOG_MAX_LINES_PER_ATTEMPT` lines; later lines are dropped and counted in a single `[synclet] log truncated: N further lines dropped` line.

With `PIPELINE_LOG_ARCHIVE_BACKEND=none` (the default) logs stay in Postgres until job retention deletes the job. With `filesystem`, `s3` or `gcs`, a finished job's logs are moved out of Postgres into one gzipped NDJSON object per attempt, keyed `logs/<workspace>/<connection>/<job>/<attempt>.ndjson.gz`, and read back from there; retention and workspace deletion remove the objects too.

- `filesystem` writes under `PIPELINE_LOG_ARCHIVE_DIR`. Every server replica must see the same directory, so use a shared volume.
- `s3` works with AWS S3, MinIO, Cloudflare R2 and Google Cloud Storage (HMAC keys via its S3 interoperability endpoint). Set `PIPELINE_LOG_ARCHIVE_S3_ENDPOINT` for anything but AWS, and `PIPELINE_LOG_ARCHIVE_S3_USE_PATH_STYLE=true` for MinIO. Without static keys the default AWS credential chain is used (environment, IRSA, instance profile).
- `gcs` uses the native Google Cloud Storage API. With `PIPELINE_LOG_ARCHIVE_GCS_CREDENTIALS_FILE` it authenticates with that service account key; without it, Application Default Credentials (GKE Workload Identity, `GOOGLE_APPLICATION_CREDENTIALS`, or `gcloud auth application-default login` locally). The server logs which credential source it uses at startup. Credentials are resolved on first use, so missing ADC shows up as archive errors, not a startup failure.

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

The executor deployment (distributed mode) shares this service account because it needs its RBAC to launch connector pods. Connector pods run in `k8s.namespace` under `pipeline.resources.serviceAccountName` (or that namespace's default account), so they get no bucket access.

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

## Native Connectors

In addition to Docker-based Airbyte connectors, Synclet includes Go-native connectors built with the airbyte-go-sdk for improved performance and simplified deployment:

| Connector | Type | Directory |
|-----------|------|-----------|
| Google Sheets | Source | `connectors/source-google-sheets/` |
| Google Sheets | Destination | `connectors/destination-google-sheets/` |
| MySQL | Source | `connectors/source-mysql/` |
| BigQuery | Destination | `connectors/destination-bigquery/` |

Native connectors implement the same Airbyte protocol and can be used interchangeably with their Docker-based counterparts.

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
├── connectors/           # Go-native Airbyte connectors
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

### CLI Reference

Every command accepts `--dotenv <file>` (repeatable) to load environment files first.

| Command | Description |
|---------|-------------|
| `synclet server` | Public API, UI and internal executor API |
| `synclet server --standalone` | Also run background jobs, event consumers and an executor (Kubernetes inside a pod, Docker otherwise) |
| `synclet jobs` | Background jobs and event consumers only (distributed mode) |
| `synclet docker-executor [--standalone]` | Docker sync executor; distributed by default, talking to `EXECUTOR_API_URL` |
| `synclet k8s-executor [--standalone]` | Kubernetes sync executor; distributed by default |
| `synclet migrate up [--module <name>]` | Apply pending migrations |
| `synclet migrate down --module <name>` | Roll back the latest migration of one module |
| `synclet migrate down --all` | Roll back the latest migration of every module |
| `synclet migrate status [--module <name>]` | Show applied and pending migrations |
| `synclet migrate create <module> <name>` | Create an empty migration file |
| `synclet generate dotenv` | Append every missing configuration variable to `.env` |

`migrate down` also needs `--yes` when `ENVIRONMENT` is anything other than `development`. `synclet --help` and `synclet <command> --help` list every flag.

## License

See LICENSE
