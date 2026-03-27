import os
from logging.config import fileConfig

from alembic import context
from sqlalchemy import MetaData, engine_from_config, pool, text

config = context.config

if config.config_file_name is not None:
    fileConfig(config.config_file_name)

# Build DB URL from env vars (override alembic.ini default)
db_user = os.getenv("POSTGRES_USER", "postgres")
db_pass = os.getenv("POSTGRES_PASSWORD", "postgres")
db_host = os.getenv("DB_HOST", "localhost")
db_port = os.getenv("DB_PORT", "5433")
db_name = os.getenv("POSTGRES_DB", "platform")
config.set_main_option(
    "sqlalchemy.url",
    f"postgresql://{db_user}:{db_pass}@{db_host}:{db_port}/{db_name}",
)

SCHEMA = os.getenv("ALEMBIC_SCHEMA", "mastermgmt")

# SQL-first workflow: empty metadata (no models to autogenerate from)
# Allows `alembic check` to run without error
target_metadata = MetaData(schema=SCHEMA)


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
        connection.execute(text(f"SET search_path TO {SCHEMA}"))
        connection.commit()

        def include_name(name, type_, parent_names):
            if type_ == "schema":
                return name == SCHEMA
            return True

        context.configure(
            connection=connection,
            target_metadata=target_metadata,
            version_table_schema=SCHEMA,
            include_schemas=True,
            include_name=include_name,
        )
        with context.begin_transaction():
            context.run_migrations()


if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()
