# Contributing to Synclet

Thanks for your interest in Synclet. Bug reports, feature ideas, documentation fixes, connectors and code are all welcome.

## Where to start

- **Questions and ideas:** [GitHub Discussions](https://github.com/synclet-io/synclet/discussions) — use *Q&A* for "how do I…", *Ideas* for proposals, *Show and tell* for what you built.
- **Bugs:** open an issue with the *Bug report* template. Include the Synclet version, how you run it (binary, Docker, Helm) and the job log if a sync failed.
- **Connector requests:** use the *Connector request* template. Synclet runs connectors that implement the Airbyte Protocol, so name the image you want to run and what happens when you try.
- **First contribution:** look for issues labelled [`good first issue`](https://github.com/synclet-io/synclet/labels/good%20first%20issue) or [`help wanted`](https://github.com/synclet-io/synclet/labels/help%20wanted). Comment on the issue before you start so nobody duplicates the work.
- **Security problems:** do not open a public issue; see [SECURITY.md](SECURITY.md).

## Development setup

Requirements: Go (version from `go.mod`), PostgreSQL 16 (started through Docker Compose), Docker, [Bun](https://bun.sh) for the frontend and [Task](https://taskfile.dev) as the command runner.

```bash
git clone https://github.com/synclet-io/synclet.git
cd synclet
cp .env.dist .env                  # then fill DB_DSN, AUTH_JWT_SECRET, SECRET_ENCRYPTION_KEY
go run . generate dotenv           # appends every other variable with its default

task dev                           # starts Postgres and runs the server with hot reload
cd front && bun install && bun run dev   # Vite dev server on http://localhost:5173
```

The frontend uses Bun only; please don't commit `package-lock.json` or `yarn.lock`.

### Useful commands

| Command | What it does |
|---|---|
| `task build` | Builds `./bin/synclet` |
| `task test` | Runs the Go tests; the database tests skip themselves |
| `task test:db` | Runs the Go tests with the database tests on the dev PostgreSQL |
| `task lint` / `task lint:fix` | Runs golangci-lint |
| `task proto` | Regenerates protobuf code (`buf generate`) |
| `task boilerplate-go` | Regenerates domain models and storage from the model definitions |
| `task migrate:up` | Applies database migrations |
| `task e2e` | End-to-end tests (needs Docker and Mailpit) |
| `cd front && bun run test:run` | Frontend unit tests (Vitest) |

Tests that need PostgreSQL read `TEST_DATABASE_DSN` and are skipped when it is unset, so `task test` alone does not run them. `task test:db` starts the dev PostgreSQL from `docker-compose.dev.yml`, creates the `synclet_test` database there if it is missing and runs the whole suite with `TEST_DATABASE_DSN` pointing at it. Set `POSTGRES_PORT` when port 5432 is taken, or set `TEST_DATABASE_DSN` yourself to use another database; arguments after `--` go to `go test`:

```bash
task test:db
POSTGRES_PORT=5438 task test:db -- -count=1
TEST_DATABASE_DSN='postgres://user:pass@localhost:5432/synclet_test?sslmode=disable' task test:db
```

`task e2e` runs against the database in `E2E_DATABASE_URL` and expects a Mailpit on `localhost:8025` (API) and `localhost:1025` (SMTP) for the password reset test. When :8025 is not Mailpit, start one on other ports and set `E2E_MAILPIT_URL` and `E2E_MAILPIT_SMTP_PORT`:

```bash
docker run -d -p 18025:8025 -p 11025:1025 axllent/mailpit:v1.27
E2E_MAILPIT_URL=http://localhost:18025 E2E_MAILPIT_SMTP_PORT=11025 task e2e
```

## Code layout

- `modules/` — the backend, a modular monolith wired with Uber FX. Handlers call use cases; use cases call storage; SQL lives only in storage.
- `pkg/` — shared infrastructure, including the coordinator that runs connector containers.
- `proto/` — ConnectRPC API definitions; generated code lands in `gen/`.
- `front/` — the Vue 3 UI.
- Synclet's own Go connectors live in a separate repository, [synclet-io/connectors](https://github.com/synclet-io/connectors).
- `deploy/` — the Helm chart.

Generated files (`gen/`, generated storage and models) are not edited by hand: change the source definition and run the generator. CI fails when generated code drifts.

Database migrations are append-only once merged: a schema change goes into a new migration file. Enum-like columns use PostgreSQL enum types with lowercase values.

## Pull requests

1. Fork the repository and create a branch from `main`.
2. Keep a pull request to one change; add or update tests for it.
3. Run `task lint` and `task test` (and the frontend tests if you touched `front/`).
4. Title the pull request as a [Conventional Commit](https://www.conventionalcommits.org/): `feat(pipeline): …`, `fix(front): …`, `docs: …`. CI checks the title, and release notes are generated from it.
5. Fill in the pull request template.

A maintainer reviews pull requests, usually within a few days. By contributing you agree that your contribution is licensed under the [MIT License](LICENSE).

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating you agree to uphold it.
