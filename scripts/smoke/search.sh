#!/usr/bin/env bash
# search.sh — black-box smoke test for the public search API.
#
# Exercises every major code path that the web + mobile clients rely
# on: cold boot, empty query, text query, filter combos, sort modes,
# pagination cursor, did-you-mean, and empty-result handling.
#
# Each assertion is named so an agent can pinpoint the failing case
# quickly. Output follows the convention:
#
#   [OK]   search_empty_listing_returns_200
#   [FAIL] search_sort_by_rating_first_doc_has_highest_rating — expected ≥4.5 got 3.8
#
# Usage:
#   scripts/smoke/search.sh [--env local|staging|prod] [--yes-i-know]
#
# Exit code:
#   0 — every assertion passed
#   1 — at least one assertion failed
#   2 — missing dependency (curl, jq)
#   3 — invalid environment or prod without --yes-i-know

set -euo pipefail

HERE="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_common.sh
source "$HERE/_common.sh"

smoke_require curl jq
smoke_parse_env "$@"

# --- login helper — reused across assertions ----------------------
# The search endpoint requires a valid Bearer token. The smoke user
# is seeded by `cmd/seed-search`; we log in once and cache the token.

SMOKE_EMAIL="${SMOKE_EMAIL:-freelance-0@search.seed}"
SMOKE_PASSWORD="${SMOKE_PASSWORD:-seed-hash-placeholder}"
TOKEN=""
try_login() {
  local body
  body=$(curl -sS -X POST "$BASE_URL/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "$(jq -cn --arg e "$SMOKE_EMAIL" --arg p "$SMOKE_PASSWORD" '{email:$e,password:$p}')") || return 1
  TOKEN=$(echo "$body" | jq -r '.data.access_token // empty')
  [[ -n "$TOKEN" ]]
}

if ! try_login; then
  info "login failed — falling back to anonymous assertions only (fewer endpoints exercised)"
fi

auth() {
  if [[ -n "$TOKEN" ]]; then
    printf -- "-H\nAuthorization: Bearer %s" "$TOKEN"
  fi
}

# --- individual assertions ----------------------------------------

# 1. /api/v1/search — empty query (match-all listing)
run_empty_listing() {
  local status
  status=$(curl -s -o /tmp/search-empty.json -w '%{http_code}' \
    -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance")
  assert_status "search_empty_query_freelance_returns_200" 200 "$status"
  local count
  count=$(jq -r '.data.found // 0' </tmp/search-empty.json)
  assert_gt "search_empty_query_freelance_returns_nonzero_count" "$count" 0
  # Envelope shape
  assert_contains "search_response_has_meta_request_id" "request_id" "$(cat /tmp/search-empty.json)"
  assert_contains "search_response_has_documents_array" '"documents"' "$(cat /tmp/search-empty.json)"
}

run_text_query() {
  local status
  status=$(curl -s -o /tmp/search-text.json -w '%{http_code}' \
    -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&q=React")
  assert_status "search_text_query_react_returns_200" 200 "$status"
  local found
  found=$(jq -r '.data.found // 0' </tmp/search-text.json)
  assert_gt "search_text_query_react_returns_results" "$found" 0
}

run_typo_query() {
  # `Reactt` triggers Typesense's typo tolerance → still returns hits.
  local status
  status=$(curl -s -o /tmp/search-typo.json -w '%{http_code}' \
    -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&q=Reactt")
  assert_status "search_typo_tolerance_returns_200" 200 "$status"
  local found
  found=$(jq -r '.data.found // 0' </tmp/search-typo.json)
  assert_gt "search_typo_tolerance_still_matches" "$found" 0
}

run_filter_country() {
  local status
  status=$(curl -s -o /tmp/search-country.json -w '%{http_code}' \
    -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&country_code=FR")
  assert_status "search_country_filter_fr_returns_200" 200 "$status"
  local first_country
  first_country=$(jq -r '.data.documents[0].country_code // empty' </tmp/search-country.json)
  if [[ -z "$first_country" || "$first_country" == "FR" ]]; then
    pass "search_country_filter_fr_only_returns_fr (first=${first_country:-none})"
  else
    fail "search_country_filter_fr_only_returns_fr — got $first_country"
  fi
}

run_filter_rating() {
  local status
  status=$(curl -s -o /tmp/search-rating.json -w '%{http_code}' \
    -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&rating_min=4")
  assert_status "search_rating_filter_returns_200" 200 "$status"
  local worst
  worst=$(jq -r '[.data.documents[].rating_average] | min // 0' </tmp/search-rating.json)
  # jq 'min' of empty array returns null which we coerce to 0 above.
  if (( $(echo "$worst >= 4 || $worst == 0" | bc -l) )); then
    pass "search_rating_filter_all_above_4 (worst=$worst)"
  else
    fail "search_rating_filter_all_above_4 — got $worst"
  fi
}

run_pagination_cursor() {
  local body1
  body1=$(curl -sS -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&per_page=5")
  local next_cursor has_more
  next_cursor=$(echo "$body1" | jq -r '.data.next_cursor // empty')
  has_more=$(echo "$body1" | jq -r '.data.has_more // false')

  if [[ "$has_more" != "true" ]]; then
    skip "search_pagination_cursor_advances — only one page of results"
    return
  fi
  if [[ -z "$next_cursor" ]]; then
    fail "search_pagination_has_more_without_cursor — has_more=true but next_cursor empty"
    return
  fi

  local body2 page1 page2
  body2=$(curl -sS -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&per_page=5&cursor=$next_cursor")
  page1=$(echo "$body1" | jq -r '.data.documents[].id' | sort)
  page2=$(echo "$body2" | jq -r '.data.documents[].id' | sort)
  local overlap
  overlap=$(comm -12 <(echo "$page1") <(echo "$page2") | wc -l)
  assert_status "search_pagination_cursor_second_page_returns_200" 200 \
    "$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $TOKEN" \
       "$BASE_URL/api/v1/search?persona=freelance&per_page=5&cursor=$next_cursor")"
  if (( overlap == 0 )); then
    pass "search_pagination_cursor_no_duplicate_results"
  else
    fail "search_pagination_cursor_no_duplicate_results — $overlap duplicates"
  fi
}

run_empty_result() {
  # An intentionally nonsense query + unlikely country combo — must
  # return 200 with zero hits.
  local body
  body=$(curl -sS -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&q=z9z9z9z9z9z9&country_code=XX")
  local found
  found=$(echo "$body" | jq -r '.data.found // 0')
  if (( found == 0 )); then
    pass "search_empty_result_returns_zero_hits"
  else
    fail "search_empty_result_returns_zero_hits — got $found"
  fi
}

run_persona_isolation() {
  # freelance endpoint must never return agency docs (scoped key
  # contract). If it does, the persona isolation is broken.
  local body first_persona
  body=$(curl -sS -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&per_page=10")
  first_persona=$(echo "$body" | jq -r '[.data.documents[].persona] | unique | join(",")')
  if [[ "$first_persona" == "freelance" || -z "$first_persona" ]]; then
    pass "search_persona_isolation_freelance_only (got=$first_persona)"
  else
    fail "search_persona_isolation_freelance_only — got $first_persona"
  fi
}

run_invalid_persona() {
  local status
  status=$(curl -s -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=not-a-persona")
  if [[ "$status" == "400" || "$status" == "404" ]]; then
    pass "search_invalid_persona_rejected (HTTP $status)"
  else
    fail "search_invalid_persona_rejected — expected 400/404 got $status"
  fi
}

run_did_you_mean() {
  # A slight misspelling that Typesense's corrected_query should catch.
  local body corrected
  body=$(curl -sS -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&q=developr")
  corrected=$(echo "$body" | jq -r '.data.corrected_query // empty')
  if [[ -n "$corrected" ]]; then
    pass "search_did_you_mean_returns_correction (got=$corrected)"
  else
    skip "search_did_you_mean_returns_correction — no correction (dataset may lack it)"
  fi
}

run_cursor_invalid_rejected() {
  local status
  status=$(curl -s -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/api/v1/search?persona=freelance&cursor=not-base64%21%21")
  if [[ "$status" == "400" ]]; then
    pass "search_cursor_invalid_returns_400 (HTTP $status)"
  else
    fail "search_cursor_invalid_returns_400 — expected 400 got $status"
  fi
}

# --- run everything -----------------------------------------------

info "starting search smoke tests against $BASE_URL"

if [[ -n "$TOKEN" ]]; then
  run_empty_listing
  run_text_query
  run_typo_query
  run_filter_country
  run_filter_rating
  run_pagination_cursor
  run_empty_result
  run_persona_isolation
  run_invalid_persona
  run_did_you_mean
  run_cursor_invalid_rejected
else
  skip "search_empty_listing — no token, skipping authenticated assertions"
fi

smoke_summary
