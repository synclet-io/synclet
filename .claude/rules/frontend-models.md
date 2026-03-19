---
paths:
  - "front/src/entities/**"
---

# Frontend Entity Models

Each entity must declare its own TypeScript model types in its entity directory. Do NOT reuse proto-generated types directly in components or stores.

API composables must map proto response objects to these frontend-declared models before returning data to consumers. This decouples the frontend from the wire format and makes future refactoring easier.
