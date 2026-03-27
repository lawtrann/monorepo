# Phase 4: Skill `smoke-test`

## Goal

Write the full `smoke-test` skill — environment health checker that adapts to the project's tech stack. Called by `start-to-code` Step 5, or manually via `/smoke-test`.

## Context

- `start-to-code` Step 5 calls `/smoke-test` — needs clear pass/fail signal
- Skill must be generic (template repo) — detects what exists and checks accordingly
- Output is detailed: each check listed with pass/fail + error on failure
- `disable-model-invocation: true` — only called explicitly

## Skill structure

```
.claude/skills/smoke-test/
└── SKILL.md          # Tier 1 (frontmatter) + Tier 2 (instructions)
```

No supporting files — the skill auto-detects what to check based on filesystem.

## File Contents

### .claude/skills/smoke-test/SKILL.md

Replace the placeholder entirely with:

```markdown
---
name: smoke-test
description: >
  Verify environment health before starting new work.
  Runs build checks, existing tests, and service health checks.
  Adapts to the project's tech stack and infrastructure.
  If any check fails, reports the failure for fixing before new work begins.
disable-model-invocation: true
---

# Smoke test

Verify environment health. Detect what exists in the project and run appropriate checks.
Report each check individually with pass/fail and error details.

## Invocation

```
/smoke-test
```

No arguments. Auto-detects checks based on filesystem.

## Process

### Step 1: Detect project stack

Scan the repo to determine which checks apply:

| If this exists | Then check |
|---|---|
| `**/go.mod` | Go build + Go test for that module |
| `**/package.json` | npm/pnpm install status + lint |
| `**/pyproject.toml` or `**/requirements.txt` | Python env exists + deps installed |
| `docker-compose.yml` | Services running (`docker-compose ps`) |
| `**/alembic.ini` | Alembic migrations up to date |
| `**/buf.yaml` | `buf lint` passes |
| `Makefile` | `make` is available |

Only check what exists. Early-phase projects may have very few checks — that's fine.

### Step 2: Run checks in order

Run checks in this priority order (most fundamental first):

1. **Git status** — is working tree clean? (warn only, don't fail)
2. **Docker services** — if `docker-compose.yml` exists and project expects services
3. **Build checks** — compile each module that has a build system
4. **Test checks** — run existing tests for each module (only if test files exist)
5. **Lint checks** — buf lint, golangci-lint, etc. (only if configured)
6. **Migration checks** — Alembic current head matches (only if DB is running)

For each check, capture:
- Check name
- Command run
- Pass or fail
- Error output (first 20 lines) if failed

### Step 3: Report results

Present results as a checklist:

```
Smoke test results:

[PASS] Git status — working tree clean
[PASS] Docker services — db, minio running (3/3 healthy)
[PASS] Go build — pkg/goshared compiles
[PASS] Go build — app/mastermgmt compiles
[FAIL] Go test — app/mastermgmt: 2 tests failed
       → TestCreateTenant: expected status 200, got 500
       → TestListLocations: connection refused
[SKIP] Alembic — no DB connection available
[SKIP] Buf lint — no .proto files yet

Result: 4 passed, 1 failed, 2 skipped
```

### Step 4: Verdict

Based on results:

- **All pass (or pass + skip)** → Report "Smoke test passed. Environment is healthy." and return.
- **Any fail** → Report failures with details. Do NOT attempt to fix automatically. The caller (start-to-code or human) decides what to do.

## Check details

### Go modules
```bash
# Find all go.mod files
find . -name "go.mod" -not -path "*/vendor/*"

# For each module directory:
cd <module-dir>
go build ./...          # build check
go test ./... 2>&1      # test check (only if *_test.go files exist)
```

### Docker services
```bash
# Only if docker-compose.yml exists
docker-compose ps --format json 2>/dev/null
# Check: are expected services running and healthy?
```

### Alembic migrations
```bash
# Only if alembic.ini exists AND DB is reachable
cd <migration-dir>
alembic check 2>&1
# "No new upgrade operations detected" = pass
```

### Buf lint
```bash
# Only if buf.yaml exists AND .proto files exist
cd <proto-dir>
buf lint 2>&1
```

## Important rules

- NEVER fix failures automatically. Only detect and report.
- NEVER skip checks silently. If a check is skipped, report it as [SKIP] with reason.
- Keep error output concise — first 20 lines max per failure.
- If a check command hangs (>30 seconds), kill it and report as [FAIL] with timeout.
- This skill is read-only — it does not modify any files.
```

## Commands

```bash
git checkout develop && git pull
git checkout -b setup/phase-4

# Replace smoke-test SKILL.md placeholder
# (write the content above into .claude/skills/smoke-test/SKILL.md)

git add .claude/skills/smoke-test/SKILL.md
git commit -m "feat(harness): implement smoke-test skill with auto-detection"

git push -u origin setup/phase-4
```

## Verification

1. `cat .claude/skills/smoke-test/SKILL.md` — full content with frontmatter + 4 steps + check details
2. `grep "disable-model-invocation: true" .claude/skills/smoke-test/SKILL.md` — present
3. `grep "NEVER fix failures" .claude/skills/smoke-test/SKILL.md` — read-only rule present
4. Test: type `/smoke-test` in Claude Code — should recognize skill and run detection

## What's next

Phase 5: CI pipeline (`task-ci.yml`) — the automated verification + task marking system.
