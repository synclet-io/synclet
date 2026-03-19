---
name: feature
description: End-to-end feature implementation with plan, development, code review, and security audit loop.
argument-hint: <feature description>
---

Implement the following feature for Synclet (Go backend + Vue 3 frontend data synchronization platform):

**Feature:** $ARGUMENTS

Follow this pipeline strictly. Do NOT skip steps.

---

## Step 1: Plan

Enter plan mode. This is mandatory — do NOT start coding without an approved plan.

In the plan:
- Read `CLAUDE.md` to understand project conventions and architecture
- Identify which modules, files, and layers are affected
- Design the implementation approach (API changes, service logic, storage, frontend)
- Consider module isolation boundaries
- List the files you will create or modify
- Present the plan for user approval

Wait for the user to approve the plan before proceeding.

---

## Step 2: Develop

Implement the feature according to the approved plan:
- Follow all conventions in `CLAUDE.md` (module prefix naming, error handling, DI patterns)
- Write clean, minimal code — no over-engineering
- Ensure the code compiles: run `go build ./...` after backend changes
- Run `go test ./...` to verify no tests break

---

## Step 3: Code Review

After implementation is complete, invoke the `/code-review` skill to review all changes.

Analyze the review output:
- If verdict is **REQUEST_CHANGES**: fix all Critical issues and as many Warnings as reasonable, then re-run `/code-review`. Repeat until verdict is **APPROVE**.
- If verdict is **APPROVE**: proceed to Step 4.

---

## Step 4: Security Audit

Invoke the `/security-audit` skill to audit the changes for security issues.

Focus the audit on the files and modules changed in this feature, not the entire codebase.

Analyze the audit output:
- If there are **CRITICAL** or **HIGH** findings: fix them, then go back to Step 2 (develop the fix), Step 3 (code review the fix), and Step 4 (re-audit).
- If only **MEDIUM/LOW** or no findings: proceed to completion.

---

## Completion

Summarize what was implemented:
- Files created/modified
- Key design decisions
- Any remaining MEDIUM/LOW security notes for future attention

Ask the user if they want to commit the changes.
