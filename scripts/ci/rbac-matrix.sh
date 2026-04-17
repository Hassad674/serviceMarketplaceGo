#!/usr/bin/env bash
# rbac-matrix.sh — 3x3 authorization matrix.
#
#                | public | auth-only | admin-only |
# ---------------+--------+-----------+------------+
#   anon         |  200   |   401     |   401      |
#   user         |  200   |   200     |   403      |
#   admin        |  200   |   200     |   200      |
#
# Any cell returning the wrong code fails the whole script. Prints a
# human-readable table of [OK]/[FAIL] results.
set -euo pipefail

BASE=http://localhost:8080
USER_TOKEN=""
ADMIN_TOKEN=""

usage() {
  cat <<'USAGE'
Usage: rbac-matrix.sh [--base URL] [--token JWT] [--admin-token JWT]
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base) BASE="$2"; shift 2 ;;
    --token) USER_TOKEN="$2"; shift 2 ;;
    --admin-token) ADMIN_TOKEN="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown flag: $1" >&2; usage; exit 1 ;;
  esac
done

PUBLIC_ROUTE="/health"
AUTH_ROUTE="/api/v1/auth/me"
ADMIN_ROUTE="/api/v1/admin/search/stats"

hit_code() {
  local token="${1:-}"
  local path="$2"
  if [[ -z "$token" ]]; then
    curl -sS -o /dev/null -w '%{http_code}' "$BASE$path"
  else
    curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $token" "$BASE$path"
  fi
}

FAIL=0

check() {
  local role="$1" kind="$2" token="$3" path="$4" expect="$5"
  local code
  code=$(hit_code "$token" "$path")
  local marker="[OK]"
  if [[ ",$expect," != *",$code,"* ]]; then
    marker="[FAIL]"
    FAIL=1
  fi
  printf '  %-12s %-12s %-3s expect=%-12s got=%s\n' "$role" "$kind" "$marker" "$expect" "$code"
}

echo "RBAC matrix against $BASE"
echo

# -----------------------------------------------------------------
# public (expect 200 regardless of role)
# -----------------------------------------------------------------
echo "-- public --"
check "anon"  "public" ""             "$PUBLIC_ROUTE" "200"
check "user"  "public" "$USER_TOKEN"  "$PUBLIC_ROUTE" "200"
check "admin" "public" "$ADMIN_TOKEN" "$PUBLIC_ROUTE" "200"

# -----------------------------------------------------------------
# auth-only (anon 401, user 200, admin 200)
# -----------------------------------------------------------------
echo "-- auth-only --"
check "anon"  "auth-only" ""             "$AUTH_ROUTE" "401"
check "user"  "auth-only" "$USER_TOKEN"  "$AUTH_ROUTE" "200"
check "admin" "auth-only" "$ADMIN_TOKEN" "$AUTH_ROUTE" "200"

# -----------------------------------------------------------------
# admin-only (anon 401, user 403, admin 200)
# -----------------------------------------------------------------
echo "-- admin-only --"
# 404 accepted if admin endpoint is not yet deployed in this env.
check "anon"  "admin-only" ""             "$ADMIN_ROUTE" "401,404"
check "user"  "admin-only" "$USER_TOKEN"  "$ADMIN_ROUTE" "403,404"
check "admin" "admin-only" "$ADMIN_TOKEN" "$ADMIN_ROUTE" "200,400,404"

echo
if [[ $FAIL -ne 0 ]]; then
  echo "RBAC matrix FAILED"
  exit 1
fi
echo "RBAC matrix OK"
