# Monorepo

Multi-service monorepo for the LMS platform.

## Structure

- `app/mastermgmt/` — Master management service (Go)
- `app/eureka/` — Eureka service (Go)
- `app/web/` — Web frontend (deferred)
- `pkg/goshared/` — Shared Go library
- `deployment/` — Per-service deployment configs

## Setup

```bash
make setup
```
