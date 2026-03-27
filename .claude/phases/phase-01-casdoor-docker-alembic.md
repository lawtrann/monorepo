# Phase 1: Casdoor + Docker Compose + Alembic

## Prerequisites
> Phase 0 completed: monorepo scaffold, Go modules, deployment dirs, tooling
> Key files from prior phase: `Makefile`, `deployment/casdoor/local/`, `app/mastermgmt/migration/`, `app/eureka/migration/`

## Decisions
- **Docker Compose**: Minimal — only db (postgres:18) + casdoor. Expanded in Phase 10
- **DB Init**: Single `scripts/init-db.sql` creates all 3 schemas + casdoor_user role
- **Casdoor Config**: Automated via REST API seed script, not manual UI
- **Migration**: SQL-first Alembic pattern (from backend-py reference): target_metadata=None, post-write hook creates SQL stubs, custom script.py.mako
- **Alembic**: Separate setup per app, each targeting its own schema
> See .claude/specs/03-casdoor-setup.md for Casdoor config details and auth flow

## Tasks in this phase
| ID  | Description | Size | File |
|-----|-------------|------|------|
| 1.1 | DB init script: 3 schemas + casdoor_user role | small | scripts/init-db.sql |
| 1.2 | Casdoor app.conf for local dev | small | deployment/casdoor/local/app.conf |
| 1.3 | docker-compose.yml: db + casdoor | medium | docker-compose.yml |
| 1.4 | Alembic SQL-first setup for mastermgmt | medium | app/mastermgmt/migration/alembic.ini |
| 1.5 | Alembic SQL-first setup for eureka | medium | app/eureka/migration/alembic.ini |
| 1.6 | Casdoor REST API seed script | medium | scripts/seed-casdoor.sh |
| 1.7 | Verify JWT end-to-end | small | scripts/verify-jwt.sh |

## Key patterns

### DB init script
```sql
-- scripts/init-db.sql
CREATE SCHEMA IF NOT EXISTS casdoor;
CREATE SCHEMA IF NOT EXISTS mastermgmt;
CREATE SCHEMA IF NOT EXISTS eureka;

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

### Casdoor config
```ini
# deployment/casdoor/local/app.conf
appname = casdoor
httpport = 8000
driverName = postgres
dataSourceName = "user=casdoor_user password=casdoor_pass host=db port=5432 sslmode=disable dbname=platform"
dbName = platform
```

### docker-compose.yml
```yaml
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

### Alembic SQL-first setup (per app)

Reference: https://github.com/lawtrann/backend-py

Directory structure per app:
```
app/<service>/migration/
├── alembic.ini              # post-write hook: create_sql_files
├── pyproject.toml            # alembic dependency
├── scripts/
│   └── create_sql_files.py   # auto-creates upgrade/downgrade SQL stubs
└── migrations/
    ├── env.py                # target_metadata=None, DB URL from env vars
    ├── script.py.mako        # custom template reading sql files
    ├── versions/
    └── sql/
```

Key differences from backend-py:
- DB URL built from env vars directly (no Python app config module)
- Each app targets its own schema (mastermgmt or eureka)
- `search_path` set in env.py connection

alembic.ini post-write hook:
```ini
[post_write_hooks]
hooks = create_sql_files
create_sql_files.type = exec
create_sql_files.executable = python3
create_sql_files.options = %(here)s/scripts/create_sql_files.py REVISION_SCRIPT_FILENAME
```

env.py key pattern:
```python
# target_metadata = None (SQL-first, no autogenerate)
# DB URL from env vars: POSTGRES_USER, POSTGRES_PASSWORD, DB_HOST, DB_PORT, POSTGRES_DB
# Set search_path to service schema on connect
```

### Casdoor seed script

Uses Casdoor REST API to create:
1. Organization `lms` (tenant slug = org name)
2. Platform application (OAuth2/OIDC)
3. Casbin model (RBAC with domains)
4. Roles: admin, school_admin, teacher, student, parent
5. Role hierarchy: staff = teacher + school_admin
6. Test user with assigned roles

### JWT claims (expected after login)
```json
{
  "sub": "user-uuid",
  "name": "Nguyen Van A",
  "org": "lms",
  "roles": ["teacher"],
  "permissions": ["book:read", "book:create"],
  "groups": ["staff"],
  "iss": "casdoor",
  "aud": "platform-app"
}
```

## Verification
- `docker compose up -d && curl -sf http://localhost:8000` — Casdoor UI reachable
- `cd app/mastermgmt/migration && uv run alembic check` — Alembic connects to DB
- `cd app/eureka/migration && uv run alembic check` — Alembic connects to DB
- `bash scripts/seed-casdoor.sh` — Casdoor seeded with org, app, roles
- `bash scripts/verify-jwt.sh` — JWT contains org, roles, permissions claims

## Dependencies
- Requires: Phase 0 (scaffold, deployment dirs, Go modules)
- Blocks: Phase 2 (goshared needs DB running for integration tests later), Phase 4 (tenant CRUD needs migrations)
