# Roadmap

What Synclet works on next, as of 2026-10-18. Synclet has one maintainer, so this is a plan, not a promise: items move when users report what they need. Open an [issue](https://github.com/synclet-io/synclet/issues) or a [discussion](https://github.com/synclet-io/synclet/discussions) to push an item up or propose a new one.

Self-hosted Synclet stays MIT with all features. A managed cloud version may come later; if it does, it will be said openly, and the self-hosted binary keeps every feature.

## Now: OSS launch (October to November 2026)

- A one-command start with Docker Compose and Helm, released binaries and images for every release.
- A README and docs site that answer what Synclet is, how it compares and how to try it, checked by clean-machine installs by people outside the project.
- Honest connector wording: a [tested-with matrix](https://synclet-io.github.io/synclet-site/docs/concepts/connectors/#tested-with) of the connector images and versions actually run, and a [Connector licenses](https://synclet-io.github.io/synclet-site/docs/reference/connector-licenses/) page.
- A published benchmark of Synclet against Airbyte Core on the same machine: idle memory, install time and per-sync overhead.
- Distribution: Artifact Hub for the Helm chart and a Homebrew tap.

## Next: connector compatibility and Go connectors

- Fix the known issues of the tested-with matrix: state handling of full-refresh streams, stream status messages, and a `/local` mount for file destinations.
- Grow the tested-with matrix: more source and destination pairs, each with pinned versions, and a test per major Airbyte Protocol change.
- MIT Go connectors in [synclet-io/connectors](https://github.com/synclet-io/connectors), built on [synclet-go-sdk](https://github.com/synclet-io/synclet-go-sdk): hardening the MySQL, PostgreSQL and Google Sheets sources, then MongoDB, S3 (CSV, JSONL, Parquet), Stripe, GitHub, HubSpot, Salesforce and Shopify sources, and CDC for PostgreSQL and MySQL.
- A connector developer guide for writing your own connector.
- A guide for migrating from Airbyte Core.

## Later: sync engine

- Executor groups with capacity caps per group, schedule jitter and per-workspace concurrency caps.
- Retry backoff with jitter, and no retries for configuration errors.
- Chained syncs: a connection runs after other connections complete.
- An executor registry: executors report themselves and the settings page shows the fleet.
- Native destinations that run in process instead of as a container (PostgreSQL, ClickHouse, BigQuery, S3/Parquet), checked against the Airbyte destination they replace by a conformance harness.
- A Cloud Run Jobs executor (spike).

## Later: product

- Enhanced alerting: richer alert rules on top of the Slack, email, Telegram and webhook channels.
- A dbt integration: run dbt after a sync completes.
- A Terraform provider for sources, destinations and connections.
- Sync dry run: check what a connection would sync before it writes anything.

## Under consideration

- An MCP server for Synclet.
- A public demo instance with synthetic data.
- A connector picker with starred and recently used connectors.

## Not planned

- A low-code connector builder inside Synclet.
- Features held back from the MIT binary for a paid tier.

Airbyte is a trademark of Airbyte, Inc. Synclet is not affiliated with, endorsed by or sponsored by Airbyte, Inc.
