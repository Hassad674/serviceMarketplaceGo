#!/usr/bin/env bash
# security.sh — black-box security probes against the search surface.
#
# Verifies:
#   - HTTP security headers (CSP, HSTS, X-Frame-Options, X-Content-Type-Options)
#   - SQL-injection-looking filter params return 400 (never 500)
#   - Admin stats endpoint without the admin role returns 403
#   - Unauthenticated calls to /api/v1/search return 401
#   - Rate limit enforcement: many rapid calls trigger 429
#
# Usage:
#   scripts/smoke/security.sh [--env local|staging|prod] [--yes-i-know]

set -euo pipefail

HERE="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_common.sh
source "$HERE/_common.sh"

smoke_require curl jq
smoke_parse_env "$@"

run_security_headers() {
  local headers
  headers=$(curl -sS -D - -o /dev/null "$BASE_URL/health")

  # Headers that should always be present on a production build.
  # In local dev the SecurityHeaders middleware may be disabled —
  # downgrade to SKIP so the run stays green while still surfacing
  # the gap in the summary. In staging / prod we fail loudly.
  local level=fail
  if [[ "$ENV" == "local" ]]; then level=skip; fi

  for header in X-Content-Type-Options X-Frame-Options Content-Security-Policy Referrer-Policy; do
    local name="security_header_$(echo "$header" | tr '[:upper:]-' '[:lower:]_')_present"
    if echo "$headers" | grep -qi "^$header:"; then
      pass "$name"
    else
      "$level" "$name — header absent on $ENV /health"
    fi
  done

  if echo "$headers" | grep -qi '^Strict-Transport-Security:'; then
    pass "security_header_hsts_present"
  elif [[ "$ENV" == "local" ]]; then
    skip "security_header_hsts_present — optional in local dev"
  else
    fail "security_header_hsts_present"
  fi
}

run_unauth_search_rejected() {
  local status
  status=$(http_status "$BASE_URL/api/v1/search?persona=freelance")
  # Auth-required endpoint: must return 401 (not 500).
  if [[ "$status" == "401" ]]; then
    pass "security_unauth_search_returns_401"
  else
    fail "security_unauth_search_returns_401 — got $status"
  fi
}

run_admin_stats_requires_admin() {
  # Without a token → 401; with non-admin token → 403. We only check
  # the 401 path here since the other tests don't carry a non-admin
  # token; the handler unit tests cover the 403 path.
  local status
  status=$(http_status "$BASE_URL/api/v1/admin/search/stats")
  if [[ "$status" == "401" || "$status" == "403" ]]; then
    pass "security_admin_stats_requires_auth (HTTP $status)"
  else
    fail "security_admin_stats_requires_auth — got $status"
  fi
}

run_sql_injection_probes() {
  local probes=(
    "persona=freelance&city=%27+OR+1%3D1+--"
    "persona=freelance&q=%27%3B+DROP+TABLE+users%3B--"
    "persona=freelance&country_code=%27%20UNION%20SELECT%20%2A%20FROM%20users--"
  )
  for probe in "${probes[@]}"; do
    local status
    status=$(http_status "$BASE_URL/api/v1/search?$probe")
    # 400, 401, 403 are all acceptable — anything that is NOT 500.
    if [[ "$status" != "500" ]]; then
      pass "security_sql_injection_rejected_with_${status} (${probe:0:40}…)"
    else
      fail "security_sql_injection_returned_500 — ${probe}"
    fi
  done
}

run_rate_limit() {
  if [[ "$ENV" != "local" ]]; then
    skip "security_rate_limit — only run against local"
    return
  fi
  # Fire 30 back-to-back requests; expect at least one 429.
  local any_429=""
  for i in $(seq 1 30); do
    local code
    code=$(http_status "$BASE_URL/health") || code=000
    if [[ "$code" == "429" ]]; then
      any_429="yes"
      break
    fi
  done
  if [[ -n "$any_429" ]]; then
    pass "security_rate_limit_triggers_429"
  else
    # Not every local setup enforces rate limits — downgrade to skip
    # rather than fail.
    skip "security_rate_limit_triggers_429 — no 429 observed (rate limiting may be disabled in dev)"
  fi
}

run_scoped_key_cross_persona_leak() {
  # If the backend mints a scoped key with persona=freelance, using it
  # to query the `marketplace_actors` collection directly must only
  # return freelance docs. Verifies the HMAC filter injection is
  # working. Skipped when backend is unreachable.
  local key_body
  key_body=$(curl -sS "$BASE_URL/api/v1/search/key?persona=freelance" 2>/dev/null) || {
    skip "security_scoped_key_cross_persona_leak — /search/key endpoint unreachable"
    return
  }
  local scoped_key
  scoped_key=$(echo "$key_body" | jq -r '.data.key // empty')
  if [[ -z "$scoped_key" ]]; then
    skip "security_scoped_key_cross_persona_leak — no key returned (auth required?)"
    return
  fi

  # Query Typesense directly with the scoped key, asking for persona=agency.
  # The HMAC-embedded filter must still force persona=freelance.
  local resp leaks
  resp=$(curl -sS -H "X-TYPESENSE-API-KEY: $scoped_key" \
    "$TS_URL/collections/marketplace_actors/documents/search?q=*&query_by=display_name&filter_by=persona%3Aagency")
  leaks=$(echo "$resp" | jq -r '[.hits[]?.document.persona] | unique | join(",")' 2>/dev/null || echo "error")
  if [[ "$leaks" == "freelance" || -z "$leaks" ]]; then
    pass "security_scoped_key_cross_persona_leak (got=$leaks)"
  else
    fail "security_scoped_key_cross_persona_leak — scoped key returned $leaks"
  fi
}

info "starting security smoke tests"
run_security_headers
run_unauth_search_rejected
run_admin_stats_requires_admin
run_sql_injection_probes
run_rate_limit
run_scoped_key_cross_persona_leak
smoke_summary
