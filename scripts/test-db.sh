#!/usr/bin/env bash
# Runs the Go tests with TEST_DATABASE_DSN set. Without a DSN in the
# environment it starts the dev Postgres from docker-compose.dev.yml and
# creates the test database there if it is missing.
set -euo pipefail

if [[ -z "${TEST_DATABASE_DSN:-}" ]]; then
  compose=(docker compose -f docker-compose.dev.yml)
  db="${TEST_DB_NAME:-synclet_test}"

  "${compose[@]}" up -d postgres

  for _ in $(seq 60); do
    "${compose[@]}" exec -T postgres pg_isready -h localhost -U synclet -q && break
    sleep 1
  done
  "${compose[@]}" exec -T postgres pg_isready -h localhost -U synclet -q

  exists=$("${compose[@]}" exec -T postgres psql -U synclet -d synclet -tAc \
    "SELECT 1 FROM pg_database WHERE datname = '${db}'")
  if [[ -z "${exists}" ]]; then
    "${compose[@]}" exec -T postgres createdb -U synclet "${db}"
  fi

  export TEST_DATABASE_DSN="postgres://synclet:synclet@localhost:${POSTGRES_PORT:-5432}/${db}?sslmode=disable"
fi

go test "$@" ./...
