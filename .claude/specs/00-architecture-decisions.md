# 00 — Architecture Decisions

**Go (ConnectRPC) — Education Platform**

---

## Architecture Overview

```
┌─────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Web    │────▶│   Casdoor    │     │  Mastermgmt  │     │   Eureka     │
│(Next.js)│     │  (external)  │     │   (Go)       │     │   (Go)       │
│         │     │              │     │              │     │              │
│         │     │ • Auth (IdP) │     │ • Tenants    │     │ • Books      │
│         │     │ • Users      │     │ • Locations  │     │              │
│         │     │ • Roles/RBAC │     │ • Courses    │     │              │
│         │     │   (Casbin)   │     │ • Course ↔   │     │              │
│         │     │ • Orgs       │     │   Locations  │     │              │
│         │     │ • JWT+claims │     │              │     │              │
│         │     │              │     │              │     │              │
│         │     │ casdoor.*    │     │ mastermgmt.* │     │ eureka.*     │
└─────────┘     └──────┬───────┘     └──────┬───────┘     └──────┬───────┘
                       │                    │                    │
                       ▼                    ▼                    ▼
              ┌─────────────────────────────────────────────────────────┐
              │              PostgreSQL (single database: platform)     │
              │   casdoor.*         │  mastermgmt.*  │  eureka.*        │
              └─────────────────────────────────────────────────────────┘
                       │                    │                    │
                       ▼                    ▼                    ▼
              ┌─────────────────────────────────────────────────────────┐
              │           Debezium CDC + Outbox Event Router            │
              │  • casdoor.user → Kafka → mastermgmt-worker, eureka-worker│
              │  • mastermgmt.outbox → Kafka (domain events)           │
              │  • mastermgmt.courses → Kafka → eureka-worker (CDC)    │
              │  • eureka.outbox → Kafka (domain events)               │
              └─────────────────────────────────────────────────────────┘
```

All services share one PostgreSQL database (`platform`) with isolated schemas. This simplifies Debezium CDC (one connector, one WAL stream), operations (one backup, one connection endpoint), and local development.

---

## Decisions

| Concern | Decision |
|---|---|
| **Services** | Casdoor (external) + Mastermgmt (Go) + Eureka (Go) + Web (Next.js) |
| **API** | ConnectRPC + Buf. Each service validates JWT locally, no gateway |
| **Validation** | protovalidate (constraints in .proto) |
| **Database** | Single DB (`platform`) with 3 schemas: `casdoor`, `mastermgmt`, `eureka` |
| **DB Driver** | pgx v5 + pgxpool |
| **DB Models** | sqlc generates Row structs from schema DDL. `sqlgen/` in each service's `internal/` |
| **DB Access** | MappedRepo[E, R, ID] wraps BaseRepo for CRUD. Raw pgx for complex queries |
| **PK Convention** | `<table_name>_id` UUID v7 for all entities |
| **Tenant Identity** | `tenant_id UUID v7` (PK on tenants) + `slug TEXT UNIQUE` (used for RLS) |
| **RLS Column** | `tenant TEXT NOT NULL DEFAULT current_setting('app.current_tenant')` on all non-tenant tables |
| **Multi-tenancy** | `PrepareConn` sets `app.current_tenant` → DB DEFAULT auto-fills on INSERT → RLS policy filters all ops |
| **Migrations** | Alembic (Python) per app |
| **DI** | Uber Fx |
| **Auth (AuthN)** | Casdoor (IdP, OAuth2/OIDC) |
| **Auth (AuthZ)** | Casdoor's built-in Casbin (RBAC) |
| **JWT** | Casdoor issues JWT with custom claims (roles, permissions, org slug). Validated locally via lestrrat-go/jwx v3 + JWKS |
| **Interceptors** | requestID → logging → auth → tenant → error → validate → handler |
| **Domain Layer** | Pure business — no tenant, no timestamps, no DB concerns |
| **Row ↔ Domain** | MappedRepo with embedding. sqlc-generated Row types aliased in infra. `ToDomain()` on Row, `ToRow()` standalone. Use case sees domain types only |
| **Update Strategy** | `Update` (full replace) + `UpdateFields` (field mask with Go struct field names) |
| **Error Strategy** | Go-idiomatic error types in `apperr/` (pure Go, with `Unwrap`). `grpc/connect/interceptor/error.go` maps via `errors.As` |
| **gRPC Layer** | `grpc/interfaces/` (port) + `grpc/connect/` (adapter: interceptors, middleware) |
| **Logging** | slog (stdlib) |
| **Config** | Viper (config.yaml) + SOPS+age (secrets.yaml) per env |
| **Object Storage** | MinIO Go SDK (minio-go v7) |
| **Events** | Kafka (franz-go) + Debezium CDC outbox |
| **Pub/Sub** | `pubsub/interfaces/` (port) + `pubsub/franz/` (adapter) |
| **Event Format** | JSON (jsonb outbox column) |
| **Domain Events** | Aggregate root collects → UoW flushes to outbox atomically |
| **User Sync** | CDC via Debezium on casdoor schema (same DB) |
| **Course Sync** | CDC from `mastermgmt.courses` → `eureka.courses` |
| **Testing** | testify/mock + shared `mock/` package in goshared |
| **Module Structure** | Vertical slices with DDD inside each module |
| **Pagination** | Both cursor-based (UUID v7) and offset/limit |
| **Deployment** | Root-level `deployment/` (Terraform ready) |

---

## Go Dependencies (Pinned)

| Dependency | Version |
|---|---|
| `connectrpc.com/connect` | v1.19.1 |
| `connectrpc.com/validate` | v0.3.0 |
| `github.com/jackc/pgx/v5` | v5.8.0 |
| `go.uber.org/fx` | v1.23.0 |
| `github.com/spf13/viper` | v1.21.0 |
| `github.com/lestrrat-go/jwx/v3` | v3 (latest) |
| `github.com/minio/minio-go/v7` | v7.0.83 |
| `github.com/twmb/franz-go` | v1.18.0 |
| `github.com/stretchr/testify` | v1.10.0 |
| `google.golang.org/protobuf` | v1.36.5 |
| `github.com/sqlc-dev/sqlc` | CLI tool (latest) |

---

## Roles (Casdoor Casbin)

| Role | Scope | Example |
|---|---|---|
| `admin` | Global (all orgs) | Platform admin |
| `school_admin` | Per-org (per tenant) | School manager |
| `teacher` | Per-org | Creates/manages content |
| `student` | Per-org | Consumes content |
| `parent` | Per-org | Views student progress |

**Group hierarchy**: `staff` = `teacher` + `school_admin`

---

## Docker Compose Services (11 total)

| Service | Image | Purpose |
|---|---|---|
| `db` | `postgres:18` | Single `platform` DB (3 schemas + RLS) |
| `minio` | `minio/minio` | Object storage |
| `casdoor` | `casbin/casdoor` | IdP + built-in Casbin RBAC |
| `zookeeper` | `confluentinc/cp-zookeeper` | Kafka coordination |
| `kafka` | `confluentinc/cp-kafka` | Event streaming |
| `debezium` | `debezium/connect` | CDC + Outbox relay |
| `mastermgmt` | Dockerfile | Tenants, locations, courses |
| `mastermgmt-worker` | Dockerfile.worker | CDC + event consumer |
| `eureka` | Dockerfile | Books |
| `eureka-worker` | Dockerfile.worker | CDC + event consumer |
| `web` | Next.js | Frontend |

---

## Implementation Order (Vertical-First)

Build one feature end-to-end first, then expand. Each step produces something testable.

| Phase | Document | What's Runnable After |
|---|---|---|
| 0 | `01-monorepo-foundation.md` | Monorepo skeleton, tooling installed |
| 1 | `03-casdoor-setup.md` | docker-compose (db + casdoor), login → JWT works |
| 2 | `02-goshared.md` (minimal) | db pool + BaseRepo + MappedRepo + apperr + health check |
| 3 | `04-mastermgmt.md` (skeleton) | Fx bootstrap, health check endpoint runs |
| 4 | `04-mastermgmt.md` (tenant) | Tenant CRUD — first working feature |
| 5 | `02-goshared.md` (auth) | Auth + tenant interceptors on live endpoints |
| 6 | `04-mastermgmt.md` (location, course) | Full mastermgmt feature set |
| 7 | `05-eureka.md` | Book CRUD |
| 8 | `06-cdc-outbox-workers.md` | Cross-service sync working |
| 9 | `02-goshared.md` (remaining) | pubsub, storage, config/sops |
| 10 | `07-docker-devex.md` | Full docker-compose, Makefiles, hot-reload |
| 11 | `08-testing.md` | Integration, RLS, E2E, CI |

---

## Future Scope (Not in Initial Implementation)

- `mastermgmt.enrollments` — student ↔ course assignment
- `eureka.assessments` — educational assessments
- `eureka.course_books` — many-to-many course ↔ book
- API versioning strategy — define when first breaking change arrives
