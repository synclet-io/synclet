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

## Execution Strategy — Worktree Isolation

**CRITICAL: The main repo working directory must NEVER be modified.** All work happens in git worktrees under `.claude/worktrees/`.

### Setup

1. **Create plan branch** (without checkout): `git branch plan/{slug} main`
2. **Create plan worktree**: `git worktree add .claude/worktrees/plan-{slug} plan/{slug}`
   - This worktree is where wave merges happen and verification runs
   - Store this path as `{planWorktree}` (absolute path)

### Per-Wave Execution

For each wave:

1. **Create agent worktrees** — For each task in the wave:
   ```
   git worktree add .claude/worktrees/plan-{slug}-task-{T} -b worktree-plan-{slug}-task-{T} plan/{slug}
   ```
   - Each agent gets its own worktree branched from the current state of `plan/{slug}`
   - Store the absolute path as `{agentWorktree}`

2. **Spawn agents** — Use `Task` tool with `subagent_type: "general-purpose"` and `team_name`. Do **NOT** use `isolation: "worktree"` — worktrees are pre-created above.

3. **Agent prompt** must instruct the agent to work exclusively in its worktree path (see Agent Prompt below).

4. **After wave completes** — Merge agent branches into the plan branch:
   ```bash
   cd {planWorktree}
   git merge worktree-plan-{slug}-task-{T} --no-edit
   ```
   - Merge each agent branch sequentially
   - If merge conflicts occur, resolve them in `{planWorktree}` before merging the next branch

5. **Clean up agent worktrees**:
   ```bash
   git worktree remove .claude/worktrees/plan-{slug}-task-{T}
   git branch -d worktree-plan-{slug}-task-{T}
   ```

6. **Verify build** in `{planWorktree}` before starting the next wave

### Waves execute sequentially — wave 2 starts only after wave 1 is fully merged.

## Team Setup

1. **Create a team** via `TeamCreate`:
   - Team name: `plan-{slug}`
   - Description: Plan title from the markdown

2. **Create task items** via `TaskCreate` for each plan task:
   - Subject: task description from plan
   - Description: full details including files, acceptance criteria, details
   - Set up `blockedBy` dependencies between tasks based on wave structure

3. **Execute wave by wave** — For each wave, spawn implementer agents (max 3 concurrent):
   - Use `Task` tool with `subagent_type: "general-purpose"` and `team_name`
   - Do **NOT** use `isolation: "worktree"` — agent worktrees are pre-created
   - Each agent gets a focused prompt:

## Agent Prompt

```
You are implementing a task from a plan.

## CRITICAL: Working Directory

Your worktree is at: {agentWorktree}
You MUST use this path as the base for ALL file operations:
- Read/Write/Edit: use absolute paths under {agentWorktree}/
- Bash commands: always cd to {agentWorktree} first, or use absolute paths
- Glob/Grep: set path to {agentWorktree} or subdirectories within it
- NEVER modify files outside {agentWorktree}

## Project Conventions
{conventions summary from prerequisites}

## Your Task
- Task: {task subject}
- Task file: {planDir}/wave_{W}/TASK_{T}.md
- Files: {file list} (these are relative paths — prefix with {agentWorktree}/)
- Details: {full task details}
- Acceptance criteria: {criteria list}

## Rules
1. Follow project conventions strictly
2. Only modify files listed in your task (read others as needed — still within {agentWorktree})
3. Verify your changes compile/build: run build commands from {agentWorktree}
4. Run relevant tests if they exist (from {agentWorktree})
5. Commit your changes with a conventional commit message:
   cd {agentWorktree} && git add -A && git commit -m "feat(scope): description"
6. When done, mark your task as completed via TaskUpdate
7. Report what you did via SendMessage to the team lead

## Task Summary
After completing your work, write a summary file at {planDir}/wave_{W}/SUMMARY_{T}.md with:
- What was changed and why
- Key decisions made
- Any deviations from the plan
- Files modified

Also send a brief summary via SendMessage.
```

4. **Coordinate execution**:
   - Monitor task completion via `TaskList`
   - If an agent reports issues, help resolve or reassign

## Progress Tracking

Update the plan file as tasks complete:
- Change task checkboxes: `- [ ]` to `- [x]`
- Add completion notes under each task if relevant

## Verification

After all tasks complete, run checks **inside the plan worktree**:

1. **Build check**: `cd {planWorktree} && <build command from CLAUDE.md>`
2. **Test check**: `cd {planWorktree} && <test command>`
3. **Lint check**: `cd {planWorktree} && <lint command>` (if available)

If any check fails:
- Identify which task's changes caused the issue
- Fix directly in `{planWorktree}` or spawn a targeted agent (with its own worktree from `plan/{slug}`)

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

**Plan branch:** plan/{slug}
**Plan worktree:** {planWorktree}

{Key highlights from task summaries}
```

7. Ask user how they want to handle the plan branch:
   - **Create PR** — push the branch and create a pull request:
     ```bash
     git push -u origin plan/{slug}
     gh pr create --base main --head plan/{slug} --title "feat: {plan title}" --body "$(cat <<'EOF'
     ## Summary
     {2-3 bullet points from execution summary}

     ## Changes
     {Per-task summary of what was done}

     ## Test plan
     - [ ] Build passes
     - [ ] Tests pass
     - [ ] Lint passes
     {Additional verification items from task acceptance criteria}

     🤖 Generated with [Claude Code](https://claude.com/claude-code)
     EOF
     )"
     ```
     Then clean up the worktree (keep the branch): `git worktree remove {planWorktree}`
     Return the PR URL to the user.
   - **Squash merge** into main — from the main repo (which is still on main): `git merge --squash plan/{slug} && git commit -m "feat: {plan title}"`, then clean up: `git worktree remove {planWorktree} && git branch -d plan/{slug}`
   - **Regular merge** into main — `git merge plan/{slug} --no-edit`, then clean up: `git worktree remove {planWorktree} && git branch -d plan/{slug}`
   - **Keep branch** — leave `plan/{slug}` worktree as-is for manual review. User can clean up later with `git worktree remove {planWorktree}`
