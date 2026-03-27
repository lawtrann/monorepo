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
