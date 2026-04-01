---
paths:
  - "modules/**/*service/**"
---

# Service / Use Case Rules

Use cases must NEVER use `gorm.DB` or `sql.DB` directly. All database access goes through the Storage interface. SQL belongs in storage layer only.

# Ports / Adapters

External dependencies (other modules, third-party services, infrastructure) MUST be passed to use cases as interfaces defined in `{module}service/`. Implementations live in `{module}adapt/`, never in `{module}service/`.

- Interface = port (defined where it's consumed, in the service package)
- Adapter = implementation (in `{module}adapt/`, wired via FX)
- Use cases depend only on interfaces, never on concrete types from other modules or packages

## Domain Errors

Do NOT use `errors.New()` or `fmt.Errorf()` for domain-level errors (validation failures, not-found, permission denied, etc.). Declare them in `{module}service/errors.go`
Use `fmt.Errorf` with `%w` only for wrapping infrastructure/unexpected errors, not for domain conditions.
