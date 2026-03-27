# Harness rules

## Task discipline
- Never edit task descriptions, depends_on, or verify fields in tasks.json. Only CI or explicit human override changes the `passes` field.
- Never start a second task in the same session. One task per session, no exceptions.
- If a task is too large for one session, stop and mark BLOCKED in progress — do not half-finish.

## Git discipline
- Always create branch `task/{id}_{slug}` before coding. Never commit directly to develop.
- Always write a descriptive commit message referencing the task ID.
- Never merge to develop. Only push and open PR.
- Each commit must address one concern. Do not bundle unrelated changes in a single commit.

## Session discipline
- Always update `.claude/progress/latest.md` before ending a session.
- If smoke-test fails at session start, fix the failure before starting new work.
- If stuck and unable to fix, mark BLOCKED in progress and stop. Do not loop endlessly.

## Planning discipline
- When running plan-from-spec, never write code. Planning sessions produce only tasks.json entries and phase files.
- Each task in tasks.json must be sized to fit one session. If unsure, split into smaller tasks.
