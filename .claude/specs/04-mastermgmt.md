# 04 — Mastermgmt Service

Tenant, Location & Course Management. Implemented incrementally: skeleton → tenant → auth → location → course.

---

## Database Schema

### Tenant Isolation (RLS)

```sql
-- Session variable (set by PrepareConn from JWT org claim)
SET app.current_tenant = 'lms';

-- Every table (except tenants) has:
tenant TEXT NOT NULL DEFAULT current_setting('app.current_tenant')

-- RLS policy:
ALTER TABLE mastermgmt.locations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mastermgmt.locations
    USING (tenant = current_setting('app.current_tenant'));
```

### mastermgmt.tenants

No RLS (admin-managed). No DEFAULT on tenant.

| Column | Type | Notes |
|---|---|---|
| `tenant_id` | `UUID v7` | PK |
| `name` | `TEXT NOT NULL` | e.g. "LMS School" |
| `slug` | `TEXT NOT NULL UNIQUE` | e.g. "lms" — used for RLS across all tables |
| `settings` | `JSONB` | DEFAULT '{}' |
| `is_active` | `BOOLEAN` | DEFAULT true |
| `created_at` | `TIMESTAMPTZ` | DEFAULT now() |
| `updated_at` | `TIMESTAMPTZ` | DEFAULT now() |
| `deleted_at` | `TIMESTAMPTZ` | NULL = active |

### mastermgmt.locations

| Column | Type | Notes |
|---|---|---|
| `location_id` | `UUID v7` | PK |
| `name` | `TEXT NOT NULL` | |
| `address` | `TEXT` | |
| `phone` | `TEXT` | |
| `is_active` | `BOOLEAN` | DEFAULT true |
| `tenant` | `TEXT NOT NULL` | `DEFAULT current_setting('app.current_tenant')` |
| `created_at` | `TIMESTAMPTZ` | DEFAULT now() |
| `updated_at` | `TIMESTAMPTZ` | DEFAULT now() |
| `deleted_at` | `TIMESTAMPTZ` | NULL = active |

### mastermgmt.courses

| Column | Type | Notes |
|---|---|---|
| `course_id` | `UUID v7` | PK |
| `name` | `TEXT NOT NULL` | |
| `tenant` | `TEXT NOT NULL` | `DEFAULT current_setting('app.current_tenant')` |
| `created_at` | `TIMESTAMPTZ` | DEFAULT now() |
| `updated_at` | `TIMESTAMPTZ` | DEFAULT now() |
| `deleted_at` | `TIMESTAMPTZ` | NULL = active |

### mastermgmt.course_locations (junction)

| Column | Type | Notes |
|---|---|---|
| `course_location_id` | `UUID v7` | PK |
| `course_id` | `UUID v7 NOT NULL` | FK → courses.course_id |
| `location_id` | `UUID v7 NOT NULL` | FK → locations.location_id |
| `tenant` | `TEXT NOT NULL` | `DEFAULT current_setting('app.current_tenant')` |
| `created_at` | `TIMESTAMPTZ` | DEFAULT now() |

**Constraints**: `UNIQUE(course_id, location_id)`

### mastermgmt.users (CDC mirror)

| Column | Type | Notes |
|---|---|---|
| `user_id` | `UUID` | PK (from Casdoor) |
| `username` | `TEXT` | |
| `email` | `TEXT` | |
| `org` | `TEXT` | Casdoor org = tenant slug |
| `is_active` | `BOOLEAN` | |
| `synced_at` | `TIMESTAMPTZ` | |

### mastermgmt.outbox

| Column | Type | Notes |
|---|---|---|
| `id` | `UUID v7` | PK |
| `aggregatetype` | `TEXT NOT NULL` | e.g. "tenant", "course" |
| `aggregateid` | `TEXT NOT NULL` | Entity ID |
| `type` | `TEXT NOT NULL` | e.g. "TenantCreated" |
| `payload` | `JSONB NOT NULL` | Event data |
| `created_at` | `TIMESTAMPTZ` | DEFAULT now() |

---

## sqlc + Type Alias Pattern

sqlc reads schema DDL and generates Row structs in `internal/sqlgen/`:

```yaml
# app/mastermgmt/sqlc.yaml
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

Each vertical slice aliases the sqlc-generated type:

```go
// internal/tenant/infra/row.go
package infra

import "app/mastermgmt/internal/sqlgen"

type TenantRow = sqlgen.Tenant  // type alias — zero cost

func ToDomain(r *TenantRow) *domain.Tenant {
    return &domain.Tenant{
        ID: r.TenantID, Name: r.Name, Slug: r.Slug,
        Settings: r.Settings, IsActive: r.IsActive,
    }
}

func ToRow(e *domain.Tenant) *TenantRow {
    return &TenantRow{
        TenantID: e.ID, Name: e.Name, Slug: e.Slug,
        Settings: e.Settings, IsActive: e.IsActive,
    }
}
```

---

## Domain Entities

```go
// internal/tenant/domain/entity.go
type Tenant struct {
    ID       uuid.UUID
    Name     string
    Slug     string
    Settings map[string]any
    IsActive bool
}
type Tenants []*Tenant

// internal/location/domain/entity.go
type Location struct {
    ID       uuid.UUID
    Name     string
    Address  string
    Phone    string
    IsActive bool
}
type Locations []*Location

// internal/course/domain/entity.go
type Course struct {
    ID   uuid.UUID
    Name string
}
type Courses []*Course
// NO Locations field. Use GetCourseWithLocations() for explicit loading.
```

---

## Vertical Slices

### Tenant (`internal/tenant/`)

```
tenant/
├── domain/
│   ├── entity.go        # Tenant + Tenants
│   ├── event.go         # TenantCreated, TenantUpdated
│   ├── repository.go    # TenantRepository interface
│   └── errors.go        # ErrTenantNotFound, ErrTenantSlugTaken
├── application/
│   ├── create.go
│   ├── update.go
│   ├── get.go
│   └── list.go
├── infra/
│   ├── row.go           # type TenantRow = sqlgen.Tenant + ToDomain() + ToRow()
│   └── repo.go          # Embeds MappedRepo[Tenant, TenantRow, uuid.UUID]
└── handler.go
```

### Location (`internal/location/`)

```
location/
├── domain/
│   ├── entity.go        # Location + Locations
│   ├── repository.go
│   └── errors.go
├── application/
│   ├── create.go, update.go, get.go, list.go
├── infra/
│   ├── row.go           # type LocationRow = sqlgen.Location + mapping
│   └── repo.go          # Embeds MappedRepo[Location, LocationRow, uuid.UUID]
└── handler.go
```

### Course (`internal/course/`)

```
course/
├── domain/
│   ├── entity.go        # Course + Courses
│   ├── repository.go    # CRUD + GetWithLocations, AssignLocation, UnassignLocation
│   └── errors.go
├── application/
│   ├── create.go, update.go, get.go, list.go
│   ├── get_with_locations.go
│   ├── assign_location.go
│   └── unassign_location.go
├── infra/
│   ├── row.go           # type CourseRow = sqlgen.Course + mapping
│   └── repo.go          # Embeds MappedRepo + raw pgx for joins/assign/unassign
└── handler.go
```

### Proto Definitions

```
app/mastermgmt/proto/mastermgmt/v1/
├── health.proto
├── tenant.proto         # CreateTenant, UpdateTenant, GetTenant, ListTenants
├── location.proto       # CreateLocation, UpdateLocation, GetLocation, ListLocations
└── course.proto         # CRUD + GetCourseWithLocations, AssignLocation, UnassignLocation
```

---

## Tasks (Vertical-First Order)

### Phase 3: Skeleton + DB

- [ ] **3.1** `cmd/server/main.go` — Fx app bootstrap, health check endpoint
- [ ] **3.2** Alembic migration: `CREATE SCHEMA IF NOT EXISTS mastermgmt`
- [ ] **3.3** Alembic migration: `mastermgmt.tenants`
- [ ] **3.4** Alembic migration: `mastermgmt.locations`
- [ ] **3.5** Alembic migration: `mastermgmt.courses`
- [ ] **3.6** Alembic migration: `mastermgmt.course_locations` (UNIQUE constraint)
- [ ] **3.7** Alembic migration: `mastermgmt.users` (CDC mirror)
- [ ] **3.8** Alembic migration: `mastermgmt.outbox`
- [ ] **3.9** Alembic migration: enable RLS on all tables (except tenants)
- [ ] **3.10** Configure `sqlc.yaml`, run `sqlc generate` → `internal/sqlgen/`
- [ ] **3.11** Wire pgxpool + health check via Fx
- [ ] **3.12** Verify: service starts, health check responds

### Phase 4: Tenant CRUD (First Working Feature)

- [ ] **4.1** `internal/tenant/domain/` — entity, repository interface, errors
- [ ] **4.2** `internal/tenant/infra/row.go` — TenantRow alias + ToDomain + ToRow
- [ ] **4.3** `internal/tenant/infra/repo.go` — embeds MappedRepo[Tenant, TenantRow, uuid.UUID]
- [ ] **4.4** `internal/tenant/application/` — create, update, get, list use cases
- [ ] **4.5** `internal/tenant/handler.go` — ConnectRPC handler
- [ ] **4.6** Proto: `tenant.proto` with protovalidate → `buf generate`
- [ ] **4.7** Verify: Tenant CRUD works end-to-end (no auth yet — add in Phase 5)

### Phase 5: Auth Interceptors (Now We Have Endpoints)

- [ ] **5.1** Wire interceptor chain: requestID → logging → auth → tenant → error → validate → handler
- [ ] **5.2** Verify: JWT required, tenant isolation active

### Phase 6: Location + Course

- [ ] **6.1** `internal/location/` vertical slice (domain, infra, application, handler)
- [ ] **6.2** `internal/course/` vertical slice (including GetWithLocations, Assign/Unassign)
- [ ] **6.3** Protos: `location.proto`, `course.proto` → `buf generate`
- [ ] **6.4** Unit tests per module
