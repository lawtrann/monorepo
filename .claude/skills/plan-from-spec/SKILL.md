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
- `passes` is always `false` — only CI changes this
- `scope` is a short label (1-2 words) for the area the task touches — used as the commit scope in conventional commit messages (e.g. `auth`, `db`, `api`)
- For each `verify` command, check: does the required tool/binary exist yet? If it depends on a tool installed by another task (e.g. `alembic check` needs Python env from a prior task), ensure that task is in `depends_on`. If no prior task sets up the tool, either make the current task include setup, or create a prerequisite task.
- **Trace execution order**: After generating all tasks, mentally trace through the dependency chain. For each task, verify that its `verify` command only references files, directories, and tools that will exist after all `depends_on` tasks have completed. This catches circular references and missing intermediate dependencies.
- **No external URL references in phase files**: Phase files must be fully self-contained. Never reference external URLs (GitHub repos, docs sites) as required reading for the coding agent — embed the relevant content directly. The agent may not have internet access.
- **Validate referenced types/packages**: If code patterns in the phase file reference types from external packages (e.g. `uuid.UUID`, `pgx.Row`), verify those packages are listed in the dependency table. Flag any missing packages as open questions.
- **Include auth/credentials for service tasks**: If a task requires interacting with an external service API (e.g. Casdoor, MinIO), the phase file must document default credentials, auth flow, and API base URL.
- **Verify Docker image tags**: If tasks reference specific Docker image tags (e.g. `postgres:18`), verify the tag exists or note it as an open question.

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
