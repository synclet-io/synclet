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

Synclet is a **modular monolith** -- a single Go binary runs the API server, scheduler, and workers as goroutines. Each domain module (auth, workspace, pipeline, notify) is a bounded context with its own storage, service layer, and ConnectRPC handlers.

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

2. Start PostgreSQL:
   ```bash
   docker compose up -d
   ```

3. Configure environment:
   ```bash
   cp .env.example .env
   # Edit .env with your database URL and encryption key
   ```

4. Run database migrations:
   ```bash
   go run main.go migrate up
   ```

5. Start backend (with hot reload):
   ```bash
   task dev
   ```

6. Start frontend:
   ```bash
   cd front && bun install && bun run dev
   ```

The API server runs on `http://localhost:8080` and the frontend dev server on `http://localhost:5173`.

### Production Deployment

**Docker:**
```bash
docker build -t synclet .
docker run -p 8080:8080 --env-file .env synclet server
```

**Helm (Kubernetes):**
```bash
helm install synclet deploy/helm/synclet/ -f values.yaml
```

## Configuration Reference

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | required |
| `JWT_SECRET` | Secret for signing JWT tokens | required |
| `SECRET_ENCRYPTION_KEY` | Base64-encoded 32-byte AES key for secrets | required |
| `PIPELINE_LOG_STORE_DIR` | Directory for sync log storage | `./data/logs` |
| `PIPELINE_WORKER_INTERVAL` | Worker polling interval | `1s` |
| `PIPELINE_SCHEDULER_INTERVAL` | Cron scheduler interval | `30s` |
| `PIPELINE_MAX_SYNC_DURATION` | Maximum sync execution time | `24h` |
| `PIPELINE_IDLE_TIMEOUT` | Idle timeout before sync cancellation | `10m` |
| `REGISTRATION_ENABLED` | Enable or disable user registration | `true` |
| `SMTP_HOST` | SMTP server for email notifications | optional |
| `SMTP_PORT` | SMTP port | `587` |
| `SMTP_USERNAME` | SMTP authentication username | optional |
| `SMTP_PASSWORD` | SMTP authentication password | optional |
| `OIDC_PROVIDER_*` | OIDC provider configuration (issuer, client ID, secret) | optional |

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
├── cmd/                   # CLI commands (cobra)
├── app/                   # DI wiring (Uber FX modules)
├── modules/               # Domain modules
│   ├── auth/              # Authentication & users
│   ├── workspace/         # Multi-tenancy, roles
│   ├── pipeline/          # Sources, destinations, connections, jobs, sync
│   └── notify/            # Webhooks & notifications
├── connectors/            # Go-native Airbyte connectors
├── pkg/                   # Shared utilities
│   ├── docker/            # Docker container runner
│   ├── k8s/               # Kubernetes orchestrator
│   ├── protocol/          # Airbyte protocol message handling
│   └── connector/         # Connector client
├── proto/                 # Protobuf definitions
├── front/                 # Frontend (Vue 3 + TypeScript)
├── deploy/                # Helm charts, Dockerfiles
└── docs/                  # Documentation
```

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

| Command | Description |
|---------|-------------|
| `synclet server` | Start the API server, scheduler, and workers |
| `synclet server --standalone` | Start in standalone mode (no external dependencies) |
| `synclet run <config.yaml>` | Run a one-off sync from a YAML config file |
| `synclet migrate up` | Apply database migrations |
| `synclet migrate down` | Rollback the last database migration |
| `synclet connector list` | List available connectors |
| `synclet connector validate` | Validate connector specifications |
| `synclet orchestrate` | Start the K8s orchestrator (distributed mode) |

## License

See LICENSE
