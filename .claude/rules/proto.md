---
paths:
  - "proto/**/*.proto"
---

# Protobuf Rules

- Fields use `snake_case`: `workspace_id`, `refresh_token`.
- Enums use `SCREAMING_SNAKE_CASE`: `REQUIRED_ROLE_VIEWER`.
- Packages must be versioned: `synclet.publicapi.auth.v1`.
- Any field with a finite, known set of values MUST be an enum — never a string.
- Nested structured data with known types MUST use message types — not JSON strings or bytes. Use bytes/strings only when the schema is truly dynamic or unknown.
- Mutually exclusive fields MUST use `oneof` — never optional fields with implicit exclusion.
