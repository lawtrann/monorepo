---
name: casdoor-api
description: >
  Provides verified curl patterns for the Casdoor REST API: credential fetching,
  GET/POST resource operations (organizations, apps, models, roles, users), and
  known pitfalls. Use when writing shell scripts that call the Casdoor API or
  when the Casdoor API returns unexpected errors.
disable-model-invocation: false
---

# Casdoor API

## Step 0: Verify before writing any script

```bash
# 1. Health check
curl -sf http://localhost:8000 > /dev/null && echo "Casdoor reachable" || echo "FAIL: Casdoor not running"

# 2. Fetch credentials from DB (they change each volume recreate — never hardcode)
CLIENT_ID=$(docker compose exec -T db psql -U postgres -d platform -tAc \
  "SELECT client_id FROM public.application WHERE name='app-built-in'")
CLIENT_SECRET=$(docker compose exec -T db psql -U postgres -d platform -tAc \
  "SELECT client_secret FROM public.application WHERE name='app-built-in'")
CRED="${CLIENT_ID}:${CLIENT_SECRET}"

# 3. Smoke-test auth
curl -s -u "$CRED" 'http://localhost:8000/api/get-organizations' | \
  python3 -c "import sys,json; d=json.load(sys.stdin); print('Auth OK, orgs:', len(d.get('data') or []))"
```

Expected: `Auth OK, orgs: N`. Errors → check [PITFALLS.md](PITFALLS.md).

## Core patterns

Always use `curl -s -u "$CRED"`. Single-quote URLs (zsh glob). Use `--data-binary` for POST.

```bash
# GET list
curl -s -u "$CRED" 'http://localhost:8000/api/get-organizations'
curl -s -u "$CRED" 'http://localhost:8000/api/get-users?owner=lms'

# GET single (id = owner/name)
curl -s -u "$CRED" 'http://localhost:8000/api/get-organization?id=admin/lms'

# POST create
curl -s -u "$CRED" -X POST 'http://localhost:8000/api/add-organization' \
  -H 'Content-Type: application/json' \
  --data-binary '{"owner":"admin","name":"lms","displayName":"LMS School"}'
```

Success: `{"status":"ok","data":"Affected"}`.

**Full endpoint reference**: See [REFERENCE.md](REFERENCE.md)
**Errors / failed approaches**: See [PITFALLS.md](PITFALLS.md)
