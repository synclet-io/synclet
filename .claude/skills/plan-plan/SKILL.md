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

4. **Load all context** — Read any additional files in the plan directory:
   - `UI-SPEC.md` if it exists (UI specification for frontend tasks)
   - Any other supplementary files

## Phase 1: High-Level Wave Outline

Using context from `PLAN.md`, `RESEARCH.md`, and `UI-SPEC.md` (if present), create a high-level wave breakdown. Do NOT create task files yet.

Write `{planDir}/WAVES.md`:

```markdown
# Wave Outline: {Plan Title}

## Wave 1: {Short description of what this wave achieves}

**Goal:** {What exists after this wave that didn't before}
**Focus areas:** {e.g., "proto schema, database migration, codegen"}
**Estimated tasks:** {N}
**Key files likely affected:**
- `{path1}`
- `{path2}`

## Wave 2: {Short description}

**Goal:** {What this wave adds on top of wave 1}
**Focus areas:** {e.g., "use case logic, handler wiring"}
**Estimated tasks:** {N}
**Key files likely affected:**
- `{path1}`

## Wave 3: {Short description}
...
```

### Wave Organization Rules

- **Waves represent dependency layers** — ALL tasks in wave N depend on wave N-1 being complete
- Tasks within the same wave have **no interdependencies** and can run in parallel
- **File conflict rule**: Tasks that modify the same file MUST be in different waves — parallel agents would overwrite each other's changes
- Wave 1 has no dependencies (foundational work: schemas, types, interfaces, config fixes)
- If a task depends on a specific task in wave N (logically or by file conflict), it goes in wave N+1

Present the wave outline to the user and confirm before proceeding to detailed task breakdown.

## Phase 2: Sequential Wave Planning

For each wave (one at a time, sequentially — NOT in parallel), spawn a **sonnet Plan** agent to create detailed task files for that wave.

### Agent Prompt for Wave {W}

```
You are planning implementation tasks for Wave {W} of a plan.

## Plan Context
- Plan file: {planDir}/PLAN.md
- Research file: {planDir}/RESEARCH.md
- UI Spec file: {planDir}/UI-SPEC.md (read if exists)
- Wave outline: {planDir}/WAVES.md

Read ALL of these files to understand the full context.

## Previous Waves
{If W > 1: list the task files from previous waves so the agent knows what's already planned}
Read the task files from previous waves to understand what will already be done:
{list of wave_*/TASK_*.md paths from completed waves}

## Your Job

Create detailed task files for **Wave {W} only**: "{wave description from WAVES.md}"

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

### Task Numbering

Task numbers are **global across all waves**. The first task in this wave should be numbered {next_task_number}.

### Output

Create directory `{planDir}/wave_{W}/` and write each task to `{planDir}/wave_{W}/TASK_{T}.md`:

```markdown
---
task: {T}
wave: {W}
scope: S | M
deps: [{list of task numbers this depends on}]
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

Return a summary of tasks created: task numbers, descriptions, files, and scope.
```

After each wave agent completes:
- Read the task files it created to confirm quality
- Track the next available task number for the next wave
- Proceed to the next wave

## Phase 3: Verification

After all wave agents complete, spawn a **sonnet Plan** agent as a **verifier**:

### Verifier Prompt

```
You are a plan verifier. Review the complete implementation plan for correctness, completeness, and consistency.

## Context Files — Read ALL of these:
- Plan: {planDir}/PLAN.md
- Research: {planDir}/RESEARCH.md
- UI Spec: {planDir}/UI-SPEC.md (if exists)
- Wave outline: {planDir}/WAVES.md
- All task files: {planDir}/wave_*/TASK_*.md

## Also read the codebase files referenced in RESEARCH.md to verify accuracy.

## Verification Checklist

Check each item and report pass/fail with details:

### 1. Coverage
- [ ] Every goal from PLAN.md is addressed by at least one task
- [ ] Every affected file from RESEARCH.md is covered by a task
- [ ] Every design decision from RESEARCH.md is reflected in task details
- [ ] If UI-SPEC.md exists, every UI requirement is covered

### 2. Correctness
- [ ] File paths in tasks match actual codebase paths
- [ ] Function signatures and struct names match existing code patterns
- [ ] Proto field numbers, types, and naming follow project conventions
- [ ] Migration SQL is consistent with existing schema patterns

### 3. Dependencies
- [ ] No circular dependencies between tasks
- [ ] Tasks that modify the same file are in different waves
- [ ] Wave ordering makes sense (foundations before features, backend before frontend)
- [ ] No task references output from a same-wave or later-wave task

### 4. Completeness
- [ ] Each task has specific acceptance criteria (not vague)
- [ ] Each task specifies exact files to create/modify
- [ ] Details include enough information for an agent to implement without guessing
- [ ] Generated code steps (codegen, proto compilation) are included where needed

### 5. Coherence
- [ ] No contradictory instructions within the same task (e.g., "use approach A" followed by "actually use approach B")
- [ ] No leftover "stream of consciousness" where the author changed their mind mid-file but left both versions
- [ ] Each task has ONE clear approach — no hedging, no "alternatively we could..."
- [ ] Details sections don't contradict acceptance criteria

### 6. Risks
- [ ] No tasks try to do too much (>3 files)
- [ ] No missing error handling or edge cases for critical paths
- [ ] No security concerns in the planned approach

## Output

Write your review to {planDir}/REVIEW.md:

```markdown
# Plan Review: {Plan Title}

## Verdict: PASS | NEEDS_FIXES | NEEDS_DECISIONS

## Summary
{2-3 sentence overall assessment}

## Unresolved Questions
{If NEEDS_DECISIONS — questions that require user input before planning can proceed:}

### Question {N}: {title}
- **Context:** {why this matters for the plan}
- **Options:** {concrete options with trade-offs}
- **Affected tasks:** {which tasks are blocked by this decision}
- **Risk if ignored:** {what goes wrong if we guess}

## Issues Found
{If NEEDS_FIXES — list each issue with:}

### Issue {N}: {title}
- **Severity:** critical | warning
- **Wave/Task:** {wave and task number, or "missing task"}
- **Problem:** {what's wrong}
- **Fix:** {specific fix needed}

## Passed Checks
{List checks that passed}
```

Return the verdict (PASS, NEEDS_FIXES, or NEEDS_DECISIONS) and the full content of REVIEW.md.
```

### Handling Verification Results

- **If PASS**: proceed to Phase 4

- **If NEEDS_DECISIONS**:
  1. Read `{planDir}/REVIEW.md` to understand the unresolved questions
  2. Present each question to the user via `AskUserQuestion` — include the context, options, and risks from the review
  3. After user answers all questions, append the decisions to `{planDir}/RESEARCH.md` under a new `## Additional Decisions (from plan review)` section:
     ```markdown
     ## Additional Decisions (from plan review)

     > [!question] {Question title}
     > {Context and options presented}

     > [!answer] {User's decision}
     > {What the user chose and why}
     ```
  4. Re-run the affected wave planners with the updated RESEARCH.md context, then re-run the verifier

- **If NEEDS_FIXES**:
  1. Read `{planDir}/REVIEW.md` to understand the issues
  2. For each affected wave, re-spawn a Plan agent to fix the specific issues found:
     - Pass the REVIEW.md issues relevant to that wave
     - Tell the agent which task files to update/create
     - The agent should edit existing TASK files (not recreate from scratch)
  3. After fixes, run the verifier again

- **Max 2 verification rounds** — if still failing after 2 rounds, present remaining issues to user and ask for guidance

## Phase 4: Update PLAN.md

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

## Phase 5: Update Index

If `maintainIndex` is true in config, ensure the plan entry in `_Plan Index.md` reflects the approved status.

## Phase 6: Present Overview

Show:

---

### Plan Ready: {Title}

**Tasks:** {N} tasks in {W} waves
**Scope breakdown:** {X small, Y medium}
**Verification:** {PASS or PASS after N fix rounds}

**Wave summary:**
- **Wave 1:** {description} — {task count} tasks
- **Wave 2:** {description} — {task count} tasks
- ...

**Risks:** {top risks from RESEARCH.md}

**Plan directory:** `{planDir}/`

---

Then tell user: "Run `/plan:execute {index}` to start implementation."
