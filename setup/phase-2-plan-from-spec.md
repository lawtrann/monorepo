# Phase 2: Skill `plan-from-spec`

## Goal

Write the full `plan-from-spec` skill — the core skill that transforms a spec document into actionable tasks and phase files. This is the "initializer" from the Anthropic blog, adapted for Claude Code.

## Context

- Phase 1 created the repo structure with a placeholder SKILL.md (name + description only)
- This phase replaces the placeholder with full instructions + supporting files
- Skill runs in main conversation — human sees and approves each step
- Skill uses `$ARGUMENTS` to receive the spec file path

## Skill structure after Phase 2

```
.claude/skills/plan-from-spec/
├── SKILL.md                    # Level 1 (frontmatter) + Level 2 (instructions)
├── tasks-schema.json           # Level 3 — loaded when generating tasks
└── phase-template.md           # Level 3 — loaded when generating phase files
```

## File Contents

### .claude/skills/plan-from-spec/SKILL.md

Replace the placeholder entirely with:

```markdown
---
name: plan-from-spec
description: >
  Transforms a spec document into actionable tasks and phase files.
  Use when a new spec is placed in .claude/specs/ and needs to be
  broken down into tasks.json entries and .claude/phases/ context files.
  Planning only — never writes code.
disable-model-invocation: true
argument-hint: <spec-file-path>
---

# Plan from spec

You are a planning agent. You read a spec and produce two outputs:
1. Task entries appended to `.claude/tasks.json`
2. Phase context files in `.claude/phases/`

You NEVER write implementation code. You NEVER create project files outside `.claude/`.

## Invocation

```
/plan-from-spec .claude/specs/<spec-name>.md
```

The spec file path is passed as `$ARGUMENTS`.

## Process

### Step 1: Read current state

- Read `$ARGUMENTS` (the spec file)
- Read `.claude/tasks.json` to see which phases are already planned
- Count existing phases in `.claude/phases/`

### Step 2: Propose scope

Based on the spec and what's already planned, propose the next 2-3 phases to plan.

Present to human:
```
I've read the spec and current state.

Already planned: Phase 0-2 (15 tasks)
Remaining in spec: Phases 3-14

I propose planning Phase 3-5 next:
- Phase 3: [short description]
- Phase 4: [short description]  
- Phase 5: [short description]

Approve this scope? Or adjust?
```

**STOP and wait for human approval before continuing.**

### Step 3: Identify open questions

For the proposed phases, list anything ambiguous or underspecified in the spec.

Present to human:
```
Open questions for Phase 3-5:

1. [Question about ambiguous requirement]
2. [Question about missing detail]
3. [Question about tech choice not specified]

Please answer these before I generate tasks.
```

If there are open questions: **STOP and wait for answers.**
If zero open questions: tell human "No open questions — proceeding to generate tasks."

### Step 4: Generate tasks

For each phase in scope, generate task entries following the schema in [tasks-schema.json](tasks-schema.json).

Rules for task generation:
- Each task must fit ONE coding session. If unsure, split smaller.
- `estimated_size: small` = <30 min, `medium` = 30-90 min, `large` = 90+ min (split these!)
- `depends_on` must reference only task IDs that exist (already planned or in current batch)
- `verify` must be a concrete command the agent can run (go build, go test, make, etc.)
- `file` is the PRIMARY file — the one created or most significantly changed
- `scope` is a short label (1-2 words) for the area the task touches — used as the commit scope in conventional commit messages (e.g. `auth`, `db`, `api`)
- `passes` is always `false` — only CI changes this

Present the generated tasks to human as a table:
```
Generated tasks for Phase 3:

| ID   | Description              | Size   | Depends on | Verify command         |
|------|--------------------------|--------|------------|------------------------|
| 3.1  | Add tenant_id columns    | medium | 2.1, 2.2   | make migrate           |
| 3.2  | RLS policies per table   | medium | 3.1        | go test ./internal/... |
| ...  | ...                      | ...    | ...        | ...                    |

Approve? Or adjust any tasks?
```

**STOP and wait for human approval before writing to tasks.json.**

### Step 5: Generate phase files

For each phase in scope, generate a phase context file following the template in [phase-template.md](phase-template.md).

Key principles for phase files:
- Copy all relevant code snippets from spec — agent should NOT need to open the spec
- Link to spec for "why" decisions: `> See .claude/specs/<name>.md for rationale`
- Include skill hints where relevant: `> Use /skill-name for [specific pattern]`
- Keep Prerequisites section to 1-2 lines referencing prior phase
- Phase file should be self-contained enough for a coding agent to implement any task in that phase

Present phase file to human for review before writing.

**STOP and wait for human approval before writing phase files.**

### Step 6: Write outputs

After all approvals:
1. Append new tasks to `.claude/tasks.json` (preserve existing tasks, add new ones)
2. Write phase files to `.claude/phases/phase-{NN}-{slug}.md`
3. Report what was written

### Step 7: Check if more phases remain

```
Planned so far: Phase 0-5 (X tasks)
Remaining: Phase 6-14

Ready to plan the next batch? Or stop here for now?
```

## Important rules

- NEVER skip the approval stops. Every scope, question list, task list, and phase file must be approved by human.
- NEVER generate tasks for more than 2-3 phases at a time. Keep context focused.
- NEVER write implementation code. Not even "example" code in task descriptions.
- If the spec is ambiguous, ASK — don't assume.
- If a task seems too large for one session, split it. Err on the side of smaller tasks.
```

### .claude/skills/plan-from-spec/tasks-schema.json

This is the Level 3 supporting file — loaded when planner needs to generate tasks.

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "description": "Schema for task entries in .claude/tasks.json",
  "type": "array",
  "items": {
    "type": "object",
    "required": ["id", "phase", "scope", "description", "file", "depends_on", "verify", "estimated_size", "passes"],
    "properties": {
      "id": {
        "type": "string",
        "description": "Phase.subtask format, e.g. '2.3'. Unique across all tasks.",
        "pattern": "^[0-9]+\\.[0-9]+$"
      },
      "phase": {
        "type": "integer",
        "description": "Phase number (0-14+). Groups related tasks.",
        "minimum": 0
      },
      "scope": {
        "type": "string",
        "description": "Commit scope — short label (1-2 words) for the area this task touches, e.g. 'auth', 'db', 'api'.",
        "maxLength": 30
      },
      "description": {
        "type": "string",
        "description": "What to build. One line. Clear enough that a coding agent understands the task without reading the spec.",
        "maxLength": 200
      },
      "file": {
        "type": "string",
        "description": "Primary file to create or edit. Relative to repo root.",
        "examples": ["app/eureka/internal/shared/db/pool.go", "docker-compose.yml"]
      },
      "depends_on": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Task IDs that must have passes: true before this task can start. Empty array if no dependencies.",
        "examples": [["1.1", "1.2"], []]
      },
      "verify": {
        "type": "string",
        "description": "Shell command to verify task completion. Must be runnable by the coding agent. Exit 0 = pass.",
        "examples": ["cd app/eureka && go build ./cmd/server/", "cd app/eureka && go test ./internal/shared/db/..."]
      },
      "estimated_size": {
        "type": "string",
        "enum": ["small", "medium", "large"],
        "description": "small: <30 min. medium: 30-90 min. large: 90+ min — should be split into smaller tasks."
      },
      "passes": {
        "type": "boolean",
        "description": "Always false when created. Only CI pipeline changes this to true.",
        "default": false
      }
    },
    "additionalProperties": false
  }
}
```

### .claude/skills/plan-from-spec/phase-template.md

This is the Level 3 supporting file — template for generating phase context files.

```markdown
# Phase {NN}: {Phase Title}

## Prerequisites
> Phase {NN-1} completed: {1-2 line summary of what prior phase produced}
> Key files from prior phase: `{path/to/file1}`, `{path/to/file2}`

## Decisions
- {Decision 1}: {WHAT + HOW in one line}
- {Decision 2}: {WHAT + HOW in one line}
> See .claude/specs/{spec-name}.md for rationale behind these decisions

## Tasks in this phase
| ID | Description | Size | File |
|----|-------------|------|------|
| {NN}.1 | {description} | {size} | {file} |
| {NN}.2 | {description} | {size} | {file} |

## Key patterns

{Copy relevant code snippets from spec. These are the patterns
the coding agent should follow — not invent from scratch.}

```{language}
// Pattern for {what this does}
{code from spec}
```

> Use `/skill-name` for {specific pattern} if available

## Verification
- `{verify command 1}` — {what it checks}
- `{verify command 2}` — {what it checks}

## Dependencies
- Requires: Phase {X} ({summary})
- Blocks: Phase {Y} ({why})
```

## Commands

```bash
# Create branch from develop
git checkout -b setup/phase-2

# SKILL.md placeholder already exists from Phase 1 — we're replacing its content
# Create supporting files
touch .claude/skills/plan-from-spec/tasks-schema.json
touch .claude/skills/plan-from-spec/phase-template.md
```

Then write the contents listed above into each file.

## Verification

After writing all files:

1. `cat .claude/skills/plan-from-spec/SKILL.md` — should show full YAML frontmatter + markdown instructions with 7 steps
2. `cat .claude/skills/plan-from-spec/tasks-schema.json` — valid JSON schema with 8 fields
3. `cat .claude/skills/plan-from-spec/phase-template.md` — template with Prerequisites, Decisions, Tasks, Key patterns, Verification, Dependencies sections
4. Test: start a new Claude Code session and type `/plan-from-spec` — it should recognize the skill name

```bash
git add .claude/skills/plan-from-spec/
git commit -m "feat(harness): implement plan-from-spec skill with schema and phase template"
git push -u origin setup/phase-2
# Open PR to develop
```

## What's next

Phase 3: Write `start-to-code` skill — the session-start skill that reads progress, picks next task, loads phase context, and runs smoke test.

But first: consider dropping a spec into `.claude/specs/` and testing `/plan-from-spec` to validate the skill works before building more skills on top of it.
