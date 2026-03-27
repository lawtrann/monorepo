# 05 — Eureka Service

Educational Content — Books.

---

## Database Schema

### eureka.books

| Column | Type | Notes |
|---|---|---|
| `book_id` | `UUID v7` | PK |
| `name` | `TEXT NOT NULL` | |
| `tenant` | `TEXT NOT NULL` | `DEFAULT current_setting('app.current_tenant')` |
| `created_at` | `TIMESTAMPTZ` | DEFAULT now() |
| `updated_at` | `TIMESTAMPTZ` | DEFAULT now() |
| `deleted_at` | `TIMESTAMPTZ` | NULL = active |

### eureka.users (CDC mirror)

| Column | Type | Notes |
|---|---|---|
| `user_id` | `UUID` | PK (from Casdoor) |
| `username` | `TEXT` | |
| `email` | `TEXT` | |
| `org` | `TEXT` | Casdoor org = tenant slug |
| `is_active` | `BOOLEAN` | |
| `synced_at` | `TIMESTAMPTZ` | |

### eureka.courses (CDC mirror from mastermgmt)

| Column | Type | Notes |
|---|---|---|
| `course_id` | `UUID v7` | PK (from mastermgmt) |
| `name` | `TEXT` | |
| `tenant` | `TEXT` | |
| `synced_at` | `TIMESTAMPTZ` | |

### eureka.outbox

Same structure as mastermgmt.outbox.

### RLS

```sql
ALTER TABLE eureka.books ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON eureka.books
    USING (tenant = current_setting('app.current_tenant'));
```

---

## sqlc + Type Alias

```yaml
# app/eureka/sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migration/alembic/sql/"
    queries: "internal/queries/"
    gen:
      go:
        package: "sqlgen"
        out: "internal/sqlgen"
```

```go
// internal/book/infra/row.go
package infra

import "app/eureka/internal/sqlgen"

type BookRow = sqlgen.Book  // type alias

func ToDomain(r *BookRow) *domain.Book {
    return &domain.Book{ID: r.BookID, Name: r.Name}
}

func ToRow(e *domain.Book) *BookRow {
    return &BookRow{BookID: e.ID, Name: e.Name}
    // tenant, timestamps — not set. DB handles them.
}
```

---

## Domain Entity

```go
// internal/book/domain/entity.go
type Book struct {
    ID   uuid.UUID
    Name string
}
type Books []*Book

// internal/book/domain/errors.go
var ErrBookNotFound     = &apperr.NotFound{Description: "book not found"}
var ErrBookNameRequired = &apperr.InvalidInput{Description: "book name is required"}
```

---

## Vertical Slice

### Book (`internal/book/`)

```
book/
├── domain/
│   ├── entity.go        # Book + Books
│   ├── repository.go    # BookRepository interface
│   └── errors.go
├── application/
│   ├── create.go
│   ├── update.go
│   ├── get.go
│   └── list.go
├── infra/
│   ├── row.go           # type BookRow = sqlgen.Book + ToDomain() + ToRow()
│   └── repo.go          # Embeds MappedRepo[Book, BookRow, uuid.UUID]
└── handler.go
```

### Repo

```go
// internal/book/infra/repo.go
type BookRepo struct {
    *repo.MappedRepo[domain.Book, BookRow, uuid.UUID]
}

func NewBookRepo(pool *pgxpool.Pool) *BookRepo {
    base := repo.NewBaseRepo[BookRow, uuid.UUID](pool, "books", "book_id")
    return &BookRepo{
        MappedRepo: repo.NewMappedRepo(base, ToDomain, ToRow),
    }
}
```

### Proto

```
app/eureka/proto/eureka/v1/
├── health.proto
└── book.proto           # CreateBook, UpdateBook, GetBook, ListBooks
```

---

## Tasks

### Phase 7: Bootstrap + Book CRUD

- [ ] **7.1** `cmd/server/main.go` — Fx app bootstrap, health check
- [ ] **7.2** Alembic migration: `CREATE SCHEMA IF NOT EXISTS eureka`
- [ ] **7.3** Alembic migration: `eureka.books`
- [ ] **7.4** Alembic migration: `eureka.users` (CDC mirror)
- [ ] **7.5** Alembic migration: `eureka.courses` (CDC mirror)
- [ ] **7.6** Alembic migration: `eureka.outbox`
- [ ] **7.7** Alembic migration: enable RLS, create policies
- [ ] **7.8** Configure `sqlc.yaml`, run `sqlc generate` → `internal/sqlgen/`
- [ ] **7.9** Wire pgxpool + health check via Fx
- [ ] **7.10** `internal/book/` vertical slice (domain, infra, application, handler)
- [ ] **7.11** Proto: `book.proto` → `buf generate`
- [ ] **7.12** Wire interceptor chain
- [ ] **7.13** Unit tests
