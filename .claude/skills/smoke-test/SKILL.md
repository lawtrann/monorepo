---
name: smoke-test
description: >
  Verify environment health before starting new work.
  Runs build checks, existing tests, and service health checks.
  Adapts to the project's tech stack and infrastructure.
  If any check fails, reports the failure for fixing before new work begins.
disable-model-invocation: false
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

| If this exists                               | Then check                             |
| -------------------------------------------- | -------------------------------------- |
| `**/go.mod`                                  | Go build + Go test for that module     |
| `**/package.json`                            | npm/pnpm install status + lint         |
| `**/pyproject.toml` or `**/requirements.txt` | Python env exists + deps installed     |
| `docker-compose.yml`                         | Services running (`docker-compose ps`) |
| `**/alembic.ini`                             | Alembic migrations up to date          |
| `**/buf.yaml`                                | `buf lint` passes                      |
| `Makefile`                                   | `make` is available                    |

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

### Resolve repo root first

Always determine the absolute repo root before running any checks. Use it for all `cd` commands so checks work regardless of the shell's current working directory.

```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
```

### Go modules

```bash
# Find all go.mod files (run from REPO_ROOT)
find "$REPO_ROOT" -name "go.mod" -not -path "*/vendor/*"

# For each module directory — always use absolute path:
cd "$REPO_ROOT/<module-dir>"
go build ./...          # build check
go test ./... 2>&1      # test check (only if *_test.go files exist)
```

### Docker services

```bash
# Only if docker-compose.yml exists
cd "$REPO_ROOT" && docker compose ps 2>/dev/null
# Check: are expected services running and healthy?
```

### Alembic migrations

```bash
# Only if alembic.ini exists AND DB is reachable
cd "$REPO_ROOT/<migration-dir>"
uv run alembic check 2>&1
# "No new upgrade operations detected" = pass
```

### Buf lint

```bash
# Only if buf.yaml exists AND .proto files exist
cd "$REPO_ROOT/<proto-dir>"
buf lint 2>&1
```

## Important rules

- NEVER fix failures automatically. Only detect and report.
- NEVER skip checks silently. If a check is skipped, report it as [SKIP] with reason.
- Keep error output concise — first 20 lines max per failure.
- If a check command hangs (>30 seconds), kill it and report as [FAIL] with timeout.
- This skill is read-only — it does not modify any files.
