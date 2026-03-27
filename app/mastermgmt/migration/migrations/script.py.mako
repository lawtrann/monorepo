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
