# Phase 2: goshared (Minimal)

## Prerequisites
> Phase 0 completed: monorepo scaffold, `pkg/goshared/go.mod` with deps, `go.work` linking modules
> Key files from prior phase: `go.work`, `pkg/goshared/go.mod`

## Decisions
- **BaseRepo**: Depends on `Pool` interface (not concrete `*pgxpool.Pool`) — follows dependency rule
- **Tests**: Unit tests with mock Pool — real DB integration tests deferred to Phase 11
- **Scope**: apperr + db pool + repo layer only. Auth interceptors (Phase 5), UoW/pubsub/config (Phase 9)
> See .claude/specs/02-goshared.md for full library structure and code patterns

## Tasks in this phase
| ID  | Description | Size | File |
|-----|-------------|------|------|
| 2.1 | apperr error types with Unwrap | small | pkg/goshared/apperr/apperr.go |
| 2.2 | Pool interface + Config struct | small | pkg/goshared/db/db.go |
| 2.3 | PgxPool: AfterConnect, PrepareConn, AfterRelease | medium | pkg/goshared/db/pgx/pool.go |
| 2.4 | ListFilter + Page[T] | small | pkg/goshared/repo/filter.go |
| 2.5 | structToColumnsAndValues + field mapping | medium | pkg/goshared/repo/reflect.go |
| 2.6 | BaseRepo[R, ID] CRUD using Pool interface | medium | pkg/goshared/repo/base.go |
| 2.7 | MappedRepo[E, R, ID] with toDomain/toRow | medium | pkg/goshared/repo/mapped.go |
| 2.8 | Unit tests: apperr, BaseRepo, MappedRepo, reflect | medium | pkg/goshared/repo/base_test.go |

## Key patterns

### Error types (apperr)
```go
// pkg/goshared/apperr/apperr.go
type NotFound struct {
    Description string
    Err         error
}
func (e *NotFound) Error() string { return e.Description }
func (e *NotFound) Unwrap() error { return e.Err }

// Same pattern for: AlreadyExists, InvalidInput, Forbidden, Conflict
```

### Pool interface
```go
// pkg/goshared/db/db.go
type Pool interface {
    Acquire(ctx context.Context) (*pgxpool.Conn, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Close()
}

type Config struct {
    Host     string
    Port     int
    User     string
    Password string
    DBName   string
    Schema   string
    PoolSize int
}
```

### PgxPool — tenant-aware connection
```go
// pkg/goshared/db/pgx/pool.go
// AfterConnect: SET search_path TO <schema>
// PrepareConn: SET app.current_tenant = '<slug>'  (from context)
// AfterRelease: RESET app.current_tenant
```

### BaseRepo — generic CRUD
```go
// pkg/goshared/repo/base.go
type BaseRepo[R any, ID comparable] struct {
    pool  db.Pool    // interface, not concrete
    table string
    pk    string
}

func (r *BaseRepo[R, ID]) GetByID(ctx context.Context, id ID) (*R, error)
func (r *BaseRepo[R, ID]) List(ctx context.Context, f ListFilter) (*Page[R], error)
func (r *BaseRepo[R, ID]) Create(ctx context.Context, row *R) (*R, error)
func (r *BaseRepo[R, ID]) Update(ctx context.Context, id ID, row *R) (*R, error)
func (r *BaseRepo[R, ID]) UpdateFields(ctx context.Context, id ID, row *R, fields []string) (*R, error)
func (r *BaseRepo[R, ID]) SoftDelete(ctx context.Context, id ID) error
```

### MappedRepo — domain translation layer
```go
// pkg/goshared/repo/mapped.go
type MappedRepo[E any, R any, ID comparable] struct {
    base     *BaseRepo[R, ID]
    toDomain func(*R) *E
    toRow    func(*E) *R
}
```

### Reflect — struct to columns
```go
// pkg/goshared/repo/reflect.go
// structToColumnsAndValues reads `db:"col"` tags from Row struct
// Skips: tenant, created_at, updated_at, deleted_at
// For UpdateFields: Go field name → db tag → SQL column
```

### Pagination
```go
type ListFilter struct {
    Cursor    *uuid.UUID
    Limit     int32
    Offset    *int32
    PageSize  int32
    SortBy    string
    SortOrder string  // "ASC" | "DESC"
}

type Page[T any] struct {
    Items      []T
    NextCursor *uuid.UUID
    Total      *int64  // only for offset pagination
}
```

## Verification
- `cd pkg/goshared && go build ./apperr/...` — error types compile
- `cd pkg/goshared && go build ./db/...` — pool interface + pgx impl compile
- `cd pkg/goshared && go build ./repo/...` — repo layer compiles
- `cd pkg/goshared && go test ./...` — all unit tests pass

## Dependencies
- Requires: Phase 0 (Go module + deps)
- Blocks: Phase 3 (mastermgmt skeleton uses db.Pool), Phase 4 (tenant CRUD uses BaseRepo/MappedRepo)
