# Session 17

Date: 2026-03-27
Task: 1.6 — Write Casdoor REST API seed script (scripts/seed-casdoor.sh)
Phase: 1
Status: COMPLETED

## Summary

- Wrote `scripts/seed-casdoor.sh` — idempotent Casdoor seed using REST API with Basic auth (clientId:clientSecret from DB)
- Creates: org `lms`, application `platform` (reuses cert-built-in), Casbin RBAC model `rbac-with-domains`, roles (admin, school_admin, teacher, student, parent, staff hierarchy), test user `test-teacher` assigned to teacher role
- Auth pattern: fetch credentials from `public.application` WHERE name='app-built-in' each run (never hardcoded)
- Idempotency: GET-check before each create; role assignment is always applied (safe to re-run)

## Commits

- 2f60b79: feat(casdoor): write REST API seed script [task 1.6]

## Infra state

Docker Compose brought down after verify. Services: db (postgres:18) on port 5433, casdoor on port 8000 — ready to start.

## Next

Task 1.7 (verify-jwt.sh: seed → login → decode JWT → confirm claims) is next in dependency order.
