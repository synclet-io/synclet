---
name: plan:create
description: Light codebase overview, create plan folder with PLAN.md describing the problem and high-level approach.
argument-hint: <feature or task description>
allowed-tools: Read, Write, Edit, Bash, Glob, Grep, Task, AskUserQuestion
---

Create a new plan for:

**Task:** $ARGUMENTS

## Prerequisites

1. **Load config** — read `.claude/plan-config.json`. If missing, tell user to run `/plan:init` first and stop.

## Phase 1: Quick Codebase Overview

Spawn an **Explore** agent (subagent_type: "Explore", thoroughness: "quick") with this prompt:

> Do a quick overview of the codebase for: {task description}
>
> 1. Read `CLAUDE.md` in the repo root
> 2. Identify which modules/layers are likely affected
> 3. Note the general architecture and tech stack relevant to this task
>
> Return a brief summary of: project structure, affected areas, and any obvious constraints.

## Phase 2: Write PLAN.md

1. **Determine the plan index** — List existing directories in `{planningDir}` matching `{N}-*` pattern. The new index is `max(N) + 1` (or `1` if none exist).
2. **Generate a plan slug** from the task description (kebab-case, max 40 chars).
3. **Create the plan directory**: `{planningDir}/{index}-{slug}/`
4. **Write**: `{planningDir}/{index}-{slug}/PLAN.md`

### PLAN.md Format

```markdown
---
tags: [plan, {module-tags}]
status: draft
created: {ISO date}
scope: {affected modules}
---

# {Plan Title}

> [!abstract] Summary
> {2-3 sentence overview of what this plan achieves and why}

## Problem

{What problem are we solving? Why does it matter? What's the current state?}

## Goal

{What does success look like? What should exist when this is done?}

## Scope

{What's in scope and what's explicitly out of scope}

## Affected Areas

| Layer | Area | Impact |
|-------|------|--------|
| {layer} | {module/component} | {brief description} |

## High-Level Approach

{2-5 sentences on the general strategy. Not detailed implementation — just the direction.}

## Open Questions

- {Questions that need answering during research phase}
- {Unknowns that affect the approach}
```

## Phase 3: Update Index

If `maintainIndex` is true in config, update `_Plan Index.md`:
- Add a wiki link under "Active Plans": `- [[{index}-{slug}/PLAN]] — {one-line summary} ({date})`

## Phase 4: Present to User

Show:

---

### Plan Created: {Title}

**Index:** {index}
**Scope:** {affected areas}
**Approach:** {1-2 sentences}

**Plan saved to:** `{path}`

---

Then tell user: "Run `/plan:research {index}` to deep-dive into the codebase and make design decisions."
