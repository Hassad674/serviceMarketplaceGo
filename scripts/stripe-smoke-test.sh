#!/usr/bin/env bash
#
# stripe-smoke-test.sh
#
# End-to-end smoke test for the Stripe Embedded Components notification
# pipeline. Sends signed webhook payloads directly to the backend and
# verifies the correct notifications land in the database.
#
# What it validates (the FULL chain):
#   Backend receives webhook → signature verified → snapshot parsed
#   → Notifier diffs state → notification persisted in DB
#
# Requirements:
#   - Backend running on BACKEND_URL (default: http://localhost:8084)
#   - Postgres reachable via DB_URL (from backend/.env)
#   - A row in test_embedded_accounts linking a user_id to a stripe_account_id
#   - psql, curl, openssl available
#
# Usage:
#   ./scripts/stripe-smoke-test.sh
#   ./scripts/stripe-smoke-test.sh --account acct_XXX --user USER_UUID
#
# Exit codes: 0 = all pass, 1 = one or more scenarios failed

set -uo pipefail

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

BACKEND_URL="${BACKEND_URL:-http://localhost:8084}"
ENV_FILE="$(dirname "$0")/../backend/.env"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "❌ $ENV_FILE not found"
  exit 1
fi

# Load STRIPE_WEBHOOK_SECRET + DATABASE_URL from backend/.env
set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

WEBHOOK_URL="${BACKEND_URL}/api/v1/stripe/webhook"
WEBHOOK_SECRET="${STRIPE_WEBHOOK_SECRET:?STRIPE_WEBHOOK_SECRET must be set}"
DB_URL="${DATABASE_URL:?DATABASE_URL must be set}"

# Override via CLI flags
ACCOUNT_ID=""
USER_ID=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --account) ACCOUNT_ID="$2"; shift 2 ;;
    --user)    USER_ID="$2"; shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

# Auto-detect first account in test_embedded_accounts if not provided
if [[ -z "$ACCOUNT_ID" || -z "$USER_ID" ]]; then
  row=$(psql "$DB_URL" -t -A -F'|' -c "SELECT user_id, stripe_account_id FROM test_embedded_accounts ORDER BY updated_at DESC LIMIT 1;" 2>/dev/null)
  if [[ -z "$row" ]]; then
    echo "❌ No row in test_embedded_accounts — create an account via /fr/payment-info-v2 first"
    exit 1
  fi
  USER_ID="${row%%|*}"
  ACCOUNT_ID="${row##*|}"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Stripe Embedded — Smoke Test Pipeline"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Backend:    $BACKEND_URL"
echo "  Account:    $ACCOUNT_ID"
echo "  User:       $USER_ID"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo

# Restart backend to flush in-memory Notifier cooldown. This is needed
# because the 5-min cooldown is a sync.Map inside the process — scenarios
# would otherwise silently drop duplicates from previous runs.
echo "Restarting backend to flush cooldown..."
pkill -f mp-embedded 2>/dev/null
sleep 1
(cd "$(dirname "$0")/../backend" \
  && go build -o /tmp/mp-embedded ./cmd/api 2>&1 >/dev/null \
  && set -a && source .env && set +a \
  && nohup /tmp/mp-embedded >/tmp/embedded-backend.log 2>&1 &)
# Wait for backend to be ready
for i in {1..15}; do
  if curl -sf "${BACKEND_URL}/health" >/dev/null 2>&1; then
    echo "  Backend ready."
    break
  fi
  sleep 1
done
echo

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# Reset the notifier state for this account so each scenario starts fresh.
# Also clears old notifications + in-memory cooldown by restarting backend
# would be ideal, but we avoid that. Instead we use scenarios with unique
# notification keys to avoid cooldown collisions.
reset_state() {
  psql "$DB_URL" -q -c "UPDATE test_embedded_accounts SET last_state = NULL WHERE stripe_account_id = '$ACCOUNT_ID';" >/dev/null 2>&1
}

# Purge old notifications so count_recent_notifs gives clean deltas.
purge_recent_notifs() {
  psql "$DB_URL" -q -c "DELETE FROM notifications WHERE user_id = '$USER_ID' AND created_at > now() - interval '2 minutes' AND type IN ('stripe_requirements','stripe_account_status');" >/dev/null 2>&1
}

# Count notifications for this user matching a pattern in title OR body.
count_notifs_matching() {
  local pattern="$1"
  psql "$DB_URL" -t -A -c "SELECT count(*) FROM notifications WHERE user_id = '$USER_ID' AND (title ILIKE '%${pattern}%' OR body ILIKE '%${pattern}%') AND created_at > now() - interval '30 seconds';" 2>/dev/null | tr -d ' '
}

# Count ALL recent notifications for this user (used to compute delta).
count_recent_notifs() {
  psql "$DB_URL" -t -A -c "SELECT count(*) FROM notifications WHERE user_id = '$USER_ID' AND created_at > now() - interval '30 seconds';" 2>/dev/null | tr -d ' '
}

# Sign a payload using Stripe's webhook signature scheme and POST it.
# Args: $1 = JSON payload
send_webhook() {
  local payload="$1"
  local ts
  ts=$(date +%s)
  local sig
  sig=$(printf '%s.%s' "$ts" "$payload" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -hex | awk '{print $NF}')
  local header="t=${ts},v1=${sig}"

  local resp
  resp=$(curl -sS -o /dev/null -w "%{http_code}" -X POST "$WEBHOOK_URL" \
    -H "Stripe-Signature: $header" \
    -H "Content-Type: application/json" \
    -d "$payload" 2>&1)
  echo "$resp"
}

# Run one scenario: send webhook, wait 500ms, check expected pattern exists.
# Args: $1 = scenario label, $2 = JSON payload, $3 = expected title/body pattern
run_scenario() {
  local label="$1"
  local payload="$2"
  local expected="$3"

  reset_state

  local before
  before=$(count_recent_notifs)

  local status
  status=$(send_webhook "$payload")
  if [[ "$status" != "200" ]]; then
    printf "  ❌ %-50s HTTP %s\n" "$label" "$status"
    return 1
  fi

  # Give the notifier + DB insert a moment
  sleep 0.5

  local after
  after=$(count_recent_notifs)

  local delta=$((after - before))
  if [[ "$delta" -eq 0 ]]; then
    printf "  ❌ %-50s no notif created\n" "$label"
    return 1
  fi

  local matches
  matches=$(count_notifs_matching "$expected")
  if [[ "$matches" -eq 0 ]]; then
    printf "  ❌ %-50s notif found but no match for '%s'\n" "$label" "$expected"
    return 1
  fi

  printf "  ✅ %-50s %s notif(s), %s match\n" "$label" "$delta" "$matches"
  return 0
}

# ---------------------------------------------------------------------------
# Event payload builder
# ---------------------------------------------------------------------------

# Build an account.updated event payload.
build_event() {
  local event_type="${1:-account.updated}"
  local charges="${2:-true}"
  local payouts="${3:-true}"
  local details_submitted="${4:-true}"
  local currently_due="${5:-[]}"
  local eventually_due="${6:-[]}"
  local past_due="${7:-[]}"
  local pending="${8:-[]}"
  local errors="${9:-[]}"
  local disabled_reason="${10:-null}"

  cat <<EOF
{
  "id": "evt_smoke_$(date +%s%N)",
  "object": "event",
  "type": "${event_type}",
  "created": $(date +%s),
  "livemode": false,
  "data": {
    "object": {
      "id": "${ACCOUNT_ID}",
      "object": "account",
      "country": "FR",
      "business_type": "individual",
      "charges_enabled": ${charges},
      "payouts_enabled": ${payouts},
      "details_submitted": ${details_submitted},
      "requirements": {
        "currently_due": ${currently_due},
        "eventually_due": ${eventually_due},
        "past_due": ${past_due},
        "pending_verification": ${pending},
        "errors": ${errors},
        "disabled_reason": ${disabled_reason}
      }
    }
  }
}
EOF
}

# ---------------------------------------------------------------------------
# Scenarios
# ---------------------------------------------------------------------------

echo "Running 7 scenarios..."
echo

PASSED=0
FAILED=0

# 1. Account activated (charges + payouts true, 0 requirements)
run_scenario \
  "1. Account activated" \
  "$(build_event account.updated true true true)" \
  "activé" && PASSED=$((PASSED+1)) || FAILED=$((FAILED+1))

# 2. Multiple requirements (currently_due + pluralisation check)
run_scenario \
  "2. Requirements added (plural count)" \
  "$(build_event account.updated true true true '["individual.verification.document","individual.phone","external_account"]')" \
  "3 informations" && PASSED=$((PASSED+1)) || FAILED=$((FAILED+1))

# 3. Eventually due (non-urgent anticipation)
run_scenario \
  "3. Eventually_due (anticipated req)" \
  "$(build_event account.updated true true true '[]' '["individual.verification.additional_document"]')" \
  "à prévoir" && PASSED=$((PASSED+1)) || FAILED=$((FAILED+1))

# 4. Document expired error
run_scenario \
  "4. Document expired" \
  "$(build_event account.updated true true true '[]' '[]' '[]' '[]' '[{"requirement":"individual.verification.document","code":"verification_document_expired","reason":"Document has expired."}]')" \
  "Document expiré" && PASSED=$((PASSED+1)) || FAILED=$((FAILED+1))

# 5. Document blurry error
run_scenario \
  "5. Document illisible/blurry" \
  "$(build_event account.updated true true true '[]' '[]' '[]' '[]' '[{"requirement":"individual.verification.document","code":"verification_document_too_blurry","reason":"Blurry."}]')" \
  "illisible" && PASSED=$((PASSED+1)) || FAILED=$((FAILED+1))

# 6. Past due (urgent)
run_scenario \
  "6. Past_due urgent" \
  "$(build_event account.updated true true true '[]' '[]' '["individual.verification.document"]')" \
  "urgente" && PASSED=$((PASSED+1)) || FAILED=$((FAILED+1))

# 7. Account suspended (charges + payouts disabled)
run_scenario \
  "7. Account suspended (charges disabled)" \
  "$(build_event account.updated false false true '[]' '[]' '[]' '[]' '[]' '"requirements.past_due"')" \
  "suspendu" && PASSED=$((PASSED+1)) || FAILED=$((FAILED+1))

# Reset so next runs start clean
reset_state

# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
TOTAL=$((PASSED + FAILED))
if [[ "$FAILED" -eq 0 ]]; then
  echo "  ✅ $PASSED/$TOTAL scenarios PASSED"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  echo "Still to verify manually (channels the script can't observe):"
  echo "  • Email inbox — open mailbox configured in backend"
  echo "  • Push notification — check device (if FCM configured)"
  echo "  • Visual UI — open /fr/payment-info-v2 in browser"
  echo
  exit 0
else
  echo "  ❌ $FAILED/$TOTAL scenarios FAILED ($PASSED passed)"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  echo "Debug tips:"
  echo "  • Check backend logs:  tail -f /tmp/embedded-backend.log"
  echo "  • Query notifications: psql \"\$DATABASE_URL\" -c \"SELECT title,created_at FROM notifications WHERE user_id = '$USER_ID' ORDER BY created_at DESC LIMIT 10;\""
  echo "  • Verify signature:    echo \$STRIPE_WEBHOOK_SECRET"
  echo
  exit 1
fi
