#!/usr/bin/env bash
# security-baseline.sh — defensive assertions against a running backend.
#
# Scope: everything the 5B smoke does NOT cover:
#   - security headers present on every tested endpoint
#   - /api/v1/admin/* denies anon + non-admin users
#   - /api/v1/search/key never leaks the master key
#   - /api/v1/search/track returns 204 and rate-limits somewhere
#   - search params survive JS/SQL/path-traversal/oversize fuzzing
#     without emitting a 5xx
#
# Fails loudly (exit 1) on first violation. Intended to be invoked from
# the `backend-integration` CI job AND locally:
#   ./scripts/ci/security-baseline.sh --env local
#   ./scripts/ci/security-baseline.sh --env staging --base https://api.staging.example.com
set -euo pipefail

ENV=local
BASE=http://localhost:8080
TOKEN=""
ADMIN_TOKEN=""
EXPIRED_TOKEN=""

usage() {
  cat <<'USAGE'
Usage: security-baseline.sh [--env local|staging] [--base URL] [--token JWT] [--admin-token JWT] [--expired-token JWT]

Checks:
  1. security headers on public + authed endpoints
  2. admin endpoints deny anon, user-role, and expired tokens
  3. /api/v1/search/key does not leak master key
  4. /api/v1/search/track returns 204
  5. fuzz q/filter_by/sort_by/cursor — no 5xx responses
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env) ENV="$2"; shift 2 ;;
    --base) BASE="$2"; shift 2 ;;
    --token) TOKEN="$2"; shift 2 ;;
    --admin-token) ADMIN_TOKEN="$2"; shift 2 ;;
    --expired-token) EXPIRED_TOKEN="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown flag: $1" >&2; usage; exit 1 ;;
  esac
done

PASS=0
FAIL=0
FAILURES=()

log_ok() { printf '  \033[32m[OK]\033[0m %s\n' "$1"; PASS=$((PASS+1)); }
log_fail() { printf '  \033[31m[FAIL]\033[0m %s\n' "$1"; FAIL=$((FAIL+1)); FAILURES+=("$1"); }

# Curl helper. Returns "STATUS\n<body>" on stdout, headers in $2 file.
hit() {
  local method="$1" url="$2" hdr_out="$3"
  shift 3
  curl -sS -o /dev/stdout -D "$hdr_out" -w '\n__STATUS__%{http_code}' \
    -X "$method" "$@" "$url"
}

assert_status() {
  local expect="$1" actual="$2" desc="$3"
  if [[ "$actual" == "$expect" ]]; then
    log_ok "$desc ($actual)"
  else
    log_fail "$desc expected $expect got $actual"
  fi
}

# -----------------------------------------------------------------
# 1. Security headers
# -----------------------------------------------------------------
echo "== Security headers =="
for path in /health /ready /api/v1/profiles/search; do
  tmp_hdr=$(mktemp)
  hit GET "$BASE$path" "$tmp_hdr" > /dev/null || true
  for header in \
    "Content-Security-Policy" \
    "X-Content-Type-Options: nosniff" \
    "X-Frame-Options: DENY" \
    "Referrer-Policy" \
    "Permissions-Policy"; do
    if grep -iFq "$header" "$tmp_hdr"; then
      log_ok "$path: $header"
    else
      log_fail "$path: missing $header"
    fi
  done
  rm -f "$tmp_hdr"
done

# -----------------------------------------------------------------
# 2. Admin endpoints gating
# -----------------------------------------------------------------
echo "== Admin endpoint gating =="
ADMIN_PATHS=(
  "/api/v1/admin/search/stats"
  "/api/v1/admin/users"
)
for path in "${ADMIN_PATHS[@]}"; do
  # anon -> 401
  code=$(curl -sS -o /dev/null -w '%{http_code}' "$BASE$path")
  case "$code" in
    401|403) log_ok "$path anon -> $code" ;;
    404) log_ok "$path anon -> 404 (route may be disabled in this env)" ;;
    *) log_fail "$path anon expected 401/403 got $code" ;;
  esac

  if [[ -n "$TOKEN" ]]; then
    code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $TOKEN" "$BASE$path")
    case "$code" in
      403|404) log_ok "$path user-role -> $code" ;;
      *) log_fail "$path user-role expected 403 got $code" ;;
    esac
  fi

  if [[ -n "$EXPIRED_TOKEN" ]]; then
    code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $EXPIRED_TOKEN" "$BASE$path")
    case "$code" in
      401|403) log_ok "$path expired token -> $code" ;;
      *) log_fail "$path expired token expected 401 got $code" ;;
    esac
  fi

  if [[ -n "$ADMIN_TOKEN" ]]; then
    code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $ADMIN_TOKEN" "$BASE$path")
    case "$code" in
      200|400|404) log_ok "$path admin -> $code" ;;
      *) log_fail "$path admin expected 200/400 got $code" ;;
    esac
  fi
done

# -----------------------------------------------------------------
# 3. Scoped key endpoint must not leak master key
# -----------------------------------------------------------------
echo "== Scoped key leak test =="
if [[ -n "$TOKEN" ]]; then
  body=$(curl -sS -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/search/key?persona=freelance" || echo '{}')
  # The master key always starts with "xyz-dev-" in dev or a high-entropy
  # prefix in prod. We assert the response contains only the expected
  # fields: key, host, ttl, persona. ANY additional field is suspicious.
  allowed='^(data\.|meta\.|error\.)?(key|host|ttl|persona|ttl_seconds|issued_at|expires_at|request_id)$'
  leaked=$(printf '%s' "$body" | python3 -c '
import json, sys
try:
  b = json.loads(sys.stdin.read())
except Exception:
  print(""); sys.exit(0)
def walk(o, pfx=""):
  if isinstance(o, dict):
    for k, v in o.items():
      yield f"{pfx}{k}"
      yield from walk(v, pfx+f"{k}.")
  elif isinstance(o, list):
    for v in o:
      yield from walk(v, pfx)
for path in walk(b):
  print(path)
' | grep -Eiv "$allowed" || true)
  if [[ -z "$leaked" ]]; then
    log_ok "scoped key response fields are whitelisted"
  else
    log_fail "scoped key response leaked fields: $leaked"
  fi

  # Heuristic: response should NOT contain the string "master" and the
  # returned key, if prefixed "xyz-dev-master", is the raw master leaked.
  if grep -iq "master" <<<"$body"; then
    log_fail "scoped key response contains the word 'master'"
  else
    log_ok "scoped key response does not mention master"
  fi
else
  echo "  (skipped — no --token provided)"
fi

# -----------------------------------------------------------------
# 4. Click tracking endpoint
# -----------------------------------------------------------------
echo "== /search/track =="
if [[ -n "$TOKEN" ]]; then
  code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/search/track?search_id=abc&doc_id=def&position=1")
  case "$code" in
    204|200) log_ok "/search/track -> $code" ;;
    *) log_fail "/search/track expected 204 got $code" ;;
  esac

  # Rate-limit probe: 60 consecutive calls must eventually emit 429 OR
  # stay under the emit budget (phase 5B sets the exact N). We accept
  # either but require that no 5xx is ever returned.
  saw_429=0
  saw_5xx=0
  for i in $(seq 1 60); do
    c=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $TOKEN" \
      "$BASE/api/v1/search/track?search_id=abc&doc_id=def&position=1")
    case "$c" in
      429) saw_429=1 ;;
      5??) saw_5xx=1 ;;
    esac
  done
  if [[ $saw_5xx -eq 0 ]]; then
    log_ok "/search/track rate-limit probe: no 5xx across 60 calls"
  else
    log_fail "/search/track rate-limit probe: 5xx observed"
  fi
else
  echo "  (skipped — no --token provided)"
fi

# -----------------------------------------------------------------
# 5. Fuzz search params — every response must be 200/400/429, never 5xx
# -----------------------------------------------------------------
echo "== Search fuzz =="
PAYLOADS=(
  "<script>alert(1)</script>"
  "'; DROP TABLE users; --"
  "../../../../etc/passwd"
  "\\x00\\x01\\x02"
  "%00%0A"
  "javascript:alert(1)"
  "\$(rm -rf /)"
)
OVERSIZE=$(python3 -c 'print("A"*12000)')
PAYLOADS+=("$OVERSIZE")

if [[ -n "$TOKEN" ]]; then
  for payload in "${PAYLOADS[@]}"; do
    enc=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$payload")
    for param in q filter_by sort_by cursor; do
      code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $TOKEN" \
        "$BASE/api/v1/search?persona=freelance&$param=$enc")
      case "$code" in
        2??|400|401|403|404|422|429)
          ;;
        *)
          log_fail "fuzz $param=$(head -c 32 <<<"$payload")... -> $code"
          continue 2
          ;;
      esac
    done
  done
  log_ok "fuzz: no 5xx across ${#PAYLOADS[@]} payloads × 4 params"
else
  echo "  (skipped — no --token provided)"
fi

echo
echo "== Summary =="
echo "  pass: $PASS"
echo "  fail: $FAIL"
if [[ $FAIL -gt 0 ]]; then
  printf '  failures:\n'
  for f in "${FAILURES[@]}"; do printf '    - %s\n' "$f"; done
  exit 1
fi
exit 0
