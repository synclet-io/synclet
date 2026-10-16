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
| `DOCKER_EXECUTOR_MAX_SYNC_DURATION` | Maximum sync run time on the Docker executor | `24h` |
| `DOCKER_EXECUTOR_TEMP_DIR_ROOT` | Scratch directory shared with the host Docker daemon | empty |
| `K8S_EXECUTOR_NAMESPACE` | Namespace for connector pods | empty |
| `EXECUTOR_API_URL` | Internal server URL used by distributed executors | empty |
| `EXECUTOR_API_TOKEN` | Token distributed executors present to the internal server | empty |
| `WATERMILL_TRANSPORT` | Event bus transport | `sql` |

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
