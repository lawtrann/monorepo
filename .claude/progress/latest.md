# Session 18

Date: 2026-03-27
Task: 1.7 — Verify end-to-end: seed Casdoor, login as test user, decode JWT, confirm claims contain org, roles, permissions
Phase: 1
Status: COMPLETED

## Summary

- Wrote `scripts/verify-jwt.sh` — logs in as test-teacher via OAuth2 password grant, decodes JWT, asserts: `owner==lms`, `name` present, `roles` contains `teacher`
- Fixed `scripts/seed-casdoor.sh` — changed `tokenFormat` from `JWT-Standard` to `JWT` (Casdoor's own format); `JWT-Standard` omits roles/permissions claims. Added update-always logic so existing environments are patched on re-run
- Key pitfall avoided: `POST /api/login` is broken in Casdoor v1.777.0 — used OAuth2 password grant (`POST /api/login/oauth/access_token`) instead
- Casdoor JWT format: roles are objects `{"owner":"lms","name":"teacher",...}` not strings; user `test-teacher` has roles `[staff, teacher]` (staff via role hierarchy)

## Commits

- 19e95a1: fix(casdoor): use tokenFormat=JWT for platform app to include roles in JWT [task 1.7]
- 524a033: feat(casdoor): write verify-jwt.sh for end-to-end JWT verification [task 1.7]

## Infra state

Docker Compose brought down after verify. Services: db (postgres:18) on port 5432, casdoor on port 8000 — ready to start.

## Next

Task 2.1 (pkg/goshared/apperr/apperr.go — error types) is next in dependency order.
