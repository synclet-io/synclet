---
name: plan:init
description: Initialize planning config — set Obsidian vault path and preferences for the plan workflow.
argument-hint: "[vault-path]"
allowed-tools: Read, Write, Edit, Bash, AskUserQuestion, Glob
---

Initialize the planning workflow configuration.

## Config Location

Config file: `.claude/plan-config.json` (MUST be gitignored).

## Process

1. **Check if config exists** — read `.claude/plan-config.json`. If it exists, show current config and ask if user wants to reconfigure.

2. **Ensure gitignored** — verify `.claude/plan-config.json` is covered by `.gitignore`. If not, append it.

3. **Collect settings** via `AskUserQuestion`:

   a. **Planning directory**: Where to store plans.
      - If `$ARGUMENTS` is provided, use that as the path
      - Otherwise ask: "Where should plans be stored?"
      - Options: "Obsidian vault (specify path)", "Local `.planning/` directory", "Custom path"
      - If Obsidian: ask for the vault path and subfolder (default: `{vault}/Projects/{project-name}/`)

   b. **Plan template style**:
      - "Detailed" (architecture + decisions + tasks + risks)
      - "Compact" (architecture + tasks only)

   c. **Auto-create index note**: Whether to maintain a `_Plan Index.md` MOC (Map of Content) that links all plans.

4. **Write config** to `.claude/plan-config.json`:

```json
{
  "planningDir": "/absolute/path/to/planning/folder",
  "templateStyle": "detailed",
  "maintainIndex": true,
  "projectName": "<detected from package.json, go.mod, or directory name>",
  "createdAt": "<ISO timestamp>"
}
```

5. **Create planning directory** if it doesn't exist.

6. **If `maintainIndex` is true**, create `_Plan Index.md` in the planning directory:

```markdown
---
tags: [plan-index, moc]
---

# Plan Index

> [!info] Map of Content
> All plans for **{projectName}**. Auto-maintained by plan workflow.

## Active Plans

_No plans yet. Use `/plan:create` to create your first plan._

## Completed Plans

_None yet._
```

7. **Confirm** — show the user what was configured and where files will be stored.
