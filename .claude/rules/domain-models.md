---
paths:
  - "modules/**/gen.models.yaml"
---

# Domain Model YAML Rules

- Any field with a finite, known set of values MUST be an enum — never a plain string.
- Nested structured data with known types MUST use typed structs — not JSON strings or raw bytes. Use unstructured types only when the schema is truly dynamic or unknown.
- Mutually exclusive fields MUST use `one_of` types — never optional fields with implicit exclusion.
