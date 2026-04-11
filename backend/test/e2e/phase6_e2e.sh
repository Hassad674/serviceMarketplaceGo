#!/bin/bash
#
# Phase 6 backend E2E — validates the admin team endpoints.
# Exercises the full force-action API surface against an isolated
# DB: get-user-organization, force-update-role, force-cancel-invite,
# force-remove-member, force-transfer-ownership. Also spot-checks
# that AdminUserResponse now carries account_type + organization_id
# and that the dashboard stats surface total_organizations +
# pending_invitations.
#
# The admin user is seeded directly in SQL so we do not depend on
# a `/admin/register` endpoint that does not exist.

set -uo pipefail

BACKEND_URL="${BACKEND_URL:-http://localhost:8089}"
BACKEND_PORT=8089
ISOLATED_DB="marketplace_go_team"
PG_PORT=5435
TS=$(date +%s)
BACKEND_LOG="/tmp/phase6-e2e-backend.log"

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
        printf "  ${GREEN}✓${NC} %-60s = ${CYAN}%s${NC}\n" "$name" "$actual"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} %-60s expected=${YELLOW}%s${NC} actual=${RED}%s${NC}\n" "$name" "$expected" "$actual"
        FAIL=$((FAIL+1))
        FAILURES+=("$name: expected '$expected', got '$actual'")
    fi
}

assert_not_empty() {
    local name="$1" value="$2"
    if [[ -n "$value" && "$value" != "null" ]]; then
        printf "  ${GREEN}✓${NC} %-60s = ${CYAN}%s${NC}\n" "$name" "$value"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} %-60s was empty/null\n" "$name"
        FAIL=$((FAIL+1))
        FAILURES+=("$name was empty")
    fi
}

section() {
    echo ""
    printf "${BLUE}${BOLD}════════════════════════════════════════════════════════════════${NC}\n"
    printf "${BLUE}${BOLD}▶ %s${NC}\n" "$1"
    printf "${BLUE}${BOLD}════════════════════════════════════════════════════════════════${NC}\n"
}

info() { printf "  ${CYAN}ℹ${NC} %s\n" "$1"; }

PG() {
    PGPASSWORD=postgres psql -h localhost -p ${PG_PORT} -U postgres -d ${ISOLATED_DB} -tA -c "$1"
}

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
    tail -60 "${BACKEND_LOG}"
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

register() {
    local email="$1" role="$2" first="$3"
    curl -sf -X POST "${BACKEND_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" -H "X-Auth-Mode: token" \
        -d "{\"email\":\"${email}\",\"password\":\"TestPass1!\",\"first_name\":\"${first}\",\"last_name\":\"P6\",\"display_name\":\"${first} P6\",\"role\":\"${role}\"}"
}

send_invitation() {
    local owner_token="$1" org_id="$2" email="$3" first="$4" role="$5"
    curl -sf -X POST "${BACKEND_URL}/api/v1/organizations/${org_id}/invitations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${owner_token}" \
        -d "{\"email\":\"${email}\",\"first_name\":\"${first}\",\"last_name\":\"Op\",\"role\":\"${role}\"}" \
        | jq -r '.id'
}

accept_invitation() {
    local inv_id="$1" password="$2"
    local token
    token=$(PG "SELECT token FROM organization_invitations WHERE id='${inv_id}'")
    curl -sf -X POST "${BACKEND_URL}/api/v1/invitations/accept" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Mode: token" \
        -d "{\"token\":\"${token}\",\"password\":\"${password}\"}"
}

seed_admin_and_login() {
    section "FIXTURES — Seed admin user + agency + operator"

    # 1. Register an ordinary user then promote them to admin via SQL.
    local admin_email="admin-p6-${TS}@phase6.test"
    local reg
    reg=$(register "$admin_email" "agency" "Admin")
    ADMIN_USER_ID=$(echo "$reg" | jq -r '.user.id')
    info "Admin user: ${ADMIN_USER_ID}"

    PG "UPDATE users SET is_admin = true WHERE id='${ADMIN_USER_ID}'" > /dev/null

    # Re-login to get a token that carries is_admin=true in the JWT.
    local login
    login=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" -H "X-Auth-Mode: token" \
        -d "{\"email\":\"${admin_email}\",\"password\":\"TestPass1!\"}")
    ADMIN_TOKEN=$(echo "$login" | jq -r '.access_token')
    assert_not_empty "admin access token" "$ADMIN_TOKEN"

    # 2. Register the agency Owner (target of the team management).
    local owner_email="owner-p6-${TS}@phase6.test"
    reg=$(register "$owner_email" "agency" "Sarah")
    OWNER_TOKEN=$(echo "$reg" | jq -r '.access_token')
    OWNER_USER_ID=$(echo "$reg" | jq -r '.user.id')
    ORG_ID=$(echo "$reg" | jq -r '.organization.id')
    info "Owner user: ${OWNER_USER_ID}"
    info "Org:        ${ORG_ID}"

    # 3. Invite + accept two operators so we have members to target.
    local inv1
    inv1=$(send_invitation "$OWNER_TOKEN" "$ORG_ID" "alice-p6-${TS}@phase6.test" "Alice" "admin")
    local alice
    alice=$(accept_invitation "$inv1" "AlicePass1!")
    ALICE_USER_ID=$(echo "$alice" | jq -r '.user.id')
    info "Alice (admin operator): ${ALICE_USER_ID}"

    local inv2
    inv2=$(send_invitation "$OWNER_TOKEN" "$ORG_ID" "bob-p6-${TS}@phase6.test" "Bob" "member")
    local bob
    bob=$(accept_invitation "$inv2" "BobPass1!")
    BOB_USER_ID=$(echo "$bob" | jq -r '.user.id')
    info "Bob (member operator):  ${BOB_USER_ID}"

    # 4. Seed one pending invitation to test force-cancel.
    PENDING_INV_ID=$(send_invitation "$OWNER_TOKEN" "$ORG_ID" "charlie-p6-${TS}@phase6.test" "Charlie" "viewer")
    info "Pending invitation: ${PENDING_INV_ID}"
}

test_admin_user_dto_includes_team_fields() {
    section "TEST 1 — AdminUserResponse exposes account_type + organization_id"

    local resp
    resp=$(curl -sf -X GET "${BACKEND_URL}/api/v1/admin/users/${OWNER_USER_ID}" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}")

    assert_eq "owner account_type" "marketplace_owner" "$(echo "$resp" | jq -r '.data.account_type')"
    assert_eq "owner organization_id matches" "$ORG_ID" "$(echo "$resp" | jq -r '.data.organization_id')"

    local alice_resp
    alice_resp=$(curl -sf -X GET "${BACKEND_URL}/api/v1/admin/users/${ALICE_USER_ID}" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}")
    assert_eq "alice account_type" "operator" "$(echo "$alice_resp" | jq -r '.data.account_type')"
    assert_eq "alice organization_id matches" "$ORG_ID" "$(echo "$alice_resp" | jq -r '.data.organization_id')"
}

test_get_user_organization() {
    section "TEST 2 — GET /admin/users/{id}/organization returns full detail"

    local resp
    resp=$(curl -sf -X GET "${BACKEND_URL}/api/v1/admin/users/${OWNER_USER_ID}/organization" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}")

    assert_eq "org id" "$ORG_ID" "$(echo "$resp" | jq -r '.organization.id')"
    assert_eq "viewing_role for owner" "owner" "$(echo "$resp" | jq -r '.viewing_role')"
    assert_eq "member count (owner + alice + bob)" "3" "$(echo "$resp" | jq '.members | length')"
    assert_eq "pending invitation count" "1" "$(echo "$resp" | jq '.pending_invitations | length')"

    # Operator view
    local op_resp
    op_resp=$(curl -sf -X GET "${BACKEND_URL}/api/v1/admin/users/${ALICE_USER_ID}/organization" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}")
    assert_eq "viewing_role for alice" "admin" "$(echo "$op_resp" | jq -r '.viewing_role')"
}

test_force_cancel_invitation() {
    section "TEST 3 — Force cancel pending invitation"

    curl -sf -X DELETE "${BACKEND_URL}/api/v1/admin/organizations/${ORG_ID}/invitations/${PENDING_INV_ID}" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" > /dev/null

    local gone
    gone=$(PG "SELECT COUNT(*) FROM organization_invitations WHERE id='${PENDING_INV_ID}'")
    assert_eq "invitation row deleted" "0" "$gone"
}

test_force_update_member_role() {
    section "TEST 4 — Force update Bob (member) → viewer"

    curl -sf -X PATCH "${BACKEND_URL}/api/v1/admin/organizations/${ORG_ID}/members/${BOB_USER_ID}" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        -d '{"role":"viewer"}' > /dev/null

    local new_role
    new_role=$(PG "SELECT role FROM organization_members WHERE organization_id='${ORG_ID}' AND user_id='${BOB_USER_ID}'")
    assert_eq "bob new role" "viewer" "$new_role"
}

test_force_remove_member() {
    section "TEST 5 — Force remove Alice (admin operator)"

    curl -sf -X DELETE "${BACKEND_URL}/api/v1/admin/organizations/${ORG_ID}/members/${ALICE_USER_ID}" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" > /dev/null

    local member_gone
    member_gone=$(PG "SELECT COUNT(*) FROM organization_members WHERE organization_id='${ORG_ID}' AND user_id='${ALICE_USER_ID}'")
    assert_eq "alice membership deleted" "0" "$member_gone"

    local user_gone
    user_gone=$(PG "SELECT COUNT(*) FROM users WHERE id='${ALICE_USER_ID}'")
    assert_eq "alice operator user purged" "0" "$user_gone"
}

test_force_transfer_ownership_to_non_admin() {
    section "TEST 6 — Force transfer ownership to Bob (who is now a Viewer)"

    # Bob is now a Viewer after TEST 4. Regular transfer would reject
    # this — force override must accept.
    curl -sf -X POST "${BACKEND_URL}/api/v1/admin/organizations/${ORG_ID}/force-transfer" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        -d "{\"target_user_id\":\"${BOB_USER_ID}\"}" > /dev/null

    local owner_col
    owner_col=$(PG "SELECT owner_user_id FROM organizations WHERE id='${ORG_ID}'")
    assert_eq "owner_user_id on org" "$BOB_USER_ID" "$owner_col"

    local bob_role
    bob_role=$(PG "SELECT role FROM organization_members WHERE organization_id='${ORG_ID}' AND user_id='${BOB_USER_ID}'")
    assert_eq "bob role (new owner)" "owner" "$bob_role"

    local sarah_role
    sarah_role=$(PG "SELECT role FROM organization_members WHERE organization_id='${ORG_ID}' AND user_id='${OWNER_USER_ID}'")
    assert_eq "sarah role (demoted)" "admin" "$sarah_role"
}

test_dashboard_stats_team_fields() {
    section "TEST 7 — Dashboard stats expose total_organizations + pending_invitations"

    local resp
    resp=$(curl -sf -X GET "${BACKEND_URL}/api/v1/admin/dashboard/stats" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}")

    local total_orgs
    total_orgs=$(echo "$resp" | jq -r '.total_organizations')
    if [[ "$total_orgs" == "null" || -z "$total_orgs" ]]; then
        assert_eq "total_organizations present" "number" "missing"
    else
        # Any positive integer is fine — we just assert the field is
        # present and numeric.
        printf "  ${GREEN}✓${NC} %-60s = ${CYAN}%s${NC}\n" "total_organizations returned" "$total_orgs"
        PASS=$((PASS+1))
    fi

    local pending
    pending=$(echo "$resp" | jq -r '.pending_invitations')
    if [[ "$pending" == "null" || -z "$pending" ]]; then
        assert_eq "pending_invitations present" "number" "missing"
    else
        printf "  ${GREEN}✓${NC} %-60s = ${CYAN}%s${NC}\n" "pending_invitations returned" "$pending"
        PASS=$((PASS+1))
    fi
}

run_all() {
    start_backend || exit 1
    seed_admin_and_login
    test_admin_user_dto_includes_team_fields
    test_get_user_organization
    test_force_cancel_invitation
    test_force_update_member_role
    test_force_remove_member
    test_force_transfer_ownership_to_non_admin
    test_dashboard_stats_team_fields

    section "SUMMARY"
    printf "  ${GREEN}PASS${NC}: %d\n" "$PASS"
    printf "  ${RED}FAIL${NC}: %d\n" "$FAIL"
    if [[ $FAIL -gt 0 ]]; then
        echo ""
        printf "  ${RED}Failures:${NC}\n"
        for f in "${FAILURES[@]}"; do
            printf "    • %s\n" "$f"
        done
        echo ""
        info "Backend log tail:"
        tail -40 "${BACKEND_LOG}"
        exit 1
    fi
    printf "  ${GREEN}${BOLD}All checks green.${NC}\n"
}

run_all
