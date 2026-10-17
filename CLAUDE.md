# Synclet

Data synchronization platform. Go backend (ConnectRPC) + Vue 3 frontend.
Modular monolith with Uber FX DI. Implements Airbyte connector protocol via Docker.

## Commands

```bash
# Backend
task dev                    # Run server with hot reload (starts DB automatically)
task build                  # Build binary to ./bin/synclet
task test                   # Run all Go tests
task lint                   # golangci-lint
task lint:fix               # Auto-fix lint issues
task boilerplate-go         # Regenerate domain models & storage from gen.models.yaml
task proto                  # Regenerate protobuf (buf generate)
task migrate:up             # Apply database migrations
task e2e                    # E2E tests (requires Docker, 10m timeout)

# Frontend (ALWAYS use bun, NEVER npm/npx)
cd front && bun install
cd front && bun run dev     # Vite dev server on :5173
cd front && bun run build   # Production build
```

## Git

Format: `feat(module): description` (conventional commits). Commit after completing a feature or fix.

## Environment

Requires: Go 1.26, PostgreSQL 16 (via `docker compose`), Bun, Docker.
Config: `.env` file (copy from `.env.dist`; `go run . generate dotenv` appends every other variable with its default). Key vars: `DB_DSN`, `AUTH_JWT_SECRET`, `SECRET_ENCRYPTION_KEY`.

## Architecture decisions

Decisions about the sync engine's architecture (job queue, executors, router, connector protocol boundary, state handling, native connectors) are made by a human: present the options with their trade-offs and wait for the choice; never settle one yourself.

## Migrations

A migration file may be modified only while it has not been pushed to GitHub; once pushed, every schema change (a new table, column, type or index) goes into a new migration file, never into an existing one.
