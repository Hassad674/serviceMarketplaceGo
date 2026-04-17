#!/usr/bin/env bash
# _common.sh — helpers shared by every smoke script.
#
# Sourced (not executed). Provides:
#   - env resolution (--env local|staging|prod)
#   - base URL lookup per env
#   - color helpers (pass / fail / skip / info)
#   - assert helpers with named sub-tests
#   - summary counter + exit code
#
# Scripts that source this file get the following globals:
#
#   ENV, BASE_URL, TS_URL, TS_API_KEY  — resolved from the chosen env
#   PASS, FAIL, SKIP                   — counters updated by assert_*
#
# Call `smoke_summary` at the very end of the sourcing script so the
# caller sees a consistent report.

set -euo pipefail

# --- color helpers (tty-only) --------------------------------------

if [[ -t 1 ]]; then
  C_RED=$'\033[31m'
  C_GREEN=$'\033[32m'
  C_YELLOW=$'\033[33m'
  C_BLUE=$'\033[34m'
  C_BOLD=$'\033[1m'
  C_RESET=$'\033[0m'
else
  C_RED='' ; C_GREEN='' ; C_YELLOW='' ; C_BLUE='' ; C_BOLD='' ; C_RESET=''
fi

# --- counters ------------------------------------------------------

PASS=0
FAIL=0
SKIP=0

info()  { printf '%s[INFO]%s  %s\n' "$C_BLUE" "$C_RESET" "$*"; }
pass()  { printf '%s[ OK ]%s  %s\n' "$C_GREEN" "$C_RESET" "$*"; PASS=$((PASS+1)); }
fail()  { printf '%s[FAIL]%s  %s\n' "$C_RED" "$C_RESET" "$*" >&2; FAIL=$((FAIL+1)); }
skip()  { printf '%s[SKIP]%s  %s\n' "$C_YELLOW" "$C_RESET" "$*"; SKIP=$((SKIP+1)); }

# --- dependency check ---------------------------------------------

smoke_require() {
  for bin in "$@"; do
    if ! command -v "$bin" >/dev/null 2>&1; then
      printf '%s[MISSING]%s  %s is required but not installed\n' "$C_RED" "$C_RESET" "$bin" >&2
      exit 2
    fi
  done
}

# --- env parsing ---------------------------------------------------
#
# Usage:   smoke_parse_env "$@"
# Reads --env local|staging|prod and --yes-i-know; resolves BASE_URL.
# Aborts on prod unless --yes-i-know is also passed.

smoke_parse_env() {
  ENV="local"
  local ack=""
  local args=("$@")
  local i=0
  while [[ $i -lt ${#args[@]} ]]; do
    case "${args[$i]}" in
      --env)
        ENV="${args[$((i+1))]}"
        i=$((i+2)) ;;
      --env=*)
        ENV="${args[$i]#--env=}"
        i=$((i+1)) ;;
      --yes-i-know)
        ack="yes" ; i=$((i+1)) ;;
      *)
        i=$((i+1)) ;;
    esac
  done

  case "$ENV" in
    local)
      BASE_URL="${MARKETPLACE_BASE_URL:-http://localhost:8083}"
      TS_URL="${TYPESENSE_HOST:-http://localhost:8108}"
      TS_API_KEY="${TYPESENSE_API_KEY:-xyz-dev-master-key-change-in-production}"
      ;;
    staging)
      BASE_URL="${MARKETPLACE_STAGING_URL:?set MARKETPLACE_STAGING_URL}"
      TS_URL="${TYPESENSE_STAGING_URL:?set TYPESENSE_STAGING_URL}"
      TS_API_KEY="${TYPESENSE_STAGING_API_KEY:?set TYPESENSE_STAGING_API_KEY}"
      ;;
    prod)
      if [[ "$ack" != "yes" ]]; then
        fail "refusing to run against prod without --yes-i-know"
        exit 3
      fi
      BASE_URL="${MARKETPLACE_PROD_URL:?set MARKETPLACE_PROD_URL}"
      TS_URL="${TYPESENSE_PROD_URL:?set TYPESENSE_PROD_URL}"
      TS_API_KEY="${TYPESENSE_PROD_API_KEY:?set TYPESENSE_PROD_API_KEY}"
      ;;
    *)
      printf '%s[FATAL]%s unknown env %q (use local|staging|prod)\n' "$C_RED" "$C_RESET" "$ENV" >&2
      exit 3 ;;
  esac

  info "env=$ENV backend=$BASE_URL typesense=$TS_URL"
}

# --- assertion helpers --------------------------------------------

assert_status() {
  local name="$1" expected="$2" actual="$3"
  if [[ "$expected" == "$actual" ]]; then
    pass "$name (HTTP $actual)"
  else
    fail "$name — expected HTTP $expected, got $actual"
  fi
}

assert_contains() {
  local name="$1" needle="$2" haystack="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    pass "$name"
  else
    fail "$name — expected to find $needle"
  fi
}

assert_not_contains() {
  local name="$1" needle="$2" haystack="$3"
  if [[ "$haystack" != *"$needle"* ]]; then
    pass "$name"
  else
    fail "$name — unexpectedly found $needle"
  fi
}

assert_gt() {
  local name="$1" actual="$2" threshold="$3"
  if (( actual > threshold )); then
    pass "$name (got $actual > $threshold)"
  else
    fail "$name — expected >$threshold, got $actual"
  fi
}

# --- summary -------------------------------------------------------

smoke_summary() {
  local total=$((PASS+FAIL+SKIP))
  printf '\n%s=== Summary ===%s\n' "$C_BOLD" "$C_RESET"
  printf '  %stotal%s    %d\n' "$C_BOLD" "$C_RESET" "$total"
  printf '  %spassed%s   %d\n' "$C_GREEN" "$C_RESET" "$PASS"
  printf '  %sfailed%s   %d\n' "$C_RED" "$C_RESET" "$FAIL"
  printf '  %sskipped%s  %d\n' "$C_YELLOW" "$C_RESET" "$SKIP"
  if (( FAIL > 0 )); then exit 1; fi
}

# --- HTTP helpers --------------------------------------------------
# http_status URL        → prints the HTTP status code only.
# http_body   URL        → prints the body only.
# http_header URL HEADER → prints the value of HEADER (lowercased key).

http_status() {
  curl -s -o /dev/null -w '%{http_code}' "$@"
}
http_body() {
  curl -s "$@"
}
http_header() {
  local url="$1" hdr="$2"
  curl -s -D - -o /dev/null "$url" \
    | awk -v h="${hdr,,}" 'BEGIN{IGNORECASE=1} tolower($1)==h":" {sub(/^[^:]*: */, ""); sub(/\r$/, ""); print; exit}'
}
