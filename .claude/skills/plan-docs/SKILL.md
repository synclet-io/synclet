---
name: plan:docs
description: Update user-facing documentation after a feature is implemented. Covers setup guides, feature docs, and usage instructions.
argument-hint: <plan-slug, plan-path, or feature description>
allowed-tools: Read, Write, Edit, Bash, Glob, Grep, Task, AskUserQuestion, WebSearch
---

Update user-facing documentation for a completed feature.

**Feature:** $ARGUMENTS

## Purpose

This skill updates **user documentation** — the kind displayed on a docs site to help end-users work with the project. This is NOT internal/technical documentation. Focus on:
- How users interact with the feature
- Setup and configuration steps
- Usage examples and workflows
- UI screenshots or descriptions where relevant

## Prerequisites

1. **Load config** — read `.claude/plan-config.json` for planning directory path.

2. **Identify the feature** — determine what was implemented:
   - If `$ARGUMENTS` is a plan slug/path, read the plan file (especially the Summary, Architecture, and Execution Summary sections)
   - If it's a description, use it directly

3. **Find existing docs** — Spawn an **Explore** agent to locate documentation:
   - Search for `docs/`, `documentation/`, `content/`, `pages/` directories
   - Look for common doc frameworks: Docusaurus, VitePress, MkDocs, Nextra, plain markdown
   - Identify the doc structure: sidebar config, navigation, categories
   - Find existing pages that may need updates (related features, getting started, config reference)
   - Return: doc framework, directory structure, existing pages to update, and conventions (frontmatter format, heading style, etc.)

## Phase 1: Scope Q&A

Use `AskUserQuestion` to clarify documentation scope:

- What audience is this for? (end users, admins, developers integrating with API)
- Should this be a new page or update to existing page(s)?
- Any specific sections or examples the user wants included?
- Where in the navigation should new pages appear?

Skip questions where the answer is obvious from context.

## Phase 2: Write Documentation

Spawn a **general-purpose** agent to write/update the documentation. Pass it:
- The feature summary (from plan or description)
- The doc framework and conventions found in Prerequisites
- The scope decisions from Phase 1
- Existing pages that need updates

The agent should:

### For New Pages
- Create the page in the correct directory following existing conventions
- Use the same frontmatter format as other pages
- Structure with clear headings: Overview, Prerequisites, Setup/Configuration, Usage, Examples
- Include code snippets, CLI commands, or UI descriptions as appropriate
- Add the page to sidebar/navigation config if applicable

### For Existing Page Updates
- Add new sections or update existing ones
- Maintain consistent tone and style with the rest of the page
- Update any "feature list" or "what's new" sections if they exist

### Content Guidelines
- Write for the target audience — avoid internal implementation details
- Lead with the "what" and "why", then the "how"
- Use concrete examples over abstract descriptions
- Include copy-pasteable code snippets and commands
- Note any prerequisites, environment variables, or config needed
- Keep an architecture section if useful, but at a high level (no internal module details)

## Phase 3: Review

After the agent writes the docs, review the output:

1. Read the written/updated files
2. Verify they follow the project's doc conventions
3. Check for broken internal links or references

Present a summary to the user:

---

### Documentation Updated: {Feature Title}

**Pages modified:** {list of files}
**Pages created:** {list of new files}

**Summary of changes:**
{Brief description of what was documented}

**Navigation:** {Where users can find it}

---

Ask: "Review the docs, request changes, or approve?"

- If **approved**: done
- If **changes requested**: incorporate feedback via another agent pass
