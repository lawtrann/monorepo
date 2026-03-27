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

Based on pattern from `lawtrann/backend-py`. All files embedded below — agent does NOT need external access.

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
    ├── versions/             # .gitkeep
    └── sql/                  # .gitkeep
```

Key differences from backend-py:
- DB URL built from env vars directly (no Python app config module)
- Each app targets its own schema (mastermgmt or eureka)
- `search_path` set in env.py connection via `SET search_path TO <schema>`

#### alembic.ini (adapt `script_location` and schema per app)
```ini
[alembic]
script_location = %(here)s/migrations
sqlalchemy.url = postgresql://postgres:postgres@localhost:5432/platform

[post_write_hooks]
hooks = create_sql_files
create_sql_files.type = exec
create_sql_files.executable = python3
create_sql_files.options = %(here)s/scripts/create_sql_files.py REVISION_SCRIPT_FILENAME

[loggers]
keys = root,sqlalchemy,alembic

[handlers]
keys = console

[formatters]
keys = generic

[logger_root]
level = WARN
handlers = console

[logger_sqlalchemy]
level = WARN
handlers =
qualname = sqlalchemy.engine

[logger_alembic]
level = INFO
handlers =
qualname = alembic

[handler_console]
class = StreamHandler
args = (sys.stderr,)
level = NOTSET
formatter = generic

[formatter_generic]
format = %(levelname)-5.5s [%(name)s] %(message)s
datefmt = %H:%M:%S
```

#### migrations/env.py (adapt schema name per app)
```python
import os
from logging.config import fileConfig

from alembic import context
from sqlalchemy import engine_from_config, event, pool

config = context.config

if config.config_file_name is not None:
    fileConfig(config.config_file_name)

# Build DB URL from env vars (override alembic.ini default)
db_user = os.getenv("POSTGRES_USER", "postgres")
db_pass = os.getenv("POSTGRES_PASSWORD", "postgres")
db_host = os.getenv("DB_HOST", "localhost")
db_port = os.getenv("DB_PORT", "5432")
db_name = os.getenv("POSTGRES_DB", "platform")
config.set_main_option(
    "sqlalchemy.url",
    f"postgresql://{db_user}:{db_pass}@{db_host}:{db_port}/{db_name}",
)

# CHANGE THIS per app: "mastermgmt" or "eureka"
SCHEMA = os.getenv("ALEMBIC_SCHEMA", "mastermgmt")

# SQL-first workflow: no model metadata needed (autogenerate disabled)
target_metadata = None


def run_migrations_offline() -> None:
    url = config.get_main_option("sqlalchemy.url")
    context.configure(
        url=url,
        target_metadata=target_metadata,
        literal_binds=True,
        dialect_opts={"paramstyle": "named"},
    )
    with context.begin_transaction():
        context.run_migrations()


def run_migrations_online() -> None:
    connectable = engine_from_config(
        config.get_section(config.config_ini_section, {}),
        prefix="sqlalchemy.",
        poolclass=pool.NullPool,
    )

    with connectable.connect() as connection:
        # Set search_path to the service schema
        connection.execute(
            connection.connection.cursor().execute(
                f"SET search_path TO {SCHEMA}"
            ) if False else connection.execution_options()
        )

        @event.listens_for(connection, "connect")
        def set_search_path(dbapi_conn, connection_record):
            cursor = dbapi_conn.cursor()
            cursor.execute(f"SET search_path TO {SCHEMA}")
            cursor.close()

        context.configure(
            connection=connection,
            target_metadata=target_metadata,
            version_table_schema=SCHEMA,
        )
        with context.begin_transaction():
            context.run_migrations()


if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()
```

#### migrations/script.py.mako
```mako
<%
import re
slug = re.sub(r'\W+', '_', message.lower()).strip('_')
%>\
"""${message}

Revision ID: ${up_revision}
Revises: ${down_revision | comma,n}
Create Date: ${create_date}

"""
from pathlib import Path
from typing import Sequence, Union

from alembic import op

revision: str = ${repr(up_revision)}
down_revision: Union[str, Sequence[str], None] = ${repr(down_revision)}
branch_labels: Union[str, Sequence[str], None] = ${repr(branch_labels)}
depends_on: Union[str, Sequence[str], None] = ${repr(depends_on)}
REVISION_MESSAGE: str = ${repr(message)}

SQL_DIR = Path(__file__).resolve().parent.parent / "sql"


def upgrade() -> None:
    sql = (SQL_DIR / "${up_revision}_${slug}_upgrade.sql").read_text()
    op.execute(sql)


def downgrade() -> None:
    sql = (SQL_DIR / "${up_revision}_${slug}_downgrade.sql").read_text()
    op.execute(sql)
```

#### scripts/create_sql_files.py
```python
#!/usr/bin/env python3
"""Post-write hook to create SQL files for Alembic migrations."""
import re
import sys
from pathlib import Path


def create_sql_files(revision_script: str) -> None:
    migration_file = Path(revision_script)
    content = migration_file.read_text()

    match = re.search(r"revision: str = ['\"]([^'\"]+)['\"]", content)
    if not match:
        print(f"Warning: Could not find revision ID in {revision_script}", file=sys.stderr)
        return

    revision_id = match.group(1)

    filename_parts = migration_file.stem.split('_')
    try:
        rev_index = next(i for i, part in enumerate(filename_parts) if revision_id.startswith(part))
        slug = '_'.join(filename_parts[rev_index + 1:])
    except (StopIteration, IndexError):
        slug = "migration"

    message_match = re.search(r'^"""(.+?)\n', content, re.MULTILINE)
    revision_message = message_match.group(1) if message_match else slug

    sql_dir = migration_file.parent.parent / "sql"
    sql_dir.mkdir(exist_ok=True)

    upgrade_file = sql_dir / f"{revision_id}_{slug}_upgrade.sql"
    upgrade_file.write_text(f"""-- {revision_message}
-- Revision: {revision_id}
-- Upgrade SQL

-- Add your upgrade SQL here

""")

    downgrade_file = sql_dir / f"{revision_id}_{slug}_downgrade.sql"
    downgrade_file.write_text(f"""-- {revision_message}
-- Revision: {revision_id}
-- Downgrade SQL

-- Add your downgrade SQL here

""")

    print(f"Created SQL files:\n  - {upgrade_file}\n  - {downgrade_file}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: create_sql_files.py <revision_script_path>", file=sys.stderr)
        sys.exit(1)
    create_sql_files(sys.argv[1])
```

#### pyproject.toml (per app)
```toml
[project]
name = "<service>-migration"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = [
    "alembic>=1.18.4",
    "psycopg2-binary>=2.9.10",
]
```

### Casdoor API authentication

Casdoor ships with a default admin account:
- **Username**: `admin`
- **Password**: `123`
- **Organization**: `built-in`
- **Application**: `app-built-in` (built-in app for admin login)

To authenticate with the Casdoor REST API:
1. POST `http://localhost:8000/api/login` with `{"type": "login/account", "username": "admin", "password": "123", "organization": "built-in", "application": "app-built-in"}`
2. Extract the `accessToken` from the response
3. Use `Authorization: Bearer <token>` for subsequent API calls

Key API endpoints for seeding:
- `POST /api/add-organization` — create org
- `POST /api/add-application` — create app
- `POST /api/add-role` — create role
- `POST /api/add-user` — create user
- `PUT /api/update-model` — set Casbin model

### Casdoor seed script

Uses Casdoor REST API (authenticated as admin) to create:
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
