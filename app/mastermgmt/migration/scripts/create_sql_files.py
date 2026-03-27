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
