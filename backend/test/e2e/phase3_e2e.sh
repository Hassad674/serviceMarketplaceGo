#!/bin/bash
#
# Phase 3 backend E2E — validates the team management flow:
# list members, promote/demote, remove, leave, transfer ownership,
# plus the session_version revocation that makes role changes
# take effect immediately.

set -uo pipefail

BACKEND_URL="${BACKEND_URL:-http://localhost:8084}"
BACKEND_PORT=8084
ISOLATED_DB="marketplace_go_team"
PG_PORT=5435
TS=$(date +%s)
BACKEND_LOG="/tmp/phase3-e2e-backend.log"

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

    cd /home/hassad/Documents/marketplaceServiceGo-team/backend
    DATABASE_URL="postgres://postgres:postgres@localhost:${PG_PORT}/${ISOLATED_DB}?sslmode=disable" \
    PORT=${BACKEND_PORT} \
    JWT_SECRET="dev-marketplace-secret-key-change-in-production-2024" \
    REDIS_URL="redis://localhost:6380" \
    STORAGE_ENDPOINT="192.168.1.156:9000" \
    STORAGE_ACCESS_KEY="minioadmin" STORAGE_SECRET_KEY="minioadmin" \
    STORAGE_BUCKET="marketplace" STORAGE_USE_SSL="false" \
    STORAGE_PUBLIC_URL="http://192.168.1.156:9000/marketplace" \
    SESSION_TTL="336h" ALLOWED_ORIGINS="http://localhost:3001" \
    RESEND_API_KEY="${RESEND_API_KEY:-dev-dummy-key}" \
    RESEND_DEV_REDIRECT_TO="hassad.smara69@gmail.com" \
    go run cmd/api/main.go > "${BACKEND_LOG}" 2>&1 &
    BACKEND_PID=$!
    info "Backend PID: ${BACKEND_PID}"

    for i in {1..30}; do
        if curl -sf "${BACKEND_URL}/health" > /dev/null 2>&1; then
            printf "  ${GREEN}✓ Backend healthy${NC}\n"
            return 0
        fi
        sleep 1
    done
    printf "  ${RED}✗ Backend did not become healthy${NC}\n"
    tail -40 "${BACKEND_LOG}"
    return 1
}

stop_backend() {
    if [[ -n "$BACKEND_PID" ]]; then
        section "TEARDOWN — Stopping backend"
        kill "$BACKEND_PID" 2>/dev/null || true
        fuser -k "${BACKEND_PORT}/tcp" 2>/dev/null || true
        wait "$BACKEND_PID" 2>/dev/null || true
    fi
}

trap stop_backend EXIT

accept_invitation() {
    local inv_id="$1" password="$2"
    local token
    token=$(PGPASSWORD=postgres psql -h localhost -p ${PG_PORT} -U postgres -d ${ISOLATED_DB} -tA -c "SELECT token FROM organization_invitations WHERE id='${inv_id}'")
    curl -sf -X POST "${BACKEND_URL}/api/v1/invitations/accept" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{\"token\":\"${token}\",\"password\":\"${password}\"}"
}

send_invitation() {
    local owner_token="$1" org_id="$2" email="$3" first_name="$4" role="$5"
    curl -sf -X POST "${BACKEND_URL}/api/v1/organizations/${org_id}/invitations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${owner_token}" \
        -d "{\"email\":\"${email}\",\"first_name\":\"${first_name}\",\"last_name\":\"Op\",\"role\":\"${role}\"}" \
        | jq -r '.id'
}

setup_fixtures() {
    section "FIXTURES — Agency + 2 operator members (Admin + Member)"

    local agency_email="owner-p3-${TS}@phase3.test"
    local reg
    reg=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" -H "X-Auth-Mode: token" \
        -d "{\"email\":\"${agency_email}\",\"password\":\"TestPass1!\",\"first_name\":\"Sarah\",\"last_name\":\"Connor\",\"display_name\":\"Acme P3\",\"role\":\"agency\"}")
    OWNER_TOKEN=$(echo "$reg" | jq -r '.access_token')
    OWNER_USER_ID=$(echo "$reg" | jq -r '.user.id')
    ORG_ID=$(echo "$reg" | jq -r '.organization.id')
    info "Owner: ${OWNER_USER_ID}"
    info "Org:   ${ORG_ID}"

    local admin_email="admin-p3-${TS}@phase3.test"
    local admin_inv_id
    admin_inv_id=$(send_invitation "$OWNER_TOKEN" "$ORG_ID" "$admin_email" "Alice" "admin")
    local admin_resp
    admin_resp=$(accept_invitation "$admin_inv_id" "AdminPass1!")
    ADMIN_TOKEN=$(echo "$admin_resp" | jq -r '.access_token')
    ADMIN_USER_ID=$(echo "$admin_resp" | jq -r '.user.id')
    info "Admin: ${ADMIN_USER_ID}"

    local member_email="member-p3-${TS}@phase3.test"
    local member_inv_id
    member_inv_id=$(send_invitation "$OWNER_TOKEN" "$ORG_ID" "$member_email" "Bob" "member")
    local member_resp
    member_resp=$(accept_invitation "$member_inv_id" "MemberPass1!")
    MEMBER_TOKEN=$(echo "$member_resp" | jq -r '.access_token')
    MEMBER_USER_ID=$(echo "$member_resp" | jq -r '.user.id')
    info "Member: ${MEMBER_USER_ID}"
    MEMBER_EMAIL="$member_email"
}

test_list_members() {
    section "TEST 1 — List members returns Owner + Admin + Member"
    local resp
    resp=$(curl -sf -X GET "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/members" \
        -H "Authorization: Bearer ${OWNER_TOKEN}")
    local count
    count=$(echo "$resp" | jq '.data | length')
    assert_eq "members count" "3" "$count"

    local has_owner
    has_owner=$(echo "$resp" | jq -r --arg id "$OWNER_USER_ID" '.data | map(select(.user_id == $id)) | .[0].role')
    assert_eq "owner role" "owner" "$has_owner"
}

test_promote_member_to_admin() {
    section "TEST 2 — Owner promotes Bob (Member) → Admin"
    local resp
    resp=$(curl -sf -X PATCH "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/members/${MEMBER_USER_ID}" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d '{"role":"admin"}')
    assert_eq "new role" "admin" "$(echo "$resp" | jq -r '.role')"
}

test_demote_admin_to_member() {
    section "TEST 3 — Owner demotes Alice (Admin) → Member"
    local resp
    resp=$(curl -sf -X PATCH "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/members/${ADMIN_USER_ID}" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d '{"role":"member"}')
    assert_eq "new role" "member" "$(echo "$resp" | jq -r '.role')"
}

test_session_revocation_after_demote() {
    section "TEST 4 — Alice's old Admin token is revoked immediately"
    info "ADMIN_TOKEN was issued when Alice was Admin; after demotion, old token should be rejected"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "${BACKEND_URL}/api/v1/auth/me" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}")
    assert_eq "HTTP status (expected 401)" "401" "$status"
}

test_cannot_promote_to_owner() {
    section "TEST 5 — Cannot directly promote to Owner (must use transfer)"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/members/${MEMBER_USER_ID}" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d '{"role":"owner"}')
    assert_eq "HTTP status (expected 400)" "400" "$status"
}

test_initiate_transfer() {
    section "TEST 6 — Owner initiates transfer to Bob (Admin)"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/transfer" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d "{\"target_user_id\":\"${MEMBER_USER_ID}\"}")
    assert_eq "HTTP status" "202" "$status"

    # Bob's MEMBER_TOKEN was invalidated by his earlier promotion —
    # refresh it before he tries to accept the transfer.
    info "Refreshing Bob's token (old token was invalidated by session bump on promote)"
    local login
    login=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{\"email\":\"${MEMBER_EMAIL}\",\"password\":\"MemberPass1!\"}")
    MEMBER_TOKEN=$(echo "$login" | jq -r '.access_token')
}

test_cannot_double_initiate_transfer() {
    section "TEST 7 — Cannot initiate a second transfer while one is pending"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/transfer" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d "{\"target_user_id\":\"${MEMBER_USER_ID}\"}")
    assert_eq "HTTP status (expected 409)" "409" "$status"
}

test_owner_cannot_accept_own_transfer() {
    section "TEST 8 — Current Owner cannot accept their own transfer"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/transfer/accept" \
        -H "Authorization: Bearer ${OWNER_TOKEN}")
    assert_eq "HTTP status (expected 403)" "403" "$status"
}

test_accept_transfer() {
    section "TEST 9 — Bob accepts the transfer → becomes Owner"
    local resp
    resp=$(curl -sf -X POST "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/transfer/accept" \
        -H "Authorization: Bearer ${MEMBER_TOKEN}")
    assert_eq "new owner_user_id" "$MEMBER_USER_ID" "$(echo "$resp" | jq -r '.current_owner_user_id')"
    assert_eq "pending cleared" "null" "$(echo "$resp" | jq -r '.pending_transfer_to_user_id')"
}

test_session_revocation_after_transfer() {
    section "TEST 10 — Old Owner's token revoked after transfer"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "${BACKEND_URL}/api/v1/auth/me" \
        -H "Authorization: Bearer ${OWNER_TOKEN}")
    assert_eq "HTTP status (expected 401)" "401" "$status"
}

test_verify_roles_after_transfer() {
    section "TEST 11 — After transfer: Bob=Owner, Sarah=Admin"
    local login
    login=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{\"email\":\"${MEMBER_EMAIL}\",\"password\":\"MemberPass1!\"}")
    local new_bob_token
    new_bob_token=$(echo "$login" | jq -r '.access_token')

    local resp
    resp=$(curl -sf -X GET "${BACKEND_URL}/api/v1/organizations/${ORG_ID}/members" \
        -H "Authorization: Bearer ${new_bob_token}")

    local bob_role
    bob_role=$(echo "$resp" | jq -r --arg id "$MEMBER_USER_ID" '.data | map(select(.user_id == $id)) | .[0].role')
    assert_eq "Bob (new owner) role" "owner" "$bob_role"

    local sarah_role
    sarah_role=$(echo "$resp" | jq -r --arg id "$OWNER_USER_ID" '.data | map(select(.user_id == $id)) | .[0].role')
    assert_eq "Sarah (old owner) role" "admin" "$sarah_role"
}

printf "${BOLD}\n"
printf "╔════════════════════════════════════════════════════════════════╗\n"
printf "║          PHASE 3 E2E — Team management backend contract      ║\n"
printf "╚════════════════════════════════════════════════════════════════╝\n"
printf "${NC}\n"

if ! start_backend; then exit 1; fi

setup_fixtures
test_list_members
test_promote_member_to_admin
test_demote_admin_to_member
test_session_revocation_after_demote
test_cannot_promote_to_owner
test_initiate_transfer
test_cannot_double_initiate_transfer
test_owner_cannot_accept_own_transfer
test_accept_transfer
test_session_revocation_after_transfer
test_verify_roles_after_transfer

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
