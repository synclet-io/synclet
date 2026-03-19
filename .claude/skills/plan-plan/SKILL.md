---
name: plan:plan
description: Break down a researched plan into waves and atomic tasks — writes wave_*/TASK_*.md files.
argument-hint: <plan-index>
allowed-tools: Read, Write, Edit, Bash, Glob, Grep, Task, AskUserQuestion
---

Break a researched plan into implementation tasks:

**Plan index:** $ARGUMENTS

## Prerequisites

1. **Load config** — read `.claude/plan-config.json`. If missing, tell user to run `/plan:init` first and stop.

2. **Find the plan** — Find the directory matching `{index}-*` in `{planningDir}/` and read both `PLAN.md` and `RESEARCH.md`. If `$ARGUMENTS` is empty, list plans and ask user to pick.

3. **Verify status** — Plan must have `status: researched`. If not, tell user to run `/plan:research` first.

## Phase 1: Task Breakdown

Using the context from `PLAN.md` (goal, scope) and `RESEARCH.md` (affected files, architecture, design decisions), break the implementation into atomic tasks grouped by dependency waves.

### Task Granularity Rules

Each task MUST:
- Touch **1-3 files maximum** — if a task needs more, split it
- Be **completable in a single focused session** by one agent
- Have **clear input/output boundaries** — what exists before, what exists after
- Include **exact function signatures**, struct definitions, or proto message names where possible
- Have **specific acceptance criteria** — not vague ("works correctly") but testable ("returns 200 with JSON body containing `id` field")

Scope estimates:
- **S** = ~1 file, straightforward change (add field, new simple function, config update)
- **M** = 2-3 files, requires some design (new handler + use case, migration + model update)

### Wave Organization Rules

- Group tasks into waves where all tasks within a wave have **no interdependencies**
- Tasks in later waves explicitly list which earlier tasks they depend on
- Wave 1 should contain foundational work (schemas, types, interfaces)
- Later waves build on earlier foundations (implementations, integrations, tests)

## Phase 2: Write Task Files

1. **Create wave subdirectories and task files** — For each wave and task:
   - Create directory: `{planDir}/wave_{W}/`
   - Write each task to: `{planDir}/wave_{W}/TASK_{T}.md`
   - Task numbering `{T}` is **global across all waves** (not per-wave)

### TASK_*.md Format

```markdown
---
task: {T}
wave: {W}
scope: S | M
status: pending
---

# Task {T}: {Short action description}

## Files

- `{path1}` — {create/modify}
- `{path2}` — {create/modify}

## Depends On

- {Task N: description} (or "None")

## Details

- {Exact function signatures, struct definitions, or API changes}
- {Step-by-step what to create/modify}
- {Reference specific patterns from RESEARCH.md to follow}

## Acceptance Criteria

- [ ] {Specific, verifiable condition 1}
- [ ] {Specific, verifiable condition 2}
```

## Phase 3: Update PLAN.md

Append an `## Implementation Tasks` section to `PLAN.md` with a summary table:

```markdown
## Implementation Tasks

| Wave | Task | Description | Files | Scope |
|------|------|-------------|-------|-------|
| 1 | T1 | {description} | `{files}` | S |
| 1 | T2 | {description} | `{files}` | M |
| 2 | T3 | {description} | `{files}` | S |
```

Update frontmatter: `status: approved`

## Phase 4: Update Index

If `maintainIndex` is true in config, ensure the plan entry in `_Plan Index.md` reflects the approved status.

## Phase 5: Present Overview

Show:

---

### Plan Ready: {Title}

**Tasks:** {N} tasks in {W} waves
**Scope breakdown:** {X small, Y medium}

**Wave summary:**
- **Wave 1:** {description} — {task count} tasks
- **Wave 2:** {description} — {task count} tasks
- ...

**Risks:** {top risks from RESEARCH.md}

**Plan directory:** `{planDir}/`

---

Then tell user: "Run `/plan:execute {index}` to start implementation."
