---
paths:
  - "**/*dbstate*/**"
  - "**/*.sql"
---

# Database Migrations

- Modify a migration file only while it is unpushed; every later schema change goes into a new migration file
- Enum fields MUST use Postgres enum types (`CREATE TYPE ... AS ENUM`), never TEXT
- Use lowercase enum values
- Never modify migrations files if they were already in main branch 