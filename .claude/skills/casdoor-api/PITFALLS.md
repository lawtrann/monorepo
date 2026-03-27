# Casdoor API — Pitfalls

## Prerequisite: copyrequestbody = true

**Without this, ALL POST/PUT requests fail with `"unexpected end of JSON input"`.**

Root cause: beego's `auto_signin_filter` middleware consumes the body stream.
Already applied in `deployment/casdoor/local/app.conf`:

```ini
copyrequestbody = true
```

## Failed auth approaches (do not use)

| Approach | Error |
|---|---|
| `POST /api/login` with JSON/form body | `GetOwnerAndNameFromId() error` — v1.777.0 bug in auto_signin_filter.go |
| `-u 'admin:123'` (username:password) | `Application not found for client ID: admin` |
| Query-param `?clientId=...&clientSecret=...` on POST | Still hits body-consumption bug |

## Shell gotchas

- **Single-quote URLs**: zsh treats `?` as glob. Always `'http://localhost:8000/api/get-users?owner=lms'`
- **Use `--data-binary`** not `-d`: `-d` strips newlines, breaking `modelText` in RBAC model POST
- **Use `docker compose exec -T`** (no TTY) when capturing psql output in bash variables
