#!/usr/bin/env bash
#
# Lint org-scoping invariant on the backend's postgres adapters.
#
# Invariant: business state (proposals, disputes, reviews, payment_records,
# conversations, jobs, etc.) MUST be queried by `organization_id`, not by
# `user_id`. The `user_id` columns that survived the org migration are
# write-only authorship references (created_by / updated_by) and a few
# tables that legitimately key by user (notifications, organization_members,
# audit_logs, conversation_read_state).
#
# This script grep-scans the postgres adapter files and flags any
# `WHERE user_id = $N` query line that is NOT in the whitelist below.
#
# Whitelist tables (legitimate per-user reads):
#   - notifications, notification_preferences, device_tokens (recipient user)
#   - organization_members, organization_invitations (membership)
#   - audit_logs (action authorship)
#   - conversation_read_state (per-user read marker)
#   - conversation_participants (per-user membership)
#   - users (root table; queried by id is fine)
#
# To exempt a single line, append the marker comment
#   // authorship-by-user-ok
# at the end of the SQL string or the surrounding Go statement.
#
# Exit non-zero on any unflagged violation.

set -euo pipefail

ROOT="${GITHUB_WORKSPACE:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
ADAPTER_DIR="$ROOT/backend/internal/adapter/postgres"

if [ ! -d "$ADAPTER_DIR" ]; then
  echo "::error::postgres adapter directory not found at $ADAPTER_DIR"
  exit 1
fi

# Files allowed to query business state by user_id without being
# flagged. These cover the legitimate per-user surfaces.
ALLOWED_FILES=(
  "audit_repository.go"
  "notification_queries.go"
  "organization_member_repository.go"
  "organization_invitation_repository.go"
  "conversation_read_state_repository.go"
  "user_repository.go"
  "user_queries.go"
  "user_export_repository.go"
  "device_token_repository.go"
  "moderation_results_repository.go"
)

# Tables whose user_id WHERE clauses are always legitimate. Match
# patterns are looked up against the line content (not just the file
# name), so any file may legitimately query these tables.
SAFE_TABLES=(
  "notifications"
  "notification_preferences"
  "device_tokens"
  "organization_members"
  "organization_invitations"
  "audit_logs"
  "conversation_read_state"
  "conversation_participants"
  "users WHERE id"          # root table queried by id only
)

violations=()

while IFS= read -r -d '' file; do
  base=$(basename "$file")

  # Skip _test.go files — only production adapters matter.
  case "$base" in
    *_test.go) continue ;;
  esac

  # Skip allowed files
  skip=0
  for allowed in "${ALLOWED_FILES[@]}"; do
    if [ "$base" = "$allowed" ]; then
      skip=1
      break
    fi
  done
  if [ "$skip" = "1" ]; then
    continue
  fi

  # Find every line containing `WHERE user_id = $`. Look one line up to
  # detect FROM clauses targeting safe tables (multiline SQL).
  while IFS= read -r match; do
    [ -z "$match" ] && continue
    lineno=$(echo "$match" | cut -d: -f1)
    content=$(echo "$match" | cut -d: -f2-)

    # Skip lines explicitly marked as authorship.
    if echo "$content" | grep -q "authorship-by-user-ok"; then
      continue
    fi

    # Look back up to 5 lines for a FROM clause targeting a safe table.
    start=$((lineno > 5 ? lineno - 5 : 1))
    block=$(sed -n "${start},${lineno}p" "$file")

    safe=0
    for tbl in "${SAFE_TABLES[@]}"; do
      if echo "$block" | grep -qE "(FROM|UPDATE|INSERT INTO|DELETE FROM)[[:space:]]+${tbl}"; then
        safe=1
        break
      fi
    done
    if [ "$safe" = "1" ]; then
      continue
    fi

    violations+=("$file:$lineno: $content")
  done < <(grep -n "WHERE user_id = " "$file" 2>/dev/null || true)
done < <(find "$ADAPTER_DIR" -type f -name "*.go" -print0)

if [ "${#violations[@]}" -gt 0 ]; then
  echo "::error::Org-scoping invariant violation(s) found:"
  echo
  for v in "${violations[@]}"; do
    echo "  $v"
  done
  echo
  echo "Business-state queries must use organization_id, not user_id."
  echo "Either:"
  echo "  - Refactor the query to filter by organization_id, OR"
  echo "  - Add the table to SAFE_TABLES if it's a legitimate per-user surface, OR"
  echo "  - Add // authorship-by-user-ok at the end of the line if it's"
  echo "    a write-only authorship column (created_by / updated_by)."
  echo
  echo "See project_org_based_model.md for the full invariant."
  exit 1
fi

echo "org-scoping lint: OK (no unflagged user_id business-state queries)"
exit 0
