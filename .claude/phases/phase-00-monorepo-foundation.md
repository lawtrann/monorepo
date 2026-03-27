# Phase 0: Monorepo Foundation

## Prerequisites
> None — this is the first phase. Starting from empty repo.

## Decisions
- **Go workspace**: `go.work` at repo root linking all Go modules (goshared, mastermgmt, eureka)
- **Tooling**: `make setup` target installs Buf, sqlc, air (`go install`), SOPS + age (`brew`)
- **SOPS**: Set up now — generate age key, encrypt placeholder secrets.yaml per app
- **Proto**: Placeholder `health.proto` per service so `buf lint` is verifiable
- **Web**: `app/web/` is a placeholder (README + .gitkeep) — Next.js setup deferred
- **Migration**: Empty dirs with `.gitkeep` — Alembic setup deferred to Phase 1 (needs running DB)
> See .claude/specs/01-monorepo-foundation.md for full directory structure

## Tasks in this phase
| ID   | Description | Size | File |
|------|-------------|------|------|
| 0.1  | Root scaffold: .gitignore, README.md, root Makefile | small | Makefile |
| 0.2  | go.work + pkg/goshared go.mod with pinned deps | small | go.work |
| 0.3  | app/mastermgmt Go module with cmd stubs | small | app/mastermgmt/go.mod |
| 0.4  | app/eureka Go module with cmd stubs | small | app/eureka/go.mod |
| 0.5  | Empty migration dirs with .gitkeep | small | app/mastermgmt/migration/.gitkeep |
| 0.6  | Proto dirs with buf.yaml, buf.gen.yaml, health.proto | small | app/mastermgmt/proto/buf.yaml |
| 0.7  | deployment/ directory tree with local/ env dirs | small | deployment/ |
| 0.8  | make setup target for tooling | medium | Makefile |
| 0.9  | SOPS + age key setup, encrypt secrets.yaml | medium | .sops.yaml |
| 0.10 | app/web/ placeholder | small | app/web/README.md |
| 0.11 | Verify buf lint passes for both services | small | — |

## Key patterns

### Monorepo directory structure
```
monorepo/
├── go.work
├── Makefile
├── .gitignore
├── .age-key.txt              # gitignored
├── .sops.yaml
├── deployment/
│   ├── mastermgmt/local/     # config.yaml + secrets.yaml (encrypted)
│   ├── eureka/local/
│   └── casdoor/local/
├── pkg/
│   ├── goshared/             # go.mod — shared Go library
│   ├── go/                   # generated proto (Go) — future
│   └── ts/                   # generated proto (TS) — future
├── app/
│   ├── mastermgmt/           # go.mod — imports pkg/goshared
│   │   ├── cmd/server/main.go
│   │   ├── cmd/worker/main.go
│   │   ├── proto/            # buf.yaml + buf.gen.yaml + health.proto
│   │   └── migration/        # .gitkeep (Alembic deferred to Phase 1)
│   ├── eureka/               # same structure as mastermgmt
│   └── web/                  # README.md + .gitkeep
└── README.md
```

### go.work
Task 0.2 creates `go.work` with only `./pkg/goshared`. Tasks 0.3 and 0.4 each add their module via `go work use ./app/mastermgmt` and `go work use ./app/eureka`.

```go
// After task 0.2:
go 1.24

use (
    ./pkg/goshared
)

// After tasks 0.3 and 0.4, final state:
go 1.24

use (
    ./pkg/goshared
    ./app/mastermgmt
    ./app/eureka
)
```

### Go pinned dependencies (for pkg/goshared/go.mod)
```
connectrpc.com/connect v1.19.1
connectrpc.com/validate v0.3.0
github.com/jackc/pgx/v5 v5.8.0
go.uber.org/fx v1.23.0
github.com/spf13/viper v1.21.0
github.com/lestrrat-go/jwx/v3 (latest)
github.com/minio/minio-go/v7 v7.0.83
github.com/twmb/franz-go v1.18.0
github.com/stretchr/testify v1.10.0
github.com/google/uuid v1.6.0
google.golang.org/protobuf v1.36.5
```

### buf.yaml (per service)
```yaml
version: v2
modules:
  - path: .
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

### Makefile setup target
```makefile
.PHONY: setup
setup:
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/air-verse/air@latest
	@echo "Install SOPS and age via brew:"
	brew install sops age
```

## Verification
- `go work sync` — all modules resolve
- `cd app/mastermgmt && go build ./...` — Go module compiles
- `cd app/eureka && go build ./...` — Go module compiles
- `cd app/mastermgmt/proto && buf lint` — proto lints clean
- `make setup && which buf && which sqlc` — tools installed
- `sops -d deployment/mastermgmt/local/secrets.yaml > /dev/null` — SOPS decryption works

## Dependencies
- Requires: None
- Blocks: Phase 1 (Casdoor setup needs docker-compose + deployment dirs), Phase 2 (goshared needs Go module)
