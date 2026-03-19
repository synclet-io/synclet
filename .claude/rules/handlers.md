---
paths:
  - "modules/**/*connect/**"
  - "modules/**/*route/**"
---

# Handler Rules

Handlers (ConnectRPC, HTTP) must NOT contain business logic. Their only job is:
1. Parse the request into domain types
2. Call a single use case
3. Convert the result to a response

Parse-level validation (required fields, type conversion) is fine. Business logic validation (password length, quota checks, etc.) belongs in use cases. No orchestration, no conditional logic beyond error mapping.
