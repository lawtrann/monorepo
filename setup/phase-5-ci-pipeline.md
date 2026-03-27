# Phase 5: CI Pipeline (`task-ci.yml`)

## Goal

Replace the CI placeholder with a working GitHub Actions workflow that:
1. Auto-detects project stack (Go, Node, Python)
2. Runs appropriate build, test, lint checks
3. On all checks pass: parses branch name → marks task as passed in tasks.json → auto-commits

## Context

- Branch naming convention: `task/{id}_{slug}` (e.g. `task/0.1_root-scaffold`)
- CI parses `{id}` from branch name to know which task to mark
- Uses `GITHUB_TOKEN` (default) — commits from it won't trigger CI again (avoids infinite loop)
- Can upgrade to GitHub App token later if needed
- Generic template — detects stack, not hardcoded for any language

## File Contents

### .github/workflows/task-ci.yml

Replace the placeholder entirely with:

```yaml
name: Task CI

on:
  push:
    branches: ['task/**']

permissions:
  contents: write

jobs:
  detect:
    name: Detect stack
    runs-on: ubuntu-latest
    outputs:
      has_go: ${{ steps.detect.outputs.has_go }}
      has_node: ${{ steps.detect.outputs.has_node }}
      has_python: ${{ steps.detect.outputs.has_python }}
      has_proto: ${{ steps.detect.outputs.has_proto }}
      has_docker: ${{ steps.detect.outputs.has_docker }}
      go_modules: ${{ steps.detect.outputs.go_modules }}
    steps:
      - uses: actions/checkout@v4

      - id: detect
        name: Detect project stack
        run: |
          # Go modules
          go_mods=$(find . -name "go.mod" -not -path "*/vendor/*" | sort)
          if [ -n "$go_mods" ]; then
            echo "has_go=true" >> "$GITHUB_OUTPUT"
            # JSON array of module directories
            echo "go_modules=$(echo "$go_mods" | xargs -I{} dirname {} | jq -R -s -c 'split("\n") | map(select(. != ""))')" >> "$GITHUB_OUTPUT"
          else
            echo "has_go=false" >> "$GITHUB_OUTPUT"
            echo "go_modules=[]" >> "$GITHUB_OUTPUT"
          fi

          # Node
          if [ -f "package.json" ] || find . -name "package.json" -maxdepth 3 | grep -q .; then
            echo "has_node=true" >> "$GITHUB_OUTPUT"
          else
            echo "has_node=false" >> "$GITHUB_OUTPUT"
          fi

          # Python
          if find . -name "pyproject.toml" -o -name "requirements.txt" | grep -q .; then
            echo "has_python=true" >> "$GITHUB_OUTPUT"
          else
            echo "has_python=false" >> "$GITHUB_OUTPUT"
          fi

          # Proto (buf)
          if find . -name "buf.yaml" | grep -q .; then
            echo "has_proto=true" >> "$GITHUB_OUTPUT"
          else
            echo "has_proto=false" >> "$GITHUB_OUTPUT"
          fi

          # Docker
          if [ -f "docker-compose.yml" ] || [ -f "docker-compose.yaml" ]; then
            echo "has_docker=true" >> "$GITHUB_OUTPUT"
          else
            echo "has_docker=false" >> "$GITHUB_OUTPUT"
          fi

  go:
    name: Go checks
    needs: detect
    if: needs.detect.outputs.has_go == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: ${{ fromJson(needs.detect.outputs.go_modules)[0] }}/go.mod

      - name: Build all modules
        run: |
          modules='${{ needs.detect.outputs.go_modules }}'
          echo "$modules" | jq -r '.[]' | while read -r dir; do
            echo "::group::Build $dir"
            cd "$GITHUB_WORKSPACE/$dir"
            go build ./...
            echo "::endgroup::"
          done

      - name: Test all modules
        run: |
          modules='${{ needs.detect.outputs.go_modules }}'
          echo "$modules" | jq -r '.[]' | while read -r dir; do
            # Only test if test files exist
            if find "$GITHUB_WORKSPACE/$dir" -name "*_test.go" | grep -q .; then
              echo "::group::Test $dir"
              cd "$GITHUB_WORKSPACE/$dir"
              go test ./... -count=1 -timeout=300s
              echo "::endgroup::"
            else
              echo "No test files in $dir — skipping"
            fi
          done

      - name: Vet all modules
        run: |
          modules='${{ needs.detect.outputs.go_modules }}'
          echo "$modules" | jq -r '.[]' | while read -r dir; do
            echo "::group::Vet $dir"
            cd "$GITHUB_WORKSPACE/$dir"
            go vet ./...
            echo "::endgroup::"
          done

  node:
    name: Node checks
    needs: detect
    if: needs.detect.outputs.has_node == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 'lts/*'

      - name: Install and lint
        run: |
          # Find package.json dirs and run checks
          find . -name "package.json" -maxdepth 3 -not -path "*/node_modules/*" | while read -r pkg; do
            dir=$(dirname "$pkg")
            echo "::group::Node checks in $dir"
            cd "$GITHUB_WORKSPACE/$dir"
            if [ -f "pnpm-lock.yaml" ]; then
              npm install -g pnpm && pnpm install --frozen-lockfile
            elif [ -f "package-lock.json" ]; then
              npm ci
            fi
            # Lint if script exists
            if grep -q '"lint"' package.json; then
              npm run lint
            fi
            echo "::endgroup::"
          done

  python:
    name: Python checks
    needs: detect
    if: needs.detect.outputs.has_python == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'

      - name: Check Python projects
        run: |
          find . -name "pyproject.toml" | while read -r toml; do
            dir=$(dirname "$toml")
            echo "::group::Python checks in $dir"
            cd "$GITHUB_WORKSPACE/$dir"
            pip install -e ".[dev]" 2>/dev/null || pip install -e . 2>/dev/null || pip install -r requirements.txt 2>/dev/null || true
            echo "::endgroup::"
          done

  proto:
    name: Proto lint
    needs: detect
    if: needs.detect.outputs.has_proto == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: bufbuild/buf-action@v1
        with:
          setup_only: true

      - name: Buf lint
        run: |
          find . -name "buf.yaml" | while read -r buf; do
            dir=$(dirname "$buf")
            echo "::group::buf lint $dir"
            cd "$GITHUB_WORKSPACE/$dir"
            buf lint
            echo "::endgroup::"
          done

  mark-passed:
    name: Mark task passed
    needs: [detect, go, node, python, proto]
    if: always() && !contains(needs.*.result, 'failure') && !contains(needs.*.result, 'cancelled')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ github.ref }}
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Parse task ID from branch
        id: parse
        run: |
          branch="${GITHUB_REF_NAME}"
          # Branch format: task/{id}_{slug}
          # Extract everything between task/ and first _
          task_id=$(echo "$branch" | sed -n 's|^task/\([0-9]*\.[0-9]*\)_.*|\1|p')
          if [ -z "$task_id" ]; then
            echo "Could not parse task ID from branch: $branch"
            echo "Expected format: task/{id}_{slug}"
            exit 1
          fi
          echo "task_id=$task_id" >> "$GITHUB_OUTPUT"
          echo "Parsed task ID: $task_id"

      - name: Mark task as passed in tasks.json
        run: |
          task_id="${{ steps.parse.outputs.task_id }}"
          
          # Check if task exists and is not already passed
          current=$(jq --arg id "$task_id" '.[] | select(.id == $id) | .passes' .claude/tasks.json)
          
          if [ "$current" = "true" ]; then
            echo "Task $task_id is already marked as passed. Skipping."
            exit 0
          fi
          
          if [ -z "$current" ]; then
            echo "Task $task_id not found in tasks.json. Skipping."
            exit 0
          fi
          
          # Update passes to true
          jq --arg id "$task_id" '
            map(if .id == $id then .passes = true else . end)
          ' .claude/tasks.json > tmp.json && mv tmp.json .claude/tasks.json
          
          echo "Marked task $task_id as passed"

      - name: Commit and push
        run: |
          task_id="${{ steps.parse.outputs.task_id }}"
          
          # Check if there are changes to commit
          if git diff --quiet .claude/tasks.json; then
            echo "No changes to tasks.json. Skipping commit."
            exit 0
          fi
          
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add .claude/tasks.json
          git commit -m "ci: mark task $task_id passed"
          git push
```

## Setup Requirements

For the CI to work, the repo needs:

1. **Branch protection on `develop`** — require PR, require status checks
2. **No additional secrets needed** — `GITHUB_TOKEN` is automatic
3. **Note**: Commits from `GITHUB_TOKEN` do NOT trigger workflows — this prevents infinite CI loops

### Future upgrade to GitHub App token

If you later need CI commits to trigger other workflows, replace `GITHUB_TOKEN` with a GitHub App:

1. Create a GitHub App with `contents: write` permission
2. Install it on the repo
3. Add `APP_ID` and `APP_PRIVATE_KEY` as repo secrets
4. Replace the checkout step in `mark-passed` job:
```yaml
      - uses: actions/create-github-app-token@v1
        id: app-token
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}

      - uses: actions/checkout@v4
        with:
          ref: ${{ github.ref }}
          token: ${{ steps.app-token.outputs.token }}
```

## Commands

```bash
git checkout develop && git pull
git checkout -b setup/phase-5

# Replace .github/workflows/task-ci.yml with content above

git add .github/workflows/task-ci.yml
git commit -m "feat(ci): implement task CI with auto-detection and mark-passed"

git push -u origin setup/phase-5
```

## Verification

1. `cat .github/workflows/task-ci.yml` — full workflow with detect + go + node + python + proto + mark-passed jobs
2. Workflow triggers on `push` to `task/**` branches
3. `mark-passed` job runs only if all checks pass (`if: always() && !contains(needs.*.result, 'failure')`)
4. Branch parsing extracts task ID correctly from `task/0.1_root-scaffold` → `0.1`
5. `jq` command correctly sets `passes: true` for matching task ID

### Manual test (after merge)

1. Create a test branch: `git checkout -b task/0.1_root-scaffold`
2. Push to trigger CI: `git push -u origin task/0.1_root-scaffold`
3. Check Actions tab — workflow should run detect + relevant checks
4. If checks pass — `mark-passed` job should commit "ci: mark task 0.1 passed"
5. Pull and verify: `git pull && jq '.[0].passes' .claude/tasks.json` → `true`

## What's next

Harness template is complete! All 5 phases done:
- Phase 1: Repo structure + CLAUDE.md + rules
- Phase 2: plan-from-spec skill
- Phase 3: start-to-code skill + dry-run fixes
- Phase 4: smoke-test skill
- Phase 5: CI pipeline

Next step: Mark repo as GitHub template, instantiate for your project, drop a spec, and start building.
