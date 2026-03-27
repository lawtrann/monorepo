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
