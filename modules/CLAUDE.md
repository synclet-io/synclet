# Backend Modules

7 modules: `auth`, `workspace`, `pipeline`, `notify`, `secret`, `audit`, `app`. Each is a bounded context.

## Package Pattern

- `{name}service/` - Use cases (one file = one operation, 50-150 lines)
- `{name}storage/` - GORM repos (generated + `custom_*.go` for complex queries)
- `{name}connect/` - ConnectRPC handlers
- `{name}adapt/` - Adapters for inter-module contracts
- `{name}dbstate/` - Goose migrations (`*_initial.sql` plus later timestamped migrations, created with `synclet migrate create <module> <name>`)

## Module Isolation
- No direct imports between modules — use interfaces + adapters
- Interface defined in consuming module, adapter also in consuming module

## Codegen Workflow

1. Define models in `{module}/gen.models.yaml`
2. Add table DDL in a new migration in `{name}dbstate/` (edit `*_initial.sql` only while nothing has shipped against it)
3. Run `task boilerplate-go` to regenerate all `gen.*.go` files
