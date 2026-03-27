# 02 — goshared Library

Implemented incrementally (see Implementation Order in 00-architecture-decisions.md).

---

## Structure

```
pkg/goshared/
├── go.mod
│
├── apperr/                      # Pure Go error types — zero external deps
│   └── apperr.go                # NotFound, AlreadyExists, InvalidInput, Forbidden, Conflict
│
├── repo/                        # Generic repository
│   ├── base.go                  # BaseRepo[R, ID] — raw CRUD on Row types (pgx v5)
│   ├── mapped.go                # MappedRepo[E, R, ID] — auto Row ↔ Domain translation
│   ├── filter.go                # ListFilter, Page[T]
│   └── reflect.go               # structToColumnsAndValues (skips tenant, timestamps)
│
├── db/                          # Database pool + UoW
│   ├── db.go                    # Pool interface, Config, UnitOfWork interface
│   └── pgx/
│       ├── pool.go              # PgxPool (AfterConnect, PrepareConn, AfterRelease)
│       └── uow.go               # PgxUnitOfWork (tx + outbox flush)
│
├── config/                      # App configuration
│   ├── config.go                # Loader interface, AppConfig struct
│   └── viper/
│       ├── loader.go            # ViperLoader
│       └── sops.go              # SOPS decryption → merge
│
├── auth/                        # Token validation
│   ├── auth.go                  # TokenValidator interface, Claims struct
│   └── jwx/
│       └── validator.go         # JwxValidator (jwx v3 + JWKS)
│
├── storage/                     # Object storage
│   ├── storage.go               # ObjectStorage interface, UploadParams
│   └── minio/
│       └── minio.go             # MinioStorage (minio-go v7)
│
├── pubsub/                      # Event messaging
│   ├── interfaces/
│   │   └── pubsub.go            # Consumer, Producer interfaces, Message, Config
│   └── franz/
│       └── consumer.go          # FranzConsumer implements pubsub Consumer (franz-go)
│
├── grpc/                        # RPC layer
│   ├── interfaces/
│   │   └── grpc.go              # Interceptor interfaces, handler types
│   └── connect/                 # ConnectRPC adapter
│       ├── interceptor/
│       │   ├── error.go         # apperr types → ConnectRPC status codes
│       │   ├── auth.go          # JWT validation (uses auth.TokenValidator)
│       │   ├── tenant.go        # Extract tenant slug from claims → context
│       │   └── logging.go       # slog structured request logging
│       └── middleware/
│           ├── requestid.go
│           ├── cors.go
│           └── recovery.go
│
├── domain/                      # Shared DDD building blocks
│   ├── aggregate.go             # Base AggregateRoot (AddEvent, Events, ClearEvents)
│   └── event.go                 # DomainEvent interface
│
└── mock/                        # Shared mocks for testing
    ├── pool.go                  # MockPool implements db.Pool
    ├── consumer.go              # MockConsumer implements pubsub Consumer
    ├── storage.go               # MockObjectStorage implements storage.ObjectStorage
    └── validator.go             # MockTokenValidator implements auth.TokenValidator
```

**Dependency rule**: application/domain code imports interfaces (`pubsub/interfaces`, `storage.ObjectStorage`), never implementations (`pubsub/franz`, `storage/minio`). Only Fx providers in `cmd/server/main.go` wire implementations.

---

## Error Types (`apperr/`)

Pure Go — zero external dependencies. Each category is a type with `Error() string` and optional `Unwrap() error` for error chaining.

```go
// pkg/goshared/apperr/apperr.go

type NotFound struct {
    Description string
    Err         error  // optional — supports errors.Is/As through Unwrap chain
}
func (e *NotFound) Error() string { return e.Description }
func (e *NotFound) Unwrap() error { return e.Err }

type AlreadyExists struct {
    Description string
    Err         error
}
func (e *AlreadyExists) Error() string { return e.Description }
func (e *AlreadyExists) Unwrap() error { return e.Err }

type InvalidInput struct {
    Description string
    Err         error
}
func (e *InvalidInput) Error() string { return e.Description }
func (e *InvalidInput) Unwrap() error { return e.Err }

type Forbidden struct {
    Description string
    Err         error
}
func (e *Forbidden) Error() string { return e.Description }
func (e *Forbidden) Unwrap() error { return e.Err }

type Conflict struct {
    Description string
    Err         error
}
func (e *Conflict) Error() string { return e.Description }
func (e *Conflict) Unwrap() error { return e.Err }
```

### Usage

```go
// Simple — Description only
return &apperr.NotFound{Description: "book not found"}

// With wrapped error — preserves original for errors.Is chain
row, err := pool.QueryRow(...)
if errors.Is(err, pgx.ErrNoRows) {
    return &apperr.NotFound{Description: "book not found", Err: err}
}

// Checking works at both levels:
errors.As(err, &apperr.NotFound{})   // ✓ matches the category
errors.Is(err, pgx.ErrNoRows)        // ✓ reaches through Unwrap
```

### Domain Sentinel Errors

```go
// internal/book/domain/errors.go
var ErrBookNotFound     = &apperr.NotFound{Description: "book not found"}
var ErrBookNameRequired = &apperr.InvalidInput{Description: "book name is required"}

// internal/tenant/domain/errors.go
var ErrTenantNotFound  = &apperr.NotFound{Description: "tenant not found"}
var ErrTenantSlugTaken = &apperr.AlreadyExists{Description: "tenant slug already taken"}
```

### Error Interceptor (ConnectRPC-specific)

```go
// pkg/goshared/grpc/connect/interceptor/error.go
func ErrorInterceptor(next connect.UnaryFunc) connect.UnaryFunc {
    return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
        resp, err := next(ctx, req)
        if err == nil { return resp, nil }

        switch {
        case errors.As(err, new(*apperr.NotFound)):
            return nil, connect.NewError(connect.CodeNotFound, err)
        case errors.As(err, new(*apperr.AlreadyExists)):
            return nil, connect.NewError(connect.CodeAlreadyExists, err)
        case errors.As(err, new(*apperr.InvalidInput)):
            return nil, connect.NewError(connect.CodeInvalidArgument, err)
        case errors.As(err, new(*apperr.Forbidden)):
            return nil, connect.NewError(connect.CodePermissionDenied, err)
        case errors.As(err, new(*apperr.Conflict)):
            return nil, connect.NewError(connect.CodeAborted, err)
        default:
            return nil, connect.NewError(connect.CodeInternal, err)
        }
    }
}
```

---

## BaseRepo + MappedRepo (`repo/`)

### BaseRepo — Raw Generic CRUD on Row Types

```go
// pkg/goshared/repo/base.go
type BaseRepo[R any, ID comparable] struct {
    pool  db.Pool       // interface, not concrete *pgxpool.Pool
    table string  // e.g. "courses" (search_path set by AfterConnect)
    pk    string  // e.g. "course_id"
}

func (r *BaseRepo[R, ID]) GetByID(ctx context.Context, id ID) (*R, error)
func (r *BaseRepo[R, ID]) List(ctx context.Context, f ListFilter) (*Page[R], error)
func (r *BaseRepo[R, ID]) Create(ctx context.Context, row *R) (*R, error)
func (r *BaseRepo[R, ID]) Update(ctx context.Context, id ID, row *R) (*R, error)
func (r *BaseRepo[R, ID]) UpdateFields(ctx context.Context, id ID, row *R, fields []string) (*R, error)
func (r *BaseRepo[R, ID]) SoftDelete(ctx context.Context, id ID) error
```

### MappedRepo — Auto Row ↔ Domain Translation

```go
// pkg/goshared/repo/mapped.go
type MappedRepo[E any, R any, ID comparable] struct {
    base     *BaseRepo[R, ID]
    toDomain func(*R) *E
    toRow    func(*E) *R
}

func NewMappedRepo[E any, R any, ID comparable](
    base *BaseRepo[R, ID],
    toDomain func(*R) *E,
    toRow func(*E) *R,
) *MappedRepo[E, R, ID] { ... }

func (r *MappedRepo[E, R, ID]) GetByID(ctx context.Context, id ID) (*E, error)
func (r *MappedRepo[E, R, ID]) Create(ctx context.Context, entity *E) (*E, error)
func (r *MappedRepo[E, R, ID]) Update(ctx context.Context, id ID, entity *E) (*E, error)
func (r *MappedRepo[E, R, ID]) UpdateFields(ctx context.Context, id ID, entity *E, fields []string) (*E, error)
func (r *MappedRepo[E, R, ID]) List(ctx context.Context, f ListFilter) (*Page[*E], error)
func (r *MappedRepo[E, R, ID]) SoftDelete(ctx context.Context, id ID) error
```

### Update Strategies

**`Update`** — full replace (GET → modify → save entire entity):
```go
tenant, _ := repo.GetByID(ctx, id)
tenant.Name = "New Name"
repo.Update(ctx, id, tenant)
```

**`UpdateFields`** — partial update with Go struct field names as mask:
```go
tenant.Name = "New Name"
repo.UpdateFields(ctx, id, tenant, []string{"Name"})
// Reflection: Go field "Name" → db tag "name" → SQL column
```

### Pagination

```go
type ListFilter struct {
    Cursor    *uuid.UUID
    Limit     int32
    Offset    *int32
    PageSize  int32
    SortBy    string
    SortOrder string       // "ASC" | "DESC"
}

type Page[T any] struct {
    Items      []T
    NextCursor *uuid.UUID
    Total      *int64      // only for offset pagination
}
```

### Reflection (`reflect.go`)

`structToColumnsAndValues` reads `db:"col"` tags from Row struct. Skips: `tenant`, `created_at`, `updated_at`, `deleted_at`.

For `UpdateFields`: maps Go struct field names → `db` tag values → SQL column names.

### Interceptor Chain

```
requestID → logging → auth → tenant → error → validate → handler
```

---

## Tasks (Ordered by Vertical-First)

### Minimal (Phase 2 — before first feature)

- [ ] **2.1** `apperr/apperr.go` — NotFound, AlreadyExists, InvalidInput, Forbidden, Conflict (with Unwrap)
- [ ] **2.2** `db/db.go` — Pool interface, Config struct
- [ ] **2.3** `db/pgx/pool.go` — PgxPool: AfterConnect (search_path), PrepareConn (tenant slug), AfterRelease (RESET)
- [ ] **2.4** `repo/base.go` — BaseRepo[R, ID] (CRUD, skips tenant column)
- [ ] **2.5** `repo/mapped.go` — MappedRepo[E, R, ID]
- [ ] **2.6** `repo/filter.go` — ListFilter, Page[T]
- [ ] **2.7** `repo/reflect.go` — structToColumnsAndValues + UpdateFields mapping
- [ ] **2.8** Unit tests (BaseRepo, MappedRepo, reflection, UpdateFields)

### Auth (Phase 5 — after first CRUD works)

- [ ] **2.9** `auth/auth.go` — TokenValidator interface, Claims struct
- [ ] **2.10** `auth/jwx/validator.go` — JwxValidator (jwx v3, JWKS)
- [ ] **2.11** `grpc/interfaces/grpc.go` — interceptor interfaces
- [ ] **2.12** `grpc/connect/interceptor/auth.go` — JWT validation
- [ ] **2.13** `grpc/connect/interceptor/tenant.go` — extract tenant slug → context
- [ ] **2.14** `grpc/connect/interceptor/error.go` — apperr → ConnectRPC codes
- [ ] **2.15** `grpc/connect/interceptor/logging.go` — slog
- [ ] **2.16** `grpc/connect/middleware/` — requestID, CORS, recovery

### Remaining (Phase 9 — when CDC/outbox phase starts)

- [ ] **2.17** `db/pgx/uow.go` — UoW (tx + outbox flush)
- [ ] **2.18** `config/config.go` + `config/viper/` — Loader + SOPS
- [ ] **2.19** `pubsub/interfaces/pubsub.go` — Consumer/Producer interfaces
- [ ] **2.20** `pubsub/franz/consumer.go` — FranzConsumer
- [ ] **2.21** `storage/storage.go` + `storage/minio/minio.go`
- [ ] **2.22** `domain/aggregate.go` + `domain/event.go`
- [ ] **2.23** `mock/` — MockPool, MockConsumer, MockObjectStorage, MockTokenValidator
- [ ] **2.24** Unit tests for remaining components
