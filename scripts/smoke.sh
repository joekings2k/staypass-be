#!/usr/bin/env bash
# End-to-end smoke test for the auth + estate-scoping flow.
#
#   onboarding -> login -> authed routes -> refresh -> old refresh rejected
#   -> estate deactivated => 403 -> reactivated => 200 -> logout -> refresh rejected
#
# Usage: make run (in another shell), then ./scripts/smoke.sh
# Env:   BASE_URL (default http://localhost:$APP_PORT or :8080)
# Needs: curl, jq, and either psql on PATH or the docker compose postgres service.
#
# The refresh cookie is Secure, so curl won't replay it over plain http from a
# cookie jar. The script reads Set-Cookie itself and sends it explicitly, which
# also lets it replay the *old* token on purpose.

set -euo pipefail

cd "$(dirname "$0")/.."
if [[ -f .env ]]; then
  set -a; source .env; set +a
fi

BASE_URL="${BASE_URL:-http://localhost:${APP_PORT:-8080}}"
API="$BASE_URL/api/v1"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

pass=0
fail() { echo "FAIL: $*" >&2; exit 1; }
ok()   { echo "  ok  $*"; pass=$((pass + 1)); }

# request METHOD PATH [curl args...] -> sets $status, body in $TMP/body, headers in $TMP/headers
request() {
  local method="$1" path="$2"; shift 2
  status="$(curl -sS -o "$TMP/body" -D "$TMP/headers" -w '%{http_code}' -X "$method" "$API$path" "$@")"
}

expect() {
  local want="$1" what="$2"
  if [[ "$status" != "$want" ]]; then
    echo "--- response body:" >&2; cat "$TMP/body" >&2; echo >&2
    fail "$what: expected $want, got $status"
  fi
  ok "$what -> $status"
}

refresh_cookie_from_headers() {
  grep -i '^set-cookie: refreshToken=' "$TMP/headers" | head -1 | sed -E 's/^[^=]+=([^;]*).*/\1/' | tr -d '\r'
}

sql() {
  if command -v psql >/dev/null 2>&1; then
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -qtAc "$1"
  else
    docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -qtAc "$1"
  fi
}

command -v jq >/dev/null || fail "jq is required"
curl -sf "$BASE_URL/health" >/dev/null || fail "API not reachable at $BASE_URL (is 'make run' up?)"

suffix="$(date +%s)$RANDOM"
email="smoke+$suffix@example.com"
phone="+234${suffix: -10}"
password="smoke-pass-123"

echo "== onboarding"
request POST /estate-onboarding -H 'Content-Type: application/json' -d "$(jq -n \
  --arg e "$email" --arg p "$phone" --arg pw "$password" --arg n "Smoke Estate $suffix" \
  '{fullName:"Smoke Tester", email:$e, phone:$p, password:$pw, estateName:$n, city:"Lagos", state:"Lagos"}')"
expect 201 "POST /estate-onboarding"
estate_id="$(jq -r '.EstateID // .estateId // .data.EstateID // empty' "$TMP/body")"
[[ -n "$estate_id" ]] || fail "no estate id in onboarding response: $(cat "$TMP/body")"
echo "     estate_id=$estate_id"

echo "== login"
request POST /auth/login -H 'Content-Type: application/json' \
  -d "$(jq -n --arg i "$email" --arg pw "$password" '{identifier:$i, password:$pw}')"
expect 200 "POST /auth/login"
access1="$(jq -r '.access_token' "$TMP/body")"
refresh1="$(refresh_cookie_from_headers)"
[[ -n "$access1" && "$access1" != null ]] || fail "no access token"
[[ -n "$refresh1" ]] || fail "no refresh cookie"

echo "== authed routes"
request GET /me
expect 401 "GET /me without token"
request GET /me -H "Authorization: Bearer $refresh1"
expect 401 "GET /me with a refresh token as Bearer"
request GET /me -H "Authorization: Bearer $access1"
expect 200 "GET /me"
request GET "/estates/$estate_id" -H "Authorization: Bearer $access1"
expect 200 "GET /estates/{id}"
request GET "/estates/00000000-0000-0000-0000-000000000000" -H "Authorization: Bearer $access1"
expect 404 "GET /estates/{unknown id}"

echo "== refresh (rotation)"
request POST /auth/refresh
expect 401 "POST /auth/refresh without cookie"
request POST /auth/refresh -H "Cookie: refreshToken=$refresh1"
expect 200 "POST /auth/refresh"
access2="$(jq -r '.access_token' "$TMP/body")"
refresh2="$(refresh_cookie_from_headers)"
[[ -n "$refresh2" && "$refresh2" != "$refresh1" ]] || fail "refresh token was not rotated"
request GET /me -H "Authorization: Bearer $access2"
expect 200 "GET /me with refreshed access token"
request POST /auth/refresh -H "Cookie: refreshToken=$refresh1"
expect 401 "POST /auth/refresh with OLD (rotated) refresh token"

echo "== estate deactivation"
sql "UPDATE estates SET is_active = false WHERE id = '$estate_id';" >/dev/null
request GET "/estates/$estate_id" -H "Authorization: Bearer $access2"
expect 403 "GET /estates/{id} while is_active=false"
sql "UPDATE estates SET is_active = true WHERE id = '$estate_id';" >/dev/null
request GET "/estates/$estate_id" -H "Authorization: Bearer $access2"
expect 200 "GET /estates/{id} after reactivation"

echo "== logout"
request POST /auth/logout -H "Cookie: refreshToken=$refresh2"
expect 204 "POST /auth/logout"
grep -qi '^set-cookie: refreshToken=;' "$TMP/headers" || fail "logout did not clear the cookie"
ok "logout cleared the refresh cookie"
request POST /auth/refresh -H "Cookie: refreshToken=$refresh2"
expect 401 "POST /auth/refresh after logout"
request POST /auth/logout -H "Cookie: refreshToken=$refresh2"
expect 204 "POST /auth/logout again (idempotent)"

echo
echo "PASS ($pass checks)"
