---
name: plan:execute
description: Execute an approved plan using Claude Code teams for parallel implementation.
argument-hint: <plan-slug or plan-path>
allowed-tools: Read, Write, Edit, Bash, Glob, Grep, Task, TeamCreate, TeamDelete, TaskCreate, TaskUpdate, TaskList, TaskGet, SendMessage, AskUserQuestion
---

Execute the approved plan:

**Plan:** $ARGUMENTS

## Prerequisites

1. **Load config** — read `.claude/plan-config.json`. If missing, tell user to run `/plan:init` first.

2. **Load project conventions** — Spawn a **haiku Explore** agent to collect all project instructions:
   - Read `CLAUDE.md` (root)
   - Glob `.claude/rules/*.md` and read all matches
   - Read any subdirectory `CLAUDE.md` files relevant to the plan scope
   - Return a condensed summary of conventions, build/test commands, and code style rules

3. **Find and read the plan**:
   - If `$ARGUMENTS` is a file path, read it directly
   - If it's a number (plan index), find the directory matching `{index}-*` in `{planningDir}/` and read `{planningDir}/{index}-*/PLAN.md`
   - If empty, list available approved plans and ask user to pick one

4. **Verify plan is approved** — check frontmatter `status: approved`. If not approved, tell user to run `/plan:plan` first.

5. **Read task files** — Read all `TASK_*.md` files from the plan's wave subdirectories (`wave_*/TASK_*.md`). These contain the detailed task specifications.

## Execution Strategy

Analyze the plan's tasks and waves:
- **Create a plan branch** from `main` before starting execution: `git checkout -b plan/{slug} main`
- Tasks within the same wave run in parallel via team agents
- Each agent works in an **isolated git worktree** (`isolation: "worktree"` on the Task tool) to avoid file conflicts
- Waves execute sequentially (wave 2 starts after wave 1 completes)
- After all agents in a wave finish, **merge all worktree branches** into the plan branch (not main) before starting the next wave
- The plan branch accumulates all changes; main stays clean until the user decides to merge

## Team Setup

1. **Create a team** via `TeamCreate`:
   - Team name: `plan-{slug}`
   - Description: Plan title from the markdown

2. **Create task items** via `TaskCreate` for each plan task:
   - Subject: task description from plan
   - Description: full details including files, acceptance criteria, details
   - Set up `blockedBy` dependencies between tasks based on wave structure

3. **Execute wave by wave** — For each wave, spawn implementer agents (max 3 concurrent):
   - Use `Task` tool with `subagent_type: "general-purpose"`, `team_name`, and `isolation: "worktree"`
   - Each agent works in its own worktree branch, committing changes there
   - Each agent gets a focused prompt including the task file path and summary file path:

```
You are implementing a task from a plan.

## Project Conventions
{conventions summary from prerequisites}

## Your Task
- Task: {task subject}
- Task file: {planDir}/wave_{W}/TASK_{T}.md
- Files: {file list}
- Details: {full task details}
- Acceptance criteria: {criteria list}

## Rules
1. Follow project conventions strictly
2. Only modify files listed in your task (read others as needed)
3. Verify your changes compile/build after making them
4. Run relevant tests if they exist
5. Commit your changes in the worktree with a conventional commit message
6. When done, mark your task as completed via TaskUpdate
7. Report what you did via SendMessage to the team lead

## Task Summary
After completing your work, write a summary file at `{planDir}/wave_{W}/SUMMARY_{T}.md` with:
- What was changed and why
- Key decisions made
- Any deviations from the plan
- Files modified

Also send a brief summary via SendMessage.
```

4. **Coordinate execution**:
   - Monitor task completion via `TaskList`
   - If an agent reports issues, help resolve or reassign

5. **Merge wave branches** — After all agents in a wave complete:
   - Ensure you are on the plan branch: `git checkout plan/{slug}`
   - Each worktree agent produces a branch with its changes
   - Merge each worktree branch into the plan branch sequentially: `git merge <worktree-branch> --no-edit`
   - If merge conflicts occur, resolve them before merging the next branch
   - After all branches are merged, clean up: remove worktree directories first (`git worktree remove <path>`), then delete branches (`git branch -d <branch>`)
   - Verify the merged result builds/compiles before starting the next wave

## Progress Tracking

Update the plan file as tasks complete:
- Change task checkboxes: `- [ ]` to `- [x]`
- Add completion notes under each task if relevant

## Verification

After all tasks complete:

1. **Build check**: Run the project's build command (check `CLAUDE.md` for the correct command)
2. **Test check**: Run the project's test command
3. **Lint check**: Run the project's lint command (if available)

If any check fails:
- Identify which task's changes caused the issue
- Fix directly or spawn a targeted agent to fix

## Completion

1. **Collect task summaries** — Read all `SUMMARY_*.md` files from the plan's wave subdirectories.

2. **Write execution summary** — Append a `## Execution Summary` section to the plan's `PLAN.md`:

```markdown
## Execution Summary

> [!success] Completed {date}

### Changes Made

{For each task, a 2-3 line summary of what was done, key decisions, and any deviations}

### Task 1: {title}
- {What was changed and why}
- {Key decisions or deviations from plan}

### Task 2: {title}
- ...

### Build & Test Results
- **Build:** pass/fail
- **Tests:** pass/fail
- **Lint:** pass/fail

### Notes
{Any issues encountered, follow-up work needed, or observations}
```

3. **Update plan frontmatter**: `status: completed`, add `completedAt: {ISO date}`

4. **Update index** if `maintainIndex` is true:
   - Move plan from "Active Plans" to "Completed Plans" in `_Plan Index.md`

5. **Shut down team**: send shutdown requests to all agents, then `TeamDelete`

6. **Present summary** to user:

```
## Execution Complete: {Plan Title}

**Tasks completed:** {N}/{total}
**Files modified:** {list}
**Build:** pass/fail
**Tests:** pass/fail

{Key highlights from task summaries}
```

7. Ask user how they want to handle the plan branch:
   - **Squash merge** into main (`git checkout main && git merge --squash plan/{slug} && git commit -m "feat: {plan title}"`) — single clean commit
   - **Regular merge** into main (`git checkout main && git merge plan/{slug} --no-edit`) — preserves full history
   - **Keep branch** — leave `plan/{slug}` as-is for manual review
