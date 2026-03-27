#!/usr/bin/env bash
# Verify end-to-end: login as test-teacher, decode JWT, confirm claims.
# Requires: Casdoor running + seed-casdoor.sh already executed.
set -euo pipefail

BASE="http://localhost:8000"

echo "=== Verify JWT End-to-End ==="

# Check Casdoor is running
curl -sf "${BASE}" > /dev/null || { echo "ERROR: Casdoor not reachable at ${BASE}"; exit 1; }

# Fetch admin credentials from DB (app-built-in — never hardcode)
echo "Fetching admin credentials from DB..."
CLIENT_ID=$(docker compose exec -T db psql -U postgres -d platform -tAc \
  "SELECT client_id FROM public.application WHERE name='app-built-in'")
CLIENT_SECRET=$(docker compose exec -T db psql -U postgres -d platform -tAc \
  "SELECT client_secret FROM public.application WHERE name='app-built-in'")
CRED="${CLIENT_ID}:${CLIENT_SECRET}"

# Fetch platform app credentials (created by seed-casdoor.sh)
echo "Fetching platform app credentials..."
PLATFORM_APP=$(curl -s -u "$CRED" 'http://localhost:8000/api/get-application?id=admin/platform')
PLATFORM_CLIENT_ID=$(echo "$PLATFORM_APP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['clientId'])")
PLATFORM_CLIENT_SECRET=$(echo "$PLATFORM_APP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['clientSecret'])")
echo "Platform client_id: ${PLATFORM_CLIENT_ID}"

# --- Login via OAuth2 password grant ---
echo ""
echo "--- Login as test-teacher (OAuth2 password grant) ---"
TOKEN_RESPONSE=$(curl -s -X POST "${BASE}/api/login/oauth/access_token" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "grant_type=password" \
  --data-urlencode "username=test-teacher" \
  --data-urlencode "password=123456" \
  --data-urlencode "scope=openid profile" \
  --data-urlencode "client_id=${PLATFORM_CLIENT_ID}" \
  --data-urlencode "client_secret=${PLATFORM_CLIENT_SECRET}")

ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | python3 -c "
import sys, json
raw = sys.stdin.read().strip()
try:
    d = json.loads(raw)
except Exception:
    print('ERROR: non-JSON response: ' + raw[:200], file=sys.stderr)
    sys.exit(1)
if 'access_token' not in d:
    print('ERROR: No access_token in response: ' + str(d), file=sys.stderr)
    sys.exit(1)
print(d['access_token'])
")

echo "[OK] Received access_token"

# --- Decode JWT payload ---
echo ""
echo "--- Decoding JWT payload ---"
JWT_PAYLOAD=$(echo "$ACCESS_TOKEN" | python3 -c "
import sys, json, base64
token = sys.stdin.read().strip()
parts = token.split('.')
if len(parts) != 3:
    print('ERROR: Not a valid JWT (got ' + str(len(parts)) + ' parts)', file=sys.stderr)
    sys.exit(1)
payload = parts[1]
# Fix base64url padding
rem = len(payload) % 4
if rem:
    payload += '=' * (4 - rem)
payload = payload.replace('-', '+').replace('_', '/')
decoded = base64.b64decode(payload).decode('utf-8')
# Pretty print
print(json.dumps(json.loads(decoded), indent=2))
")

echo "JWT Payload:"
echo "$JWT_PAYLOAD"

# --- Assert claims ---
echo ""
echo "--- Asserting JWT claims ---"
echo "$JWT_PAYLOAD" | python3 -c "
import sys, json
claims = json.load(sys.stdin)
errors = []

# owner == lms (Casdoor JWT-Standard: 'owner' field = org name)
owner = claims.get('owner', '')
if owner != 'lms':
    errors.append(f'owner: expected lms, got {owner!r}')
else:
    print(f'[PASS] owner == lms')

# name is present
name = claims.get('name', '')
if not name:
    errors.append('name: expected non-empty display name')
else:
    print(f'[PASS] name == {name!r}')

# roles contains teacher
# Casdoor JWT format: roles is an array of role objects {\"owner\":..., \"name\":...}
roles = claims.get('roles') or []
role_names = []
for r in roles:
    if isinstance(r, dict):
        role_names.append(r.get('name', ''))
    elif isinstance(r, str):
        role_names.append(r.split('/')[-1] if '/' in r else r)
if 'teacher' not in role_names:
    errors.append(f'roles: expected teacher, got {role_names}')
else:
    print(f'[PASS] roles contains teacher (all roles: {role_names})')

if errors:
    print('', file=sys.stderr)
    for e in errors:
        print(f'[FAIL] {e}', file=sys.stderr)
    sys.exit(1)

print('')
print('[PASS] All JWT claims verified')
"

echo ""
echo "=== JWT Verification Complete ==="
