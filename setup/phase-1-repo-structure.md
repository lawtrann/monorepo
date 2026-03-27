# Phase 1: Repo Structure + CLAUDE.md + Rules

## Context

Setting up `harness-template` — a GitHub template repo for agent-driven incremental development with Claude Code. Phase 1 creates the foundation skeleton: directories, placeholder files, rules, and config. No logic or code — just infrastructure.

This file is the finalized, aligned plan — it replaces the original draft spec.

## Aligned Decisions

- **Default branch**: Rename `main` → `develop` immediately after `git init`. `develop` is the protected integration branch.
- **Work branch**: Create `setup/phase-1` from `develop` → commit there → push → open PR to develop. Never directly to develop.
- **`passes` rule**: Softened to "Only CI or explicit human override changes the `passes` field."
- **Task edit rule**: Kept strict — "Never edit task descriptions, depends_on, or verify fields in tasks.json."
- **CLAUDE.md**: Persistent context + harness pointers. No contradictory "NOT file pointers" comment.
- **.gitignore**: Minimal OS/IDE entries only (no language-specific entries).
- **setup/**: Git-tracked folder for phase specs as living documentation.

## Implementation Steps

### 1. Initialize git + set up remote + develop branch
```bash
git init
git remote add origin https://github.com/lawtrann/harness-template.git
git branch -M develop             # rename default branch to develop
```
Need an initial commit on develop before pushing (git won't push an empty branch):
- Create a minimal initial commit (e.g., just `.gitignore` or empty README)
- Push develop: `git push -u origin develop`

### 1b. Create work branch
```bash
git checkout -b setup/phase-1     # work branch off develop
```

### 2. Create directory structure
```
harness-template/
├── CLAUDE.md
├── README.md
├── .gitignore
├── setup/
│   └── phase-1-repo-structure.md
├── .claude/
│   ├── rules/
│   │   └── harness.md
│   ├── skills/
│   │   ├── plan-from-spec/
│   │   │   └── SKILL.md
│   │   ├── start-to-code/
│   │   │   └── SKILL.md
│   │   └── smoke-test/
│   │       └── SKILL.md
│   ├── specs/                       ← .gitkeep
│   ├── phases/                      ← .gitkeep
│   ├── progress/
│   │   └── latest.md
│   └── tasks.json
└── .github/
    └── workflows/
        └── task-ci.yml
```

### 3. File contents

#### CLAUDE.md
```markdown
# Project

> Replace this section when using the template.

Tech stack: (describe after template instantiation)
Monorepo structure: (describe after template instantiation)

## Harness

This project uses a harness system for incremental, agent-driven development.

- Tasks and progress are tracked in `.claude/tasks.json`
- Phase context is in `.claude/phases/`
- Specs (design plans, feature requests) go in `.claude/specs/`
- Session logs are in `.claude/progress/`

Skills available: `/plan-from-spec`, `/start-to-code`, `/smoke-test`
```

#### .claude/rules/harness.md
```markdown
# Harness rules

## Task discipline
- Never edit task descriptions, depends_on, or verify fields in tasks.json. Only CI or explicit human override changes the `passes` field.
- Never start a second task in the same session. One task per session, no exceptions.
- If a task is too large for one session, stop and mark BLOCKED in progress — do not half-finish.

## Git discipline
- Always create branch `task/{id}_{slug}` before coding. Never commit directly to develop.
- Always write a descriptive commit message referencing the task ID.
- Never merge to develop. Only push and open PR.

## Session discipline
- Always update `.claude/progress/latest.md` before ending a session.
- If smoke-test fails at session start, fix the failure before starting new work.
- If stuck and unable to fix, mark BLOCKED in progress and stop. Do not loop endlessly.

## Planning discipline
- When running plan-from-spec, never write code. Planning sessions produce only tasks.json entries and phase files.
- Each task in tasks.json must be sized to fit one session. If unsure, split into smaller tasks.
```

#### .claude/tasks.json
```json
[]
```

#### .claude/progress/latest.md
```markdown
# Latest session

No sessions yet. Run `/plan-from-spec` to begin planning from a spec.
```

#### .claude/skills/plan-from-spec/SKILL.md
```markdown
---
name: plan-from-spec
description: >
  Transform a spec document into actionable tasks and phase files.
  Use when a new spec is placed in .claude/specs/ and needs to be
  broken down into tasks.json entries and .claude/phases/ context files.
  Planning only — never writes code.
---

<!-- Full instructions will be added in Phase 2 -->
```

#### .claude/skills/start-to-code/SKILL.md
```markdown
---
name: start-to-code
description: >
  Resume work at the start of a coding session. Reads progress,
  picks the next available task based on dependency order,
  loads phase context, and runs smoke test before implementation.
  Also handles continuing a task left in progress.
---

<!-- Full instructions will be added in Phase 3 -->
```

#### .claude/skills/smoke-test/SKILL.md
```markdown
---
name: smoke-test
description: >
  Verify environment health before starting new work.
  Runs build checks, existing tests, and service health checks.
  Adapts to the project's tech stack and infrastructure.
  If any check fails, reports the failure for fixing before new work begins.
---

<!-- Full instructions will be added in Phase 4 -->
```

#### .github/workflows/task-ci.yml
```yaml
# Placeholder — will be implemented in Phase 5
# CI pipeline: triggered on push to task/* branches
# Steps: build, lint, test, integration test, mark passes in tasks.json
name: Task CI
on:
  push:
    branches: ['task/**']
jobs:
  placeholder:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "CI pipeline not yet configured"
```

#### .gitignore
```gitignore
# OS
.DS_Store
Thumbs.db

# IDE
.vscode/
.idea/
*.swp
*.swo
```

#### README.md
```markdown
# harness-template

A GitHub template repo for agent-driven incremental development with Claude Code.

## Usage

1. Click "Use this template" on GitHub to create a new repo
2. Clone the new repo and open with Claude Code
3. Update `CLAUDE.md` with your project identity and tech stack
4. Drop a spec into `.claude/specs/`
5. Run `/plan-from-spec` to generate tasks and phase context
6. Run `/start-to-code` to begin coding

## Structure

| Directory | Purpose |
|---|---|
| `.claude/specs/` | Input: design plans, feature requests |
| `.claude/tasks.json` | Tasks with dependencies and status |
| `.claude/phases/` | Per-phase context for the coding agent |
| `.claude/progress/` | Session logs (latest.md + history) |
| `.claude/skills/` | plan-from-spec, start-to-code, smoke-test |
| `.claude/rules/` | Hard constraints for agent behavior |

## Workflow

**Plan** → drop spec → `/plan-from-spec` → tasks.json + phases/
**Code** → `/start-to-code` → implement 1 task → push PR
**Verify** → CI tests → CI marks task passed → human reviews → merge
```

### 4. Commit + Push + PR
```bash
git add -A
git commit -m "chore: initialize harness-template with foundation structure"
git push -u origin setup/phase-1
gh pr create --base develop --title "chore: initialize harness-template foundation" --body "..."
```

## Verification

1. `git branch` → shows `setup/phase-1` as current branch
2. `cat .claude/tasks.json` → `[]`
3. `cat .claude/skills/plan-from-spec/SKILL.md` → YAML frontmatter with name + description
4. `cat .claude/rules/harness.md` → all rules with softened `passes` rule
5. `ls .claude/specs/` → `.gitkeep`
6. `ls setup/` → `phase-1-repo-structure.md`
7. `git log --oneline` → single commit on `setup/phase-1`
8. PR created targeting `develop` branch on GitHub

## What's next

Phase 2: Write full `plan-from-spec` SKILL.md — the core skill that transforms specs into tasks and phase files.
