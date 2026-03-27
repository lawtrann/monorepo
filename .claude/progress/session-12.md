# Session 12
Date: 2026-03-27
Task: 1.1 — Create scripts/init-db.sql — schemas (casdoor, mastermgmt, eureka) + casdoor_user role with search_path
Phase: 1
Status: COMPLETED

## Summary
- Created `scripts/init-db.sql` following the phase-01 pattern exactly
- Creates 3 schemas: casdoor, mastermgmt, eureka
- Creates casdoor_user role idempotently with login/password
- Sets search_path and grants schema access for casdoor_user

## Commits
- 3e658a4: feat(db-init): create scripts/init-db.sql with schemas and casdoor_user role [task 1.1]

## Infra state
No infrastructure yet. This SQL script will be mounted into the postgres container in task 1.3.

## PR
- https://github.com/lawtrann/monorepo/pull/15

## Next
Task 1.2 (Casdoor app.conf) is next in dependency order.
