# Casdoor API — Reference

## Contents
- All GET endpoints
- All POST create endpoints
- Update pattern (GET → patch → POST)
- Idempotency check pattern
- Infrastructure facts

## All GET endpoints

```bash
curl -s -u "$CRED" 'http://localhost:8000/api/get-organizations'
curl -s -u "$CRED" 'http://localhost:8000/api/get-users?owner=lms'
curl -s -u "$CRED" 'http://localhost:8000/api/get-models?owner=built-in'
curl -s -u "$CRED" 'http://localhost:8000/api/get-certs?owner=admin'
curl -s -u "$CRED" 'http://localhost:8000/api/get-organization?id=admin/lms'
curl -s -u "$CRED" 'http://localhost:8000/api/get-application?id=admin/platform'
curl -s -u "$CRED" 'http://localhost:8000/api/get-role?id=lms/teacher'
```

## All POST create endpoints

```bash
# Organization (owner always "admin")
curl -s -u "$CRED" -X POST 'http://localhost:8000/api/add-organization' \
  -H 'Content-Type: application/json' \
  --data-binary '{"owner":"admin","name":"lms","displayName":"LMS School","passwordType":"plain","passwordOptions":["AtLeast6"],"countryCodes":["VN"],"languages":["en"]}'

# Application (reuse cert-built-in)
curl -s -u "$CRED" -X POST 'http://localhost:8000/api/add-application' \
  -H 'Content-Type: application/json' \
  --data-binary '{"owner":"admin","name":"platform","displayName":"Platform App","organization":"lms","cert":"cert-built-in","enablePassword":true,"enableSignUp":false,"redirectUris":["http://localhost:3000/callback"],"expireInHours":168,"grantTypes":["authorization_code","password"],"tokenFormat":"JWT-Standard"}'

# Model (owner = org name)
curl -s -u "$CRED" -X POST 'http://localhost:8000/api/add-model' \
  -H 'Content-Type: application/json' \
  --data-binary '{"owner":"lms","name":"rbac-with-domains","displayName":"RBAC with Domains","modelText":"[request_definition]\nr = sub, dom, obj, act\n\n[policy_definition]\np = sub, dom, obj, act\n\n[role_definition]\ng = _, _, _\n\n[policy_effect]\ne = some(where (p.eft == allow))\n\n[matchers]\nm = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act"}'

# Role (owner = org name)
curl -s -u "$CRED" -X POST 'http://localhost:8000/api/add-role' \
  -H 'Content-Type: application/json' \
  --data-binary '{"owner":"lms","name":"teacher","displayName":"Teacher"}'

# Role with sub-roles (refs use "owner/name")
curl -s -u "$CRED" -X POST 'http://localhost:8000/api/add-role' \
  -H 'Content-Type: application/json' \
  --data-binary '{"owner":"lms","name":"staff","displayName":"Staff","roles":["lms/teacher","lms/school_admin"]}'

# User (owner = org name)
curl -s -u "$CRED" -X POST 'http://localhost:8000/api/add-user' \
  -H 'Content-Type: application/json' \
  --data-binary '{"owner":"lms","name":"test-teacher","displayName":"Nguyen Van A","email":"teacher@lms.local","password":"123456","type":"normal-user"}'
```

## Update pattern (GET → patch → POST)

```bash
ROLE_JSON=$(curl -s -u "$CRED" 'http://localhost:8000/api/get-role?id=lms/teacher' | python3 -c "
import sys, json
d = json.load(sys.stdin)
r = d['data']
r['users'] = ['lms/test-teacher']
print(json.dumps(r))
")
curl -s -u "$CRED" -X POST 'http://localhost:8000/api/update-role?id=lms/teacher' \
  -H 'Content-Type: application/json' \
  --data-binary "$ROLE_JSON"
```

## Idempotency check pattern

GET first — duplicate creates return an error.

```bash
EXISTING=$(curl -s -u "$CRED" 'http://localhost:8000/api/get-organization?id=admin/lms')
echo "$EXISTING" | python3 -c "import sys,json; d=json.load(sys.stdin); exit(0 if d.get('data') else 1)" \
  && echo "Already exists, skipping" \
  || echo "Does not exist — create it"
```

## Infrastructure facts

- **Tables in `public` schema** (not `casdoor`) — XORM creates there because app.conf connects as `postgres`
- **Built-in defaults (v1.777.0)**:
  - App: `app-built-in` (owner: `admin`) — source of clientId/clientSecret
  - Cert: `cert-built-in` (RSA256, owner: `admin`) — reuse for custom apps
  - Admin user: `built-in/admin`, password `123` (login API broken — use clientId/clientSecret)
  - Models: `api-model-built-in`, `user-model-built-in`
