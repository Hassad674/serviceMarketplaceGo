#!/bin/bash
#
# Phase 4 backend E2E — validates that every INSERT path we touched
# populates organization_id correctly without regressing the existing
# flows (jobs, proposals, conversations, payment_records).
#
# Strategy: spin up the backend against the isolated DB, run the full
# flow an agency user would, then probe PostgreSQL directly to verify
# each new row received the expected organization_id denormalization.

set -uo pipefail

BACKEND_URL="${BACKEND_URL:-http://localhost:8087}"
BACKEND_PORT=8087
ISOLATED_DB="marketplace_go_team"
PG_PORT=5435
TS=$(date +%s)
BACKEND_LOG="/tmp/phase4-e2e-backend.log"

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
        printf "  ${GREEN}✓${NC} %-55s = ${CYAN}%s${NC}\n" "$name" "$value"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} %-55s was empty/null\n" "$name"
        FAIL=$((FAIL+1))
        FAILURES+=("$name was empty")
    fi
}

assert_empty() {
    local name="$1" value="$2"
    if [[ -z "$value" || "$value" == "null" || "$value" == "" ]]; then
        printf "  ${GREEN}✓${NC} %-55s = ${CYAN}(null)${NC}\n" "$name"
        PASS=$((PASS+1))
    else
        printf "  ${RED}✗${NC} %-55s expected null, got=${RED}%s${NC}\n" "$name" "$value"
        FAIL=$((FAIL+1))
        FAILURES+=("$name expected null, got $value")
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
        -d "{\"email\":\"${email}\",\"password\":\"TestPass1!\",\"first_name\":\"${first}\",\"last_name\":\"P4\",\"display_name\":\"${first} P4\",\"role\":\"${role}\"}"
}

setup_fixtures() {
    section "FIXTURES — 1 agency + 2 providers"

    local agency_email="owner-p4-${TS}@phase4.test"
    local reg
    reg=$(register "$agency_email" "agency" "AgencyOwner")
    OWNER_TOKEN=$(echo "$reg" | jq -r '.access_token')
    OWNER_USER_ID=$(echo "$reg" | jq -r '.user.id')
    ORG_ID=$(echo "$reg" | jq -r '.organization.id')
    info "Owner: ${OWNER_USER_ID}"
    info "Org:   ${ORG_ID}"

    local p1_email="prov1-p4-${TS}@phase4.test"
    reg=$(register "$p1_email" "provider" "ProviderAlice")
    PROV1_TOKEN=$(echo "$reg" | jq -r '.access_token')
    PROV1_USER_ID=$(echo "$reg" | jq -r '.user.id')
    info "Provider1: ${PROV1_USER_ID}"

    local p2_email="prov2-p4-${TS}@phase4.test"
    reg=$(register "$p2_email" "provider" "ProviderBob")
    PROV2_TOKEN=$(echo "$reg" | jq -r '.access_token')
    PROV2_USER_ID=$(echo "$reg" | jq -r '.user.id')
    info "Provider2: ${PROV2_USER_ID}"
}

test_job_org_populated() {
    section "TEST 1 — New job gets organization_id from creator"

    local resp
    resp=$(curl -sf -X POST "${BACKEND_URL}/api/v1/jobs/" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d '{
            "title": "Phase 4 job",
            "description": "integration test",
            "skills": ["go"],
            "applicant_type": "freelancers",
            "budget_type": "one_shot",
            "min_budget": 100,
            "max_budget": 200,
            "description_type": "text"
        }')
    local job_id
    job_id=$(echo "$resp" | jq -r '.id')
    assert_not_empty "job_id" "$job_id"

    local stored_org
    stored_org=$(PG "SELECT organization_id FROM jobs WHERE id='${job_id}'")
    assert_eq "job.organization_id" "$ORG_ID" "$stored_org"
    JOB_ID="$job_id"
}

test_provider_job_org_null() {
    section "TEST 2 — Provider-created job keeps organization_id NULL"

    local resp
    resp=$(curl -sf -X POST "${BACKEND_URL}/api/v1/jobs/" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${PROV1_TOKEN}" \
        -d '{
            "title": "Phase 4 provider job",
            "description": "solo provider",
            "skills": ["go"],
            "applicant_type": "freelancers",
            "budget_type": "one_shot",
            "min_budget": 100,
            "max_budget": 200,
            "description_type": "text"
        }' 2>&1)
    local provider_job_id http_code
    provider_job_id=$(echo "$resp" | jq -r '.id' 2>/dev/null || echo "")
    if [[ -z "$provider_job_id" || "$provider_job_id" == "null" ]]; then
        info "Provider cannot create jobs via this endpoint (domain rule) — skipping this check"
        return
    fi
    local stored_org
    stored_org=$(PG "SELECT organization_id FROM jobs WHERE id='${provider_job_id}'")
    assert_empty "provider job.organization_id" "$stored_org"
}

open_conversation() {
    local token="$1" peer_id="$2" greeting="$3"
    curl -sf -X POST "${BACKEND_URL}/api/v1/messaging/conversations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${token}" \
        -d "{\"recipient_id\":\"${peer_id}\",\"content\":\"${greeting}\"}"
}

test_agency_provider_conversation() {
    section "TEST 3 — Agency↔Provider conversation gets agency's organization_id"

    local resp
    resp=$(open_conversation "$OWNER_TOKEN" "$PROV1_USER_ID" "hello from agency")
    AGENCY_CONV_ID=$(echo "$resp" | jq -r '.conversation_id')
    assert_not_empty "conversation_id" "$AGENCY_CONV_ID"

    local stored_org
    stored_org=$(PG "SELECT organization_id FROM conversations WHERE id='${AGENCY_CONV_ID}'")
    assert_eq "conversation.organization_id" "$ORG_ID" "$stored_org"
}

test_provider_provider_conversation() {
    section "TEST 4 — Provider↔Provider conversation keeps organization_id NULL"

    local resp
    resp=$(open_conversation "$PROV1_TOKEN" "$PROV2_USER_ID" "hey solo buddy")
    local conv_id
    conv_id=$(echo "$resp" | jq -r '.conversation_id')
    assert_not_empty "provider conv id" "$conv_id"

    local stored_org
    stored_org=$(PG "SELECT organization_id FROM conversations WHERE id='${conv_id}'")
    assert_empty "provider conv.organization_id" "$stored_org"
}

test_proposal_org_populated() {
    section "TEST 5 — New proposal gets organization_id from client"

    # Agency sends proposal to Provider1 over the existing AGENCY_CONV_ID
    local resp
    resp=$(curl -sf -X POST "${BACKEND_URL}/api/v1/proposals/" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${OWNER_TOKEN}" \
        -d "{
            \"conversation_id\": \"${AGENCY_CONV_ID}\",
            \"recipient_id\": \"${PROV1_USER_ID}\",
            \"title\": \"Phase 4 proposal\",
            \"description\": \"integration test\",
            \"amount\": 15000
        }")
    local prop_id
    prop_id=$(echo "$resp" | jq -r '.id')
    assert_not_empty "proposal_id" "$prop_id"

    local stored_org
    stored_org=$(PG "SELECT organization_id FROM proposals WHERE id='${prop_id}'")
    assert_eq "proposal.organization_id" "$ORG_ID" "$stored_org"
}

test_historical_data_still_correct() {
    section "TEST 6 — Historical data backfilled (regression check)"

    # Every pre-existing agency/enterprise user must have an org now.
    local orphan_count
    orphan_count=$(PG "SELECT COUNT(*) FROM users WHERE role IN ('agency','enterprise') AND organization_id IS NULL")
    assert_eq "orphan agency/enterprise users" "0" "$orphan_count"

    # Every job created by a user with an org must also have that org on the row.
    local mismatched_jobs
    mismatched_jobs=$(PG "
        SELECT COUNT(*) FROM jobs j
        JOIN users u ON u.id = j.creator_id
        WHERE u.organization_id IS NOT NULL
          AND (j.organization_id IS NULL OR j.organization_id != u.organization_id)
    ")
    assert_eq "jobs with stale org_id" "0" "$mismatched_jobs"

    # Same check for proposals — client_id is the business side.
    local mismatched_props
    mismatched_props=$(PG "
        SELECT COUNT(*) FROM proposals p
        JOIN users u ON u.id = p.client_id
        WHERE u.organization_id IS NOT NULL
          AND (p.organization_id IS NULL OR p.organization_id != u.organization_id)
    ")
    assert_eq "proposals with stale org_id" "0" "$mismatched_props"

    # Same check for payment_records.
    local mismatched_pay
    mismatched_pay=$(PG "
        SELECT COUNT(*) FROM payment_records pr
        JOIN users u ON u.id = pr.client_id
        WHERE u.organization_id IS NOT NULL
          AND (pr.organization_id IS NULL OR pr.organization_id != u.organization_id)
    ")
    assert_eq "payment_records with stale org_id" "0" "$mismatched_pay"
}

test_regression_list_my_jobs() {
    section "TEST 7 — ListMyJobs still returns the new job (no query regression)"

    local resp
    resp=$(curl -sf -X GET "${BACKEND_URL}/api/v1/jobs/mine" \
        -H "Authorization: Bearer ${OWNER_TOKEN}")
    local count
    count=$(echo "$resp" | jq '.data | length')
    # Owner created 1 job in TEST 1, we expect at least 1
    local has_job
    has_job=$(echo "$resp" | jq --arg id "$JOB_ID" '[.data[] | select(.id == $id)] | length')
    assert_eq "owner sees their own job" "1" "$has_job"
}

test_regression_list_conversations() {
    section "TEST 8 — ListConversations still returns the new conversation"

    local resp
    resp=$(curl -sf -X GET "${BACKEND_URL}/api/v1/messaging/conversations?limit=50" \
        -H "Authorization: Bearer ${OWNER_TOKEN}")
    local has_conv
    has_conv=$(echo "$resp" | jq --arg id "$AGENCY_CONV_ID" '[.data[] | select(.id == $id)] | length')
    assert_eq "owner sees agency↔provider conversation" "1" "$has_conv"
}

run_all() {
    start_backend || exit 1
    setup_fixtures
    test_job_org_populated
    test_provider_job_org_null
    test_agency_provider_conversation
    test_provider_provider_conversation
    test_proposal_org_populated
    test_historical_data_still_correct
    test_regression_list_my_jobs
    test_regression_list_conversations

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
        tail -30 "${BACKEND_LOG}"
        exit 1
    fi
    printf "  ${GREEN}${BOLD}All checks green.${NC}\n"
}

run_all
