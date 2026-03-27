# Session 26

Date: 2026-03-27
Task: 2.8 — Unit tests for apperr, BaseRepo, MappedRepo, reflect utils using mock Pool
Phase: 2
Status: COMPLETED

## Summary

Created unit tests for the entire pkg/goshared repo layer:

- `pkg/goshared/apperr/apperr_test.go` — tests all 5 error types (NotFound, AlreadyExists, InvalidInput, Forbidden, Conflict): Error(), Unwrap(), errors.As, errors.Is wrapping
- `pkg/goshared/repo/reflect_test.go` — tests structToColumnsAndValues (skips tenant/timestamps/no-tag/-tag), fieldToColumn (found/not-found/dash-tag), fieldsToColumns (mapping + filtering)
- `pkg/goshared/repo/base_test.go` — local mock Pool (struct with fn fields), mock Rows (implements pgx.Rows, intercepts rowScanner for namedStructRowScanner support, then positional assignment), mock Row; tests: GetByID found/not-found/db-error, List basic/next-cursor/offset-pagination, Create, Update found/not-found, UpdateFields partial/fallback, SoftDelete found/not-found/db-error; plus MappedRepo tests: GetByID/List/Create/Update/SoftDelete with toDomain/toRow translation verified

Key design decision: mock Rows.Scan checks if len(dest)==1 and dest[0] is a rowScanner (matches pgx internal namedStructRowScanner). If so, delegates to ScanRow which then calls Scan with field pointers for positional assignment. This correctly drives pgx.RowToStructByName without a real DB.

## Commits

- c0cd4e1: feat(test): unit tests for apperr, BaseRepo, MappedRepo, reflect utils [task 2.8]

## Infra state

No infrastructure needed. Docker services not running.

## Next

Task 2.9 (if defined) or next phase tasks.
