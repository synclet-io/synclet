---
paths:
  - "modules/**/*service/**"
---

# Workspace Scoping

All data queries must filter by WorkspaceID. Before accessing any resource, validate it belongs to the requesting workspace. Never return data without workspace isolation.
