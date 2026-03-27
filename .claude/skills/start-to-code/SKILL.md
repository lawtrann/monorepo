---
name: start-to-code
description: >
  Resume work at the start of a coding session. Reads progress,
  picks the next available task based on dependency order,
  loads phase context, and runs smoke test before implementation.
  Also handles continuing a task left in progress.
disable-model-invocation: true
---

# Start to code

You are a coding agent starting a new session. You have zero memory of previous sessions.
Everything you need is on the filesystem.

## Invocation

```
/start-to-code
```

No arguments needed — the skill reads state from harness files.

## Process

### Step 1: Read progress

Read `.claude/progress/latest.md`.

Two scenarios:
- **Has `Status: IN_PROGRESS`** → A task was started but not finished. Continue that task (skip to Step 4 with the in-progress task).
- **Has `Status: COMPLETED` or `Status: BLOCKED`** → Pick a new task (continue to Step 2).
- **No `Status:` field** (e.g. initial state with no prior sessions) → Pick a new task (continue to Step 2).

If status is BLOCKED, report it:
```
Previous session was BLOCKED on task X.Y: [reason from progress]
Attempting to pick next available task instead.
```

### Step 2: Pick next task

Read `.claude/tasks.json`.

Filter logic:
1. `passes == false`
2. ALL items in `depends_on` have `passes == true` in tasks.json
3. Sort by: phase ASC, then id ASC
4. Pick first match

If no task is available:
```
All tasks are either completed or blocked by dependencies.
Nothing to work on. Run /plan-from-spec to plan more phases.
```
Then stop.

### Step 3: Load phase context

From the picked task's `phase` field, find and read:
```
.claude/phases/phase-{NN}-*.md
```

This file contains: decisions, code patterns, verification commands, and skill hints for this phase. It is your primary reference during implementation.

### Step 4: Sync develop + create branch

**First, sync with develop:**
```bash
git fetch origin develop:develop
git checkout develop
```

**Check for clean working tree:**
```bash
git status --porcelain
```
If there are uncommitted changes, warn and stop:
```
Working tree is not clean. Please commit or stash changes before starting a new session.
```

**Rotate progress file:**
- Read current `.claude/progress/latest.md`
- If it has a `Status:` field (i.e. real session content), rename it:
  - Count existing `session-*.md` files in `.claude/progress/`
  - N = count + 1
  - Rename `latest.md` → `session-{N}.md` (e.g. `session-1.md`, `session-2.md`)
- Create new `.claude/progress/latest.md` with:
```markdown
# Session {N+1}
Date: {today}
Task: {task.id} — {task.description}
Phase: {task.phase}
Status: IN_PROGRESS

## Plan
{brief plan for implementing this task}

## Infra state
{List Docker services running, DB schemas created, external services configured.
If early phase with no infra yet, write "No infrastructure yet."}
```

**Create task branch:**
```bash
git checkout -b task/{id}_{slug}
```

Where `{slug}` is the task description slugified (lowercase, hyphens, max 50 chars).

### Step 5: Smoke test

Run `/smoke-test` before proceeding.

If smoke test reports failures:
- If it's a pre-existing failure (not caused by current task): fix it first, commit as `fix: resolve pre-existing {issue} before task {id}`
- If unfixable: update progress to BLOCKED, explain the failure, stop.

If all pass or no checks apply yet (early phases): proceed to Step 6.

If `/smoke-test` does not produce a clear pass/fail verdict (e.g. skill errors out or returns ambiguous output), treat as PASS with caution — log a warning in progress and proceed, but be extra careful during implementation.

### Step 6: Implement

Now implement the task. Use the phase file as your primary reference for:
- Code patterns to follow
- Decisions already made
- Skill hints (e.g. `> Use /skill-name for ...`)

Rules during implementation:
- Stay focused on THIS task only. Do not refactor unrelated code.
- Follow patterns from the phase file, not your own preferences.
- Each concern gets its own commit. Do not bundle unrelated changes.
- Run the task's `verify` command before considering the task done.

### Step 7: Verify + commit

1. Run the verify command from `tasks.json` for this task
2. If verify fails → fix and retry. If stuck → mark BLOCKED in progress, stop.
3. If verify passes:
   - Commit with message: `feat({scope}): {description} [task {id}]`
     where `{scope}` comes from the task's `scope` field in tasks.json

### Step 8: Update progress + push + PR

1. Update `.claude/progress/latest.md`:
```markdown
# Session {N}
Date: {today}
Task: {task.id} — {task.description}
Phase: {task.phase}
Status: COMPLETED

## Summary
{what was implemented, key decisions made}

## Commits
- {commit hash}: {message}

## Infra state
{Current state: which Docker services running, DB migrations applied, external services configured}

## Next
Task {next_id} is next in dependency order (informational only).
```

2. Commit progress: `docs: update session progress to COMPLETED [task {id}]`
3. Push branch: `git push -u origin task/{id}_{slug}`
4. Create PR targeting `develop`:
   ```bash
   gh pr create --base develop --title "feat({scope}): {task.description}" --body "Task {id} from phase {phase}. Verify: \`{verify}\`"
   ```
5. Add the PR URL to `latest.md` under a `## PR` section and amend the progress commit:
   ```bash
   git add .claude/progress/latest.md
   git commit --amend --no-edit
   git push --force-with-lease
   ```

## Important rules

- ONE task per session. After pushing PR, session is done.
- If verify fails and you cannot fix in reasonable effort, mark BLOCKED — do not loop endlessly.
- Atomic commits: each commit addresses one concern. Never bundle unrelated changes.
- Do not modify tasks.json — CI handles marking passes.
- Always update progress/latest.md — it's the handoff to the next session.
