# Backend Modules

6 modules: `auth`, `workspace`, `pipeline`, `notify`, `secret`, `app`. Each is a bounded context.

## Package Pattern

- `{name}service/` - Use cases (one file = one operation, 50-150 lines)
- `{name}storage/` - GORM repos (generated + `custom_*.go` for complex queries)
- `{name}connect/` - ConnectRPC handlers
- `{name}adapt/` - Adapters for inter-module contracts
- `{name}dbstate/` - Goose migrations (single initial migration file per module)

## Module Isolation
- No direct imports between modules — use interfaces + adapters
- Interface defined in consuming module, adapter also in consuming module

## Codegen Workflow

1. Define models in `{module}/gen.models.yaml`
2. Add table DDL to the module's initial migration in `{name}dbstate/`
3. Run `task boilerplate-go` to regenerate all `gen.*.go` files
