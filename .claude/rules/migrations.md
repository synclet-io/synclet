---
paths:
  - "**/*dbstate*/**"
  - "**/*.sql"
---

# Database Migrations

- Modify initial migration files directly — do NOT create new migration files
- Enum fields MUST use Postgres enum types (`CREATE TYPE ... AS ENUM`), never TEXT
- Use lowercase enum values
- Never modify migrations files if they were already in main branch 