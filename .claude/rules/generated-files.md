---
paths:
  - "modules/**/gen.*.go"
  - "gen/**"
  - "front/src/gen/**"
---

# Generated Files

NEVER edit generated files manually:
- `gen.*.go` — auto-generated from `gen.models.yaml` via `task boilerplate-go`
- `gen/proto/` — auto-generated from `.proto` files via `task proto`
- `front/src/gen/` — auto-generated TypeScript proto types via `task proto`

To change domain models, enums, errors, or storage interfaces: edit `gen.models.yaml` and run `task boilerplate-go`.
To change API contracts: edit `.proto` files in `proto/` and run `task proto`.
