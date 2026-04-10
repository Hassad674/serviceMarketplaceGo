#!/bin/bash
#
# Phase 2 backend E2E test — validates the team invitation flow end-to-end
# against a real running backend pointed at the isolated marketplace_go_team
# DB.

set -uo pipefail

BACKEND_URL="${BACKEND_URL:-http://localhost:8084}"
BACKEND_PORT=8084
ISOLATED_DB="marketplace_go_team"
PG_PORT=5435
TS=$(date +%s)
BACKEND_LOG="/tmp/phase2-e2e-backend.log"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

PASS=0
FAIL=0
FAILURES=()

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

section() {
    echo ""
    printf "${BLUE}${BOLD}════════════════════════════════════════════════════════════════${NC}\n"
    printf "${BLUE}${BOLD}▶ %s${NC}\n" "$1"
    printf "${BLUE}${BOLD}════════════════════════════════════════════════════════════════${NC}\n"
}

info() { printf "  ${CYAN}ℹ${NC} %s\n" "$1"; }

BACKEND_PID=""

start_backend() {
    section "SETUP — Starting backend on port ${BACKEND_PORT}"
    info "Target DB: ${ISOLATED_DB}"
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
    RESEND_API_KEY="${RESEND_API_KEY:-dev-dummy-key}" \
    RESEND_DEV_REDIRECT_TO="hassad.smara69@gmail.com" \
    go run cmd/api/main.go > "${BACKEND_LOG}" 2>&1 &
    BACKEND_PID=$!
    info "Backend PID: ${BACKEND_PID}"

    for i in {1..30}; do
        if curl -sf "${BACKEND_URL}/health" > /dev/null 2>&1; then
            printf "  ${GREEN}✓ Backend healthy at ${BACKEND_URL}${NC}\n"
            return 0
        fi
        sleep 1
    done
    printf "  ${RED}✗ Backend did not become healthy in 30s${NC}\n"
    tail -40 "${BACKEND_LOG}"
    return 1
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

setup_fixtures() {
    section "FIXTURES — Register Agency owner and Provider"

    local agency_email="agency-p2-${TS}@phase2.test"
    local provider_email="provider-p2-${TS}@phase2.test"

    local agency_resp
    agency_resp=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" -H "X-Auth-Mode: token" \
        -d "{\"email\":\"${agency_email}\",\"password\":\"TestPass1!\",\"first_name\":\"Sarah\",\"last_name\":\"Connor\",\"display_name\":\"Acme Corp\",\"role\":\"agency\"}")
    OWNER_TOKEN=$(echo "$agency_resp" | jq -r '.access_token')
    ORG_ID=$(echo "$agency_resp" | jq -r '.organization.id')
    info "Owner token acquired, org_id=${ORG_ID}"

    local provider_resp
    provider_resp=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" -H "X-Auth-Mode: token" \
        -d "{\"email\":\"${provider_email}\",\"password\":\"TestPass1!\",\"first_name\":\"Marie\",\"last_name\":\"D\",\"role\":\"provider\"}")
    PROVIDER_TOKEN=$(echo "$provider_resp" | jq -r '.access_token')
    info "Provider token acquired"
}

test_send_invitation() {
    section "TEST 1 — Owner sends an invitation (201)"
    local invitee_email="invitee1-${TS}@phase2.test"

    local resp
    resp=$(curl -s -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/invitations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d "{\"email\":\"${invitee_email}\",\"first_name\":\"Paul\",\"last_name\":\"Dupont\",\"title\":\"Office Manager\",\"role\":\"member\"}")

    INVITATION_ID=$(echo "$resp" | jq -r '.id')
    assert_not_empty "invitation.id"    "$INVITATION_ID"
    assert_eq       "invitation.email"  "$invitee_email" "$(echo "$resp" | jq -r '.email')"
    assert_eq       "invitation.role"   "member" "$(echo "$resp" | jq -r '.role')"
    assert_eq       "invitation.status" "pending" "$(echo "$resp" | jq -r '.status')"
    assert_eq       "invitation.organization_id" "$ORG_ID" "$(echo "$resp" | jq -r '.organization_id')"
    INVITEE_EMAIL="$invitee_email"
}

test_list_pending() {
    section "TEST 2 — List pending invitations returns the new row"
    local resp
    resp=$(curl -s -X GET "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/invitations" \
        -H "Authorization: Bearer ${OWNER_TOKEN}")
    local count
    count=$(echo "$resp" | jq '.data | length')
    if [[ $count -ge 1 ]]; then
        printf "  ${GREEN}✓${NC} data length >= 1                                       = ${CYAN}%d${NC}\n" "$count"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} data length >= 1                                       = ${RED}%d${NC}\n" "$count"
        FAIL=$((FAIL+1))
        FAILURES+=("data length was $count, expected >= 1")
    fi
    assert_eq "first.status" "pending" "$(echo "$resp" | jq -r '.data[0].status')"
}

test_validate_token() {
    section "TEST 3 — Public validate token returns preview"
    TOKEN=$(PGPASSWORD=postgres psql -h localhost -p 5435 -U postgres -d ${ISOLATED_DB} -tA -c "SELECT token FROM organization_invitations WHERE id='${INVITATION_ID}'")
    assert_not_empty "token from DB" "$TOKEN"

    local resp
    resp=$(curl -s -X GET "${BACKEND_URL}/api/v1/invitations/validate?token=${TOKEN}")
    assert_eq "preview.email"            "$INVITEE_EMAIL" "$(echo "$resp" | jq -r '.email')"
    assert_eq "preview.role"             "member" "$(echo "$resp" | jq -r '.role')"
    assert_eq "preview.first_name"       "Paul" "$(echo "$resp" | jq -r '.first_name')"
    assert_eq "preview.organization_id"  "$ORG_ID" "$(echo "$resp" | jq -r '.organization_id')"
    assert_eq "preview.organization_type" "agency" "$(echo "$resp" | jq -r '.organization_type')"
}

test_accept_invitation() {
    section "TEST 4 — Public accept creates operator + returns AuthResponse"
    local resp
    resp=$(curl -s -X POST "${BACKEND_URL}/api/v1/invitations/accept" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{\"token\":\"${TOKEN}\",\"password\":\"OperatorPass1!\"}")

    OPERATOR_TOKEN=$(echo "$resp" | jq -r '.access_token')
    assert_not_empty "operator.access_token" "$OPERATOR_TOKEN"
    assert_eq       "operator.user.email"        "$INVITEE_EMAIL" "$(echo "$resp" | jq -r '.user.email')"
    assert_eq       "operator.user.account_type" "operator" "$(echo "$resp" | jq -r '.user.account_type')"
    assert_eq       "operator.user.role"         "agency"   "$(echo "$resp" | jq -r '.user.role')"
    assert_eq       "organization.id"            "$ORG_ID"  "$(echo "$resp" | jq -r '.organization.id')"
    assert_eq       "organization.member_role"   "member"   "$(echo "$resp" | jq -r '.organization.member_role')"

    local me
    me=$(curl -sf -X GET "${BACKEND_URL}/api/v1/auth/me" -H "Authorization: Bearer ${OPERATOR_TOKEN}")
    assert_eq "me.user.account_type"      "operator" "$(echo "$me" | jq -r '.user.account_type')"
    assert_eq "me.organization.member_role" "member"   "$(echo "$me" | jq -r '.organization.member_role')"
}

test_double_accept() {
    section "TEST 5 — Accepting the same invitation twice → 409"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/invitations/accept" \
        -H "Content-Type: application/json" \
        -d "{\"token\":\"${TOKEN}\",\"password\":\"OperatorPass1!\"}")
    assert_eq "HTTP status" "409" "$status"
}

test_duplicate_pending() {
    section "TEST 6 — Sending a second pending invitation for the same email → 409"
    local dup_email="dup-p2-${TS}@phase2.test"
    local first_status
    first_status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/invitations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d "{\"email\":\"${dup_email}\",\"first_name\":\"Dup\",\"last_name\":\"One\",\"role\":\"member\"}")
    assert_eq "first invite HTTP" "201" "$first_status"

    local second_status
    second_status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/invitations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d "{\"email\":\"${dup_email}\",\"first_name\":\"Dup\",\"last_name\":\"Two\",\"role\":\"viewer\"}")
    assert_eq "duplicate invite HTTP" "409" "$second_status"
}

test_provider_cannot_invite() {
    section "TEST 7 — Provider user cannot send invitations (403)"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/invitations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${PROVIDER_TOKEN}" \
        -d "{\"email\":\"noaccess-${TS}@phase2.test\",\"first_name\":\"N\",\"last_name\":\"A\",\"role\":\"member\"}")
    assert_eq "HTTP status" "403" "$status"
}

test_cancel_pending() {
    section "TEST 8 — Cancel pending invitation → 204"
    local cancel_email="cancel-p2-${TS}@phase2.test"
    local create_resp
    create_resp=$(curl -sf -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/invitations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d "{\"email\":\"${cancel_email}\",\"first_name\":\"Cancel\",\"last_name\":\"Me\",\"role\":\"viewer\"}")
    local cancel_id
    cancel_id=$(echo "$create_resp" | jq -r '.id')
    assert_not_empty "created invitation id" "$cancel_id"

    local cancel_status
    cancel_status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/invitations/${cancel_id}" \
        -H "Authorization: Bearer ${OWNER_TOKEN}")
    assert_eq "HTTP status" "204" "$cancel_status"

    local db_status
    db_status=$(PGPASSWORD=postgres psql -h localhost -p 5435 -U postgres -d ${ISOLATED_DB} -tA -c "SELECT status FROM organization_invitations WHERE id='${cancel_id}'")
    assert_eq "DB status" "cancelled" "$db_status"
}

test_invite_as_owner_rejected() {
    section "TEST 9 — Inviting with role=owner is rejected (400)"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/invitations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d "{\"email\":\"ownerrole-${TS}@phase2.test\",\"first_name\":\"X\",\"last_name\":\"Y\",\"role\":\"owner\"}")
    assert_eq "HTTP status" "400" "$status"
}

test_weak_password_on_accept() {
    section "TEST 10 — Weak password on accept is rejected (400)"
    local weak_email="weak-p2-${TS}@phase2.test"
    local create_resp
    create_resp=$(curl -sf -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/invitations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d "{\"email\":\"${weak_email}\",\"first_name\":\"Weak\",\"last_name\":\"Pw\",\"role\":\"member\"}")
    local weak_inv_id
    weak_inv_id=$(echo "$create_resp" | jq -r '.id')
    local weak_token
    weak_token=$(PGPASSWORD=postgres psql -h localhost -p 5435 -U postgres -d ${ISOLATED_DB} -tA -c "SELECT token FROM organization_invitations WHERE id='${weak_inv_id}'")

    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/invitations/accept" \
        -H "Content-Type: application/json" \
        -d "{\"token\":\"${weak_token}\",\"password\":\"weak\"}")
    assert_eq "HTTP status" "400" "$status"
}

printf "${BOLD}\n"
printf "╔════════════════════════════════════════════════════════════════╗\n"
printf "║          PHASE 2 E2E — Team invitations backend contract      ║\n"
printf "╚════════════════════════════════════════════════════════════════╝\n"
printf "${NC}\n"

if ! start_backend; then
    exit 1
fi

setup_fixtures
test_send_invitation
test_list_pending
test_validate_token
test_accept_invitation
test_double_accept
test_duplicate_pending
test_provider_cannot_invite
test_cancel_pending
test_invite_as_owner_rejected
test_weak_password_on_accept

section "SUMMARY"
if [[ $FAIL -eq 0 ]]; then
    printf "  ${GREEN}${BOLD}✓ ALL %d ASSERTIONS PASSED${NC}\n" "$PASS"
else
    printf "  ${GREEN}%d passed${NC}, ${RED}${BOLD}%d failed${NC}\n" "$PASS" "$FAIL"
    echo ""
    for f in "${FAILURES[@]}"; do printf "  ${RED}- %s${NC}\n" "$f"; done
fi
echo ""

if [[ $FAIL -gt 0 ]]; then exit 1; fi
exit 0
