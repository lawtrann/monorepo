# Phase 3: Skill `start-to-code` + Dry-run Fixes

## Goal

1. Write the full `start-to-code` skill — the coding session launcher
2. Fix `plan-from-spec` SKILL.md with "verify dependency check" rule (from Phase 2 dry-run)
3. Add atomic commit rule to `rules/harness.md`

## Context

- Phase 2 dry-run revealed: planner doesn't check whether verify commands have their dependencies installed
- `start-to-code` is the daily driver — agent calls it to begin every coding session
- Agent auto-codes after skill completes setup. Human reviews at PR level.
- Skill is manual-only (disable-model-invocation: true) — user decides when to start coding

---

## Part A: `start-to-code` skill

### Skill structure

```
.claude/skills/start-to-code/
└── SKILL.md          # Level 1 (frontmatter) + Level 2 (instructions)
```

No Level 3 supporting files needed — this skill reads existing harness files (tasks.json, phases/, progress/).

### .claude/skills/start-to-code/SKILL.md

Replace the placeholder entirely with:

```markdown
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
- **Status: IN_PROGRESS** → A task was started but not finished. Continue that task (skip to Step 4 with the in-progress task).
- **Status: COMPLETED or BLOCKED or "No sessions yet"** → Pick a new task (continue to Step 2).

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

### Step 4: Rotate progress + create branch

**Rotate progress file:**
- Read current `.claude/progress/latest.md`
- If it has real content (not "No sessions yet"), rename it:
  - Count existing `session-*.md` files in `.claude/progress/`
  - Rename `latest.md` → `session-{NNN}.md` (zero-padded, e.g. `session-001.md`)
- Create new `.claude/progress/latest.md` with:
```markdown
# Session {NNN}
Date: {today}
Task: {task.id} — {task.description}
Phase: {task.phase}
Status: IN_PROGRESS

## Plan
{brief plan for implementing this task}
```

**Create branch:**
```bash
git checkout develop
git pull origin develop
git checkout -b task/{id}_{slug}
```

Where `{slug}` is the task description slugified (lowercase, hyphens, max 50 chars).

### Step 5: Smoke test

Before writing any code, verify the environment is healthy.

Run these checks in order (adapt to what exists in the project):
1. **Build check** — `go build ./...` or equivalent for each module that exists
2. **Existing tests** — `go test ./...` or equivalent (only if test files exist)
3. **Service health** — if docker-compose.yml exists and services are expected, check `docker-compose ps`

If any check fails:
- If it's a pre-existing failure (not caused by current task): fix it first, commit as `fix: resolve pre-existing {issue} before task {id}`
- If unfixable: update progress to BLOCKED, explain the failure, stop.

If all pass or no checks apply yet (early phases): proceed to Step 6.

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

### Step 7: Verify + commit + push

1. Run the verify command from `tasks.json` for this task
2. If verify fails → fix and retry. If stuck → mark BLOCKED in progress, stop.
3. If verify passes:
   - Commit with message: `feat({scope}): {description} [task {id}]`
   - Push branch: `git push -u origin task/{id}_{slug}`
   - Create PR targeting `develop`:
     ```bash
     gh pr create --base develop --title "feat({scope}): {task.description}" --body "Task {id} from phase {phase}. Verify: \`{verify}\`"
     ```

### Step 8: Update progress

Update `.claude/progress/latest.md`:
```markdown
# Session {NNN}
Date: {today}
Task: {task.id} — {task.description}
Phase: {task.phase}
Status: COMPLETED

## Summary
{what was implemented, key decisions made}

## Commits
- {commit hash}: {message}

## PR
- {PR URL}

## Next
Task {next_id} is next in dependency order (informational only).
```

## Important rules

- ONE task per session. After pushing PR, session is done.
- If verify fails and you cannot fix in reasonable effort, mark BLOCKED — do not loop endlessly.
- Atomic commits: each commit addresses one concern. Never bundle unrelated changes.
- Do not modify tasks.json — CI handles marking passes.
- Always update progress/latest.md — it's the handoff to the next session.
```

---

## Part B: Fix `plan-from-spec` SKILL.md

Add this rule to Step 4 (task generation rules) in `.claude/skills/plan-from-spec/SKILL.md`, after the existing rules:

**Add after** the line `- passes is always false — only CI changes this`:

```markdown
- For each verify command, check: does the required tool/binary exist yet? If it depends on a tool installed by another task (e.g. `alembic check` needs Python env from a prior task), ensure that task is in `depends_on`. If no prior task sets up the tool, either make the current task include setup, or create a prerequisite task.
```

---

## Part C: Add atomic commit rule to `rules/harness.md`

Add to the **Git discipline** section in `.claude/rules/harness.md`:

**Add after** the line `- Never merge to develop. Only push and open PR.`:

```markdown
- Each commit must address one concern. Do not bundle unrelated changes in a single commit.
```

---

## Commands

```bash
# Create branch from develop
git checkout develop && git pull
git checkout -b setup/phase-3
```

Then:
1. Replace `.claude/skills/start-to-code/SKILL.md` with the full content above
2. Edit `.claude/skills/plan-from-spec/SKILL.md` — add the verify dependency rule to Step 4
3. Edit `.claude/rules/harness.md` — add atomic commit rule to Git discipline section
4. Commit each change separately (practicing atomic commits!):

```bash
git add .claude/skills/start-to-code/SKILL.md
git commit -m "feat(harness): implement start-to-code skill"

git add .claude/skills/plan-from-spec/SKILL.md
git commit -m "fix(harness): add verify dependency check rule to plan-from-spec"

git add .claude/rules/harness.md
git commit -m "fix(harness): add atomic commit rule to git discipline"

git push -u origin setup/phase-3
```

## Verification

1. `cat .claude/skills/start-to-code/SKILL.md` — full 8-step process with frontmatter
2. `grep "verify dependency" .claude/skills/plan-from-spec/SKILL.md` — new rule present
3. `grep "one concern" .claude/rules/harness.md` — atomic commit rule present
4. `git log --oneline` — 3 separate commits on setup/phase-3
5. Test: start a new Claude Code session and type `/start-to-code` — should recognize the skill

## What's next

Phase 4: Write `smoke-test` skill — environment health checker that adapts to project tech stack.
