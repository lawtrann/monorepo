# Session 24

Date: 2026-03-27
Task: 2.6 — Create pkg/goshared/repo/base.go — BaseRepo[R, ID] using Pool interface
Phase: 2
Status: COMPLETED

## Summary

Created `pkg/goshared/repo/base.go` with:
- `NewBaseRepo[R, ID](pool, table, pk)` constructor
- `GetByID` — SELECT * WHERE pk=$1 AND deleted_at IS NULL → apperr.NotFound on miss
- `List` — cursor pagination (pk > $cursor, limit+1 probe) or offset pagination (with COUNT total), plain limit otherwise
- `Create` — INSERT INTO ... RETURNING *, columns/values from structToColumnsAndValues
- `Update` — full replace UPDATE ... RETURNING *, apperr.NotFound on miss
- `UpdateFields` — partial UPDATE via fieldsToColumns mask + structToColumnsAndValues filter
- `SoftDelete` — UPDATE SET deleted_at=now(), apperr.NotFound if 0 rows affected
- `extractUUIDByCol` private helper for cursor next-page detection

All methods use pool.Query/QueryRow/Exec (interface, not concrete). pgx.CollectOneRow / pgx.CollectRows with pgx.RowToStructByName for struct scanning.

## Commits

- 33d44da: feat(repo): add BaseRepo[R, ID] generic CRUD using Pool interface [task 2.6]

## Infra state

No infrastructure needed. Docker services not running.

## Next

Task 2.7 is next — MappedRepo[E, R, ID] wrapping BaseRepo with toDomain/toRow translation.
