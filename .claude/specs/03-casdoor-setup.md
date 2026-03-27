# 03 — Casdoor Setup

This phase includes docker-compose for `db` + `casdoor` so you can test auth flow immediately.

---

## Docker Compose (Minimal — db + casdoor)

```yaml
# docker-compose.yml (initial — expand in phase 07)
services:
  db:
    image: postgres:18
    environment:
      POSTGRES_DB: platform
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - ./scripts/init-db.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s

  casdoor:
    image: casbin/casdoor
    depends_on:
      db: { condition: service_healthy }
    ports:
      - "8000:8000"
    volumes:
      - ./deployment/casdoor/local/app.conf:/conf/app.conf
```

### DB Init Script (`scripts/init-db.sql`)

```sql
-- Schemas
CREATE SCHEMA IF NOT EXISTS casdoor;
CREATE SCHEMA IF NOT EXISTS mastermgmt;
CREATE SCHEMA IF NOT EXISTS eureka;

-- Casdoor role — search_path scoped to casdoor schema
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'casdoor_user') THEN
    CREATE ROLE casdoor_user LOGIN PASSWORD 'casdoor_pass';
  END IF;
END
$$;
ALTER ROLE casdoor_user SET search_path TO casdoor;
GRANT ALL ON SCHEMA casdoor TO casdoor_user;
```

### Casdoor Config (`deployment/casdoor/local/app.conf`)

```ini
appname = casdoor
httpport = 8000
driverName = postgres
dataSourceName = "user=casdoor_user password=casdoor_pass host=db port=5432 sslmode=disable dbname=platform"
dbName = platform
```

Casdoor connects as `casdoor_user` → XORM creates all its tables in the `casdoor` schema automatically via `ALTER ROLE SET search_path`.

---

## Casdoor Configuration

### Organizations

1:1 mapping: one Casdoor Organization = one Tenant.

| Casdoor Org | Tenant Slug | Tenant Name |
|---|---|---|
| `lms` | `lms` | LMS School |
| `sunrise-academy` | `sunrise-academy` | Sunrise Academy |

The org name in Casdoor IS the tenant slug used for RLS.

### Application

One Casdoor Application for the platform. All orgs share it. OAuth2/OIDC endpoints:
- Authorization: `{casdoor}/login/oauth/authorize`
- Token: `{casdoor}/api/login/oauth/access_token`
- JWKS: `{casdoor}/.well-known/jwks`

### JWT Custom Claims

```json
{
  "sub": "user-uuid",
  "name": "Nguyen Van A",
  "org": "lms",
  "roles": ["teacher"],
  "permissions": ["book:read", "book:create"],
  "groups": ["staff"],
  "iss": "casdoor",
  "aud": "platform-app",
  "exp": 1234567890
}
```

- `org` — tenant slug → `SET app.current_tenant = 'lms'`
- `roles` — Casbin roles for authorization
- `permissions` — fine-grained permissions
- `groups` — role hierarchies

### Casbin Model (RBAC with Domains)

```ini
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
```

### Example Policies

```
p, teacher, lms, book, read
p, teacher, lms, book, create
p, teacher, lms, course, read
p, teacher, lms, course, create
p, student, lms, book, read
p, student, lms, course, read
p, school_admin, lms, tenant, read
p, school_admin, lms, tenant, update
g, teacher, staff, lms
g, school_admin, staff, lms
```

---

## Auth Flow

```
Client → Casdoor (login) → JWT with org="lms"
Client → Service RPC + JWT
  → interceptor/auth: validate JWT (jwx v3, JWKS)
  → interceptor/tenant: extract org "lms" → context
  → PrepareConn: SET app.current_tenant = 'lms'
  → handler → use case → repo → DB (RLS filtered)
  → AfterRelease: RESET app.current_tenant
```

---

## Tasks

- [ ] **1.1** Create `scripts/init-db.sql` (schemas + casdoor_user role)
- [ ] **1.2** Create `deployment/casdoor/local/app.conf`
- [ ] **1.3** Add `db` + `casdoor` to `docker-compose.yml`
- [ ] **1.4** Verify: `docker-compose up` → Casdoor UI at localhost:8000
- [ ] **1.5** Create organization `lms` in Casdoor
- [ ] **1.6** Create application for the platform
- [ ] **1.7** Configure Casbin model (RBAC with domains)
- [ ] **1.8** Seed roles: admin (global), school_admin, teacher, student, parent
- [ ] **1.9** Configure role hierarchy: staff = teacher + school_admin
- [ ] **1.10** Configure JWT custom claims: roles, permissions, org slug
- [ ] **1.11** Test: login → JWT contains correct claims
