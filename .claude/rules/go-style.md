---
paths:
  - "**/*.go"
---

# Go Code Style

- **Errors**: `fmt.Errorf` with `%w` wrapping. Domain errors are generated per-module in `gen.errors.go`
- **Close errors**: `defer multierr.AppendInvoke(&rerr, multierr.Close(x))` with named return `rerr error`. Never `_ = x.Close()` (except in goroutines/handlers without error returns)
- **Package naming**: Module prefix required — `syncservice`, `authservice`, NOT `service`, `storage`
- **Use case constructors**: `NewCreateProject()`, `NewGetUser()` with `Execute` method
- **DI**: Uber FX constructor injection. Register in `app/module_*.go`
- **Comments**: English only
