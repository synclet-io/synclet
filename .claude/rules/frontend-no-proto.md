---
paths:
  - "front/src/pages/**"
  - "front/src/features/**"
  - "front/src/widgets/**"
---

# Frontend Component Rules

NEVER import proto types (`@/gen/`) directly. Use entity-layer composables and types from `@entities/` for all data access.
