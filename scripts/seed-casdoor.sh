#!/usr/bin/env bash
# Seed Casdoor with org, application, RBAC model, roles, and test user.
# Idempotent: skips resources that already exist.
set -euo pipefail

BASE="http://localhost:8000"

echo "=== Casdoor Seed Script ==="

# Check Casdoor is running
curl -sf "${BASE}" > /dev/null || { echo "ERROR: Casdoor not reachable at ${BASE}"; exit 1; }

# Fetch credentials from DB (they change each volume recreate — never hardcode)
echo "Fetching client credentials from DB..."
CLIENT_ID=$(docker compose exec -T db psql -U postgres -d platform -tAc \
  "SELECT client_id FROM public.application WHERE name='app-built-in'")
CLIENT_SECRET=$(docker compose exec -T db psql -U postgres -d platform -tAc \
  "SELECT client_secret FROM public.application WHERE name='app-built-in'")
CRED="${CLIENT_ID}:${CLIENT_SECRET}"

# Verify auth
curl -s -u "$CRED" "${BASE}/api/get-organizations" | \
  python3 -c "import sys,json; d=json.load(sys.stdin); print('Auth OK, orgs:', len(d.get('data') or []))"

# --- Helpers ---

# Returns 0 if resource data exists, 1 if not
resource_exists() {
  curl -s -u "$CRED" "$1" | python3 -c "
import sys, json
d = json.load(sys.stdin)
exit(0 if d.get('data') else 1)
"
}

# Asserts response has status=ok, exits on failure
assert_ok() {
  local resp="$1"
  local label="$2"
  echo "$resp" | python3 -c "
import sys, json
d = json.load(sys.stdin)
if d.get('status') == 'ok':
    print('[OK] $label')
else:
    print('[FAIL] $label: ' + str(d), file=sys.stderr)
    sys.exit(1)
"
}

# --- 1. Organization: lms ---
echo ""
echo "--- Organization: lms ---"
if resource_exists "${BASE}/api/get-organization?id=admin/lms"; then
  echo "[SKIP] org lms already exists"
else
  RESP=$(curl -s -u "$CRED" -X POST "${BASE}/api/add-organization" \
    -H 'Content-Type: application/json' \
    --data-binary '{"owner":"admin","name":"lms","displayName":"LMS School","passwordType":"plain","passwordOptions":["AtLeast6"],"countryCodes":["VN"],"languages":["en"]}')
  assert_ok "$RESP" "create org lms"
fi

# --- 2. Application: platform ---
# tokenFormat=JWT (not JWT-Standard) so that roles/permissions are included in the token claims.
echo ""
echo "--- Application: platform ---"
if resource_exists "${BASE}/api/get-application?id=admin/platform"; then
  echo "[SKIP] application platform already exists — ensuring tokenFormat=JWT"
  APP_JSON=$(curl -s -u "$CRED" 'http://localhost:8000/api/get-application?id=admin/platform' | python3 -c "
import sys, json
d = json.load(sys.stdin)
app = d['data']
app['tokenFormat'] = 'JWT'
print(json.dumps(app))
")
  RESP=$(curl -s -u "$CRED" -X POST 'http://localhost:8000/api/update-application?id=admin/platform' \
    -H 'Content-Type: application/json' \
    --data-binary "$APP_JSON")
  assert_ok "$RESP" "update application platform tokenFormat"
else
  RESP=$(curl -s -u "$CRED" -X POST "${BASE}/api/add-application" \
    -H 'Content-Type: application/json' \
    --data-binary '{"owner":"admin","name":"platform","displayName":"Platform App","organization":"lms","cert":"cert-built-in","enablePassword":true,"enableSignUp":false,"redirectUris":["http://localhost:3000/callback"],"expireInHours":168,"grantTypes":["authorization_code","password"],"tokenFormat":"JWT"}')
  assert_ok "$RESP" "create application platform"
fi

# --- 3. Casbin RBAC model ---
echo ""
echo "--- Model: rbac-with-domains ---"
if resource_exists "${BASE}/api/get-model?id=lms/rbac-with-domains"; then
  echo "[SKIP] model rbac-with-domains already exists"
else
  RESP=$(curl -s -u "$CRED" -X POST "${BASE}/api/add-model" \
    -H 'Content-Type: application/json' \
    --data-binary '{"owner":"lms","name":"rbac-with-domains","displayName":"RBAC with Domains","modelText":"[request_definition]\nr = sub, dom, obj, act\n\n[policy_definition]\np = sub, dom, obj, act\n\n[role_definition]\ng = _, _, _\n\n[policy_effect]\ne = some(where (p.eft == allow))\n\n[matchers]\nm = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act"}')
  assert_ok "$RESP" "create model rbac-with-domains"
fi

# --- 4. Roles: admin, school_admin, teacher, student, parent ---
echo ""
echo "--- Roles ---"
for ROLE in admin school_admin teacher student parent; do
  if resource_exists "${BASE}/api/get-role?id=lms/${ROLE}"; then
    echo "[SKIP] role ${ROLE} already exists"
  else
    case "$ROLE" in
      admin)        DISPLAY="Admin" ;;
      school_admin) DISPLAY="School Admin" ;;
      teacher)      DISPLAY="Teacher" ;;
      student)      DISPLAY="Student" ;;
      parent)       DISPLAY="Parent" ;;
    esac
    RESP=$(curl -s -u "$CRED" -X POST "${BASE}/api/add-role" \
      -H 'Content-Type: application/json' \
      --data-binary "{\"owner\":\"lms\",\"name\":\"${ROLE}\",\"displayName\":\"${DISPLAY}\"}")
    assert_ok "$RESP" "create role ${ROLE}"
  fi
done

# --- 5. Role hierarchy: staff = teacher + school_admin ---
echo ""
echo "--- Role hierarchy: staff ---"
if resource_exists "${BASE}/api/get-role?id=lms/staff"; then
  echo "[SKIP] role staff already exists"
else
  RESP=$(curl -s -u "$CRED" -X POST "${BASE}/api/add-role" \
    -H 'Content-Type: application/json' \
    --data-binary '{"owner":"lms","name":"staff","displayName":"Staff","roles":["lms/teacher","lms/school_admin"]}')
  assert_ok "$RESP" "create role staff"
fi

# --- 6. Test user: test-teacher ---
echo ""
echo "--- Test user: test-teacher ---"
if resource_exists "${BASE}/api/get-user?id=lms/test-teacher"; then
  echo "[SKIP] user test-teacher already exists"
else
  RESP=$(curl -s -u "$CRED" -X POST "${BASE}/api/add-user" \
    -H 'Content-Type: application/json' \
    --data-binary '{"owner":"lms","name":"test-teacher","displayName":"Nguyen Van A","email":"teacher@lms.local","password":"123456","type":"normal-user"}')
  assert_ok "$RESP" "create user test-teacher"
fi

# --- 7. Assign test-teacher to teacher role ---
echo ""
echo "--- Assign test-teacher to teacher role ---"
ROLE_JSON=$(curl -s -u "$CRED" "${BASE}/api/get-role?id=lms/teacher" | python3 -c "
import sys, json
d = json.load(sys.stdin)
r = d['data']
users = r.get('users') or []
if 'lms/test-teacher' not in users:
    users.append('lms/test-teacher')
r['users'] = users
print(json.dumps(r))
")
RESP=$(curl -s -u "$CRED" -X POST "${BASE}/api/update-role?id=lms/teacher" \
  -H 'Content-Type: application/json' \
  --data-binary "$ROLE_JSON")
assert_ok "$RESP" "assign test-teacher to teacher role"

echo ""
echo "=== Seeding complete ==="
