---
paths:
  - "modules/**/*service/**"
---

# Service / Use Case Rules

Use cases must NEVER use `gorm.DB` or `sql.DB` directly. All database access goes through the Storage interface. SQL belongs in storage layer only.
