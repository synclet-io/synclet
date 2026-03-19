---
paths:
  - "modules/**/*adapt/**"
---

# Adapter Rules

Adapters MUST only call use cases from the target module. They must NEVER use `sql.DB`, `gorm.DB`, or Storage interfaces directly. All data access goes through use cases.
