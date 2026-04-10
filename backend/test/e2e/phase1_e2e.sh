#!/bin/bash
#
# Phase 1 backend E2E test — validates the team management auto-org
# provisioning flow end-to-end against a real running backend pointed at
# the isolated marketplace_go_team DB.
#
# The script:
#   1. Starts the backend on port 8084 in the background
#   2. Waits for /health to respond
#   3. Runs 6 test groups sequentially with rigorous pass/fail assertions
#   4. Kills the backend
#   5. Prints a summary and exits 0 on success, 1 on any failure
#
# Run from the repo root:
#   ./backend/test/e2e/phase1_e2e.sh
#
# Requires: curl, jq, a running PostgreSQL on :5435 with marketplace_go_team
# and a running Redis on :6380.

set -uo pipefail

# ---- configuration ----
BACKEND_URL="${BACKEND_URL:-http://localhost:8084}"
BACKEND_PORT=8084
ISOLATED_DB="marketplace_go_team"
PG_PORT=5435
TS=$(date +%s)
BACKEND_LOG="/tmp/phase1-e2e-backend.log"

# ---- colors ----
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ---- counters ----
PASS=0
FAIL=0
FAILURES=()

# ---- assertion helpers ----

assert_eq() {
    local name="$1" expected="$2" actual="$3"
    if [[ "$expected" == "$actual" ]]; then
        printf "  ${GREEN}✓${NC} %-55s = ${CYAN}%s${NC}\n" "$name" "$actual"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} %-55s expected=${YELLOW}%s${NC} actual=${RED}%s${NC}\n" "$name" "$expected" "$actual"
        FAIL=$((FAIL+1))
        FAILURES+=("$name: expected '$expected', got '$actual'")
    fi
}

assert_not_empty() {
    local name="$1" value="$2"
    if [[ -n "$value" && "$value" != "null" ]]; then
        local display="${value:0:50}"
        if [[ ${#value} -gt 50 ]]; then display="${display}..."; fi
        printf "  ${GREEN}✓${NC} %-55s = ${CYAN}%s${NC}\n" "$name" "$display"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} %-55s ${RED}empty or null${NC}\n" "$name"
        FAIL=$((FAIL+1))
        FAILURES+=("$name: empty or null")
    fi
}

assert_empty_or_null() {
    local name="$1" value="$2"
    if [[ -z "$value" || "$value" == "null" ]]; then
        printf "  ${GREEN}✓${NC} %-55s ${CYAN}absent as expected${NC}\n" "$name"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} %-55s ${RED}expected absent, got '${value:0:30}'${NC}\n" "$name"
        FAIL=$((FAIL+1))
        FAILURES+=("$name: expected absent, got '$value'")
    fi
}

assert_count() {
    local name="$1" expected="$2" actual="$3"
    if [[ "$expected" -eq "$actual" ]]; then
        printf "  ${GREEN}✓${NC} %-55s = ${CYAN}%d${NC}\n" "$name" "$actual"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} %-55s expected=${YELLOW}%d${NC} actual=${RED}%d${NC}\n" "$name" "$expected" "$actual"
        FAIL=$((FAIL+1))
        FAILURES+=("$name: expected count=$expected, got=$actual")
    fi
}

assert_contains() {
    local name="$1" needle="$2" haystack="$3"
    if echo "$haystack" | grep -q "$needle"; then
        printf "  ${GREEN}✓${NC} %-55s contains ${CYAN}%s${NC}\n" "$name" "$needle"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} %-55s ${RED}missing '%s'${NC}\n" "$name" "$needle"
        FAIL=$((FAIL+1))
        FAILURES+=("$name: missing '$needle'")
    fi
}

section() {
    echo ""
    printf "${BLUE}${BOLD}════════════════════════════════════════════════════════════════${NC}\n"
    printf "${BLUE}${BOLD}▶ %s${NC}\n" "$1"
    printf "${BLUE}${BOLD}════════════════════════════════════════════════════════════════${NC}\n"
}

info() {
    printf "  ${CYAN}ℹ${NC} %s\n" "$1"
}

# ---- backend lifecycle ----

BACKEND_PID=""

start_backend() {
    section "SETUP — Starting backend on port ${BACKEND_PORT}"
    info "Target DB: ${ISOLATED_DB} (isolated)"
    info "Log file: ${BACKEND_LOG}"

    # Kill anything on the target port first
    fuser -k "${BACKEND_PORT}/tcp" 2>/dev/null || true
    sleep 1

    cd /home/hassad/Documents/marketplaceServiceGo/backend

    DATABASE_URL="postgres://postgres:postgres@localhost:${PG_PORT}/${ISOLATED_DB}?sslmode=disable" \
    PORT=${BACKEND_PORT} \
    JWT_SECRET="dev-marketplace-secret-key-change-in-production-2024" \
    REDIS_URL="redis://localhost:6380" \
    STORAGE_ENDPOINT="192.168.1.156:9000" \
    STORAGE_ACCESS_KEY="minioadmin" \
    STORAGE_SECRET_KEY="minioadmin" \
    STORAGE_BUCKET="marketplace" \
    STORAGE_USE_SSL="false" \
    STORAGE_PUBLIC_URL="http://192.168.1.156:9000/marketplace" \
    SESSION_TTL="336h" \
    ALLOWED_ORIGINS="http://localhost:3001" \
    go run cmd/api/main.go > "${BACKEND_LOG}" 2>&1 &

    BACKEND_PID=$!
    info "Backend PID: ${BACKEND_PID}"

    # Wait for /health (max 30s)
    local ready=0
    for i in {1..30}; do
        if curl -sf "${BACKEND_URL}/health" > /dev/null 2>&1; then
            ready=1
            break
        fi
        sleep 1
    done

    if [[ $ready -eq 0 ]]; then
        printf "  ${RED}✗ Backend did not become healthy in 30s${NC}\n"
        echo "--- Backend log tail ---"
        tail -40 "${BACKEND_LOG}"
        echo "--- end ---"
        return 1
    fi

    printf "  ${GREEN}✓ Backend healthy at ${BACKEND_URL}${NC}\n"
    return 0
}

stop_backend() {
    if [[ -n "$BACKEND_PID" ]]; then
        section "TEARDOWN — Stopping backend"
        kill "$BACKEND_PID" 2>/dev/null || true
        fuser -k "${BACKEND_PORT}/tcp" 2>/dev/null || true
        wait "$BACKEND_PID" 2>/dev/null || true
        info "Backend stopped"
    fi
}

trap stop_backend EXIT

# ---- tests ----

test_agency_register() {
    section "TEST 1 — Agency registration auto-provisions organization with Owner member"

    local email="agency-${TS}@phase1.test"
    info "POST /api/v1/auth/register (role=agency, email=${email})"

    local response
    response=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{
            \"email\": \"${email}\",
            \"password\": \"TestPass1!\",
            \"first_name\": \"Sarah\",
            \"last_name\": \"Connor\",
            \"display_name\": \"Acme Corp\",
            \"role\": \"agency\"
        }")
    if [[ -z "$response" ]]; then
        FAIL=$((FAIL+1))
        FAILURES+=("Agency register returned empty response")
        return 1
    fi

    AGENCY_TOKEN=$(echo "$response" | jq -r '.access_token')
    AGENCY_USER_ID=$(echo "$response" | jq -r '.user.id')

    assert_eq "user.email"               "$email" "$(echo "$response" | jq -r '.user.email')"
    assert_eq "user.role"                "agency" "$(echo "$response" | jq -r '.user.role')"
    assert_eq "user.account_type"        "marketplace_owner" "$(echo "$response" | jq -r '.user.account_type')"
    assert_not_empty "access_token"      "$AGENCY_TOKEN"
    assert_not_empty "refresh_token"     "$(echo "$response" | jq -r '.refresh_token')"
    assert_not_empty "organization"      "$(echo "$response" | jq -c '.organization')"
    assert_eq "organization.type"        "agency" "$(echo "$response" | jq -r '.organization.type')"
    assert_eq "organization.member_role" "owner" "$(echo "$response" | jq -r '.organization.member_role')"
    assert_eq "organization.owner_user_id" "$AGENCY_USER_ID" "$(echo "$response" | jq -r '.organization.owner_user_id')"

    local perm_count
    perm_count=$(echo "$response" | jq '.organization.permissions | length')
    assert_count "organization.permissions length" 21 "$perm_count"

    # Assert a few critical Owner-only permissions are present
    local perms
    perms=$(echo "$response" | jq -r '.organization.permissions | join(",")')
    assert_contains "permissions includes wallet.withdraw" "wallet.withdraw" "$perms"
    assert_contains "permissions includes team.transfer_ownership" "team.transfer_ownership" "$perms"
    assert_contains "permissions includes org.delete" "org.delete" "$perms"
    assert_contains "permissions includes billing.manage" "billing.manage" "$perms"
}

test_enterprise_register() {
    section "TEST 2 — Enterprise registration auto-provisions organization"

    local email="enterprise-${TS}@phase1.test"
    info "POST /api/v1/auth/register (role=enterprise, email=${email})"

    local response
    response=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{
            \"email\": \"${email}\",
            \"password\": \"TestPass1!\",
            \"first_name\": \"John\",
            \"last_name\": \"Smith\",
            \"display_name\": \"Enterprise SAS\",
            \"role\": \"enterprise\"
        }")

    ENTERPRISE_TOKEN=$(echo "$response" | jq -r '.access_token')
    ENTERPRISE_USER_ID=$(echo "$response" | jq -r '.user.id')

    assert_eq "user.role"                "enterprise" "$(echo "$response" | jq -r '.user.role')"
    assert_eq "organization.type"        "enterprise" "$(echo "$response" | jq -r '.organization.type')"
    assert_eq "organization.member_role" "owner" "$(echo "$response" | jq -r '.organization.member_role')"
    assert_eq "organization.owner_user_id" "$ENTERPRISE_USER_ID" "$(echo "$response" | jq -r '.organization.owner_user_id')"
    assert_count "organization.permissions length" 21 "$(echo "$response" | jq '.organization.permissions | length')"
}

test_provider_register() {
    section "TEST 3 — Provider registration creates solo user (no organization)"

    local email="provider-${TS}@phase1.test"
    info "POST /api/v1/auth/register (role=provider, email=${email})"

    local response
    response=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{
            \"email\": \"${email}\",
            \"password\": \"TestPass1!\",
            \"first_name\": \"Marie\",
            \"last_name\": \"Durand\",
            \"role\": \"provider\"
        }")

    PROVIDER_TOKEN=$(echo "$response" | jq -r '.access_token')

    assert_eq "user.role"               "provider" "$(echo "$response" | jq -r '.user.role')"
    assert_eq "user.account_type"       "marketplace_owner" "$(echo "$response" | jq -r '.user.account_type')"
    assert_not_empty "access_token"     "$PROVIDER_TOKEN"
    assert_empty_or_null "organization" "$(echo "$response" | jq -c '.organization')"

    # Decode the JWT payload (base64url) and verify no org_id / org_role claims
    local payload
    payload=$(echo "$PROVIDER_TOKEN" | cut -d'.' -f2 | tr '_-' '/+' | base64 -d 2>/dev/null || true)
    assert_empty_or_null "JWT payload org_id claim"   "$(echo "$payload" | jq -r '.org_id // empty')"
    assert_empty_or_null "JWT payload org_role claim" "$(echo "$payload" | jq -r '.org_role // empty')"
}

test_me_agency() {
    section "TEST 4 — GET /me for Agency returns user + organization"
    info "GET /api/v1/auth/me with Agency Bearer token"

    local response
    response=$(curl -sf -X GET "${BACKEND_URL}/api/v1/auth/me" \
        -H "Authorization: Bearer ${AGENCY_TOKEN}")

    assert_eq "user.id"                  "$AGENCY_USER_ID" "$(echo "$response" | jq -r '.user.id')"
    assert_eq "user.role"                "agency" "$(echo "$response" | jq -r '.user.role')"
    assert_not_empty "organization"      "$(echo "$response" | jq -c '.organization')"
    assert_eq "organization.type"        "agency" "$(echo "$response" | jq -r '.organization.type')"
    assert_eq "organization.member_role" "owner" "$(echo "$response" | jq -r '.organization.member_role')"
    assert_count "organization.permissions length" 21 "$(echo "$response" | jq '.organization.permissions | length')"
}

test_me_provider() {
    section "TEST 5 — GET /me for Provider returns user only (no organization)"
    info "GET /api/v1/auth/me with Provider Bearer token"

    local response
    response=$(curl -sf -X GET "${BACKEND_URL}/api/v1/auth/me" \
        -H "Authorization: Bearer ${PROVIDER_TOKEN}")

    assert_eq "user.role"               "provider" "$(echo "$response" | jq -r '.user.role')"
    assert_eq "user.account_type"       "marketplace_owner" "$(echo "$response" | jq -r '.user.account_type')"
    assert_empty_or_null "organization" "$(echo "$response" | jq -c '.organization // empty')"
}

test_duplicate_email() {
    section "TEST 6 — Duplicate email registration returns 409"

    local email="dup-${TS}@phase1.test"
    info "First registration for ${email}"

    curl -sf -X POST "${BACKEND_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{
            \"email\": \"${email}\",
            \"password\": \"TestPass1!\",
            \"first_name\": \"First\",
            \"last_name\": \"Try\",
            \"display_name\": \"First Try\",
            \"role\": \"agency\"
        }" > /dev/null

    info "Second registration for the same email (should fail 409)"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{
            \"email\": \"${email}\",
            \"password\": \"TestPass1!\",
            \"first_name\": \"Second\",
            \"last_name\": \"Try\",
            \"display_name\": \"Second Try\",
            \"role\": \"enterprise\"
        }")

    assert_eq "HTTP status" "409" "$status"
}

# ---- main ----

printf "${BOLD}\n"
printf "╔════════════════════════════════════════════════════════════════╗\n"
printf "║          PHASE 1 E2E — Team Management backend contract        ║\n"
printf "╚════════════════════════════════════════════════════════════════╝\n"
printf "${NC}\n"

if ! start_backend; then
    exit 1
fi

test_agency_register
test_enterprise_register
test_provider_register
test_me_agency
test_me_provider
test_duplicate_email

# ---- summary ----
section "SUMMARY"
if [[ $FAIL -eq 0 ]]; then
    printf "  ${GREEN}${BOLD}✓ ALL %d ASSERTIONS PASSED${NC}\n" "$PASS"
else
    printf "  ${GREEN}%d passed${NC}, ${RED}${BOLD}%d failed${NC}\n" "$PASS" "$FAIL"
    echo ""
    echo "Failures:"
    for f in "${FAILURES[@]}"; do
        printf "  ${RED}- %s${NC}\n" "$f"
    done
fi
echo ""

# Exit code reflects pass/fail
if [[ $FAIL -gt 0 ]]; then
    exit 1
fi
exit 0
