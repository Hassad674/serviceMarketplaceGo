#!/usr/bin/env bash
#
# Test harness for scripts/ci/lint-org-scoping.sh.
#
# Builds a synthetic backend/internal/adapter/postgres/ tree, drops
# known-good and known-bad files into it, runs the lint, and asserts
# the exit code + the violation list match expectations.
#
# Run: bash scripts/ci/__tests__/test-lint-org-scoping.sh
# Exit 0 on all-passes, non-zero on any failure.

set -uo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
LINT_SCRIPT="$ROOT/scripts/ci/lint-org-scoping.sh"

if [ ! -f "$LINT_SCRIPT" ]; then
  echo "FAIL: lint script not found at $LINT_SCRIPT"
  exit 1
fi

passes=0
fails=0

assert_lint_ok() {
  local label="$1"
  local fixture_dir="$2"
  GITHUB_WORKSPACE="$fixture_dir" bash "$LINT_SCRIPT" >/dev/null 2>&1
  local rc=$?
  if [ "$rc" = "0" ]; then
    echo "  PASS: $label (lint exited 0 as expected)"
    passes=$((passes + 1))
  else
    echo "  FAIL: $label (lint exited $rc; expected 0)"
    fails=$((fails + 1))
  fi
}

assert_lint_fail() {
  local label="$1"
  local fixture_dir="$2"
  GITHUB_WORKSPACE="$fixture_dir" bash "$LINT_SCRIPT" >/dev/null 2>&1
  local rc=$?
  if [ "$rc" != "0" ]; then
    echo "  PASS: $label (lint exited $rc as expected)"
    passes=$((passes + 1))
  else
    echo "  FAIL: $label (lint exited 0; expected non-zero)"
    fails=$((fails + 1))
  fi
}

# ----------------------------------------------------------------------
# Test 1: empty postgres adapter directory → OK
# ----------------------------------------------------------------------
test_empty() {
  local tmp
  tmp=$(mktemp -d)
  mkdir -p "$tmp/backend/internal/adapter/postgres"
  assert_lint_ok "empty adapter directory" "$tmp"
  rm -rf "$tmp"
}

# ----------------------------------------------------------------------
# Test 2: a query against organization_members → OK (safe table)
# ----------------------------------------------------------------------
test_safe_table_org_members() {
  local tmp
  tmp=$(mktemp -d)
  local dir="$tmp/backend/internal/adapter/postgres"
  mkdir -p "$dir"
  cat > "$dir/some_repo.go" <<'EOF'
package postgres
const q = `SELECT 1 FROM organization_members WHERE user_id = $1`
EOF
  assert_lint_ok "FROM organization_members WHERE user_id (safe table)" "$tmp"
  rm -rf "$tmp"
}

# ----------------------------------------------------------------------
# Test 3: a query against notifications → OK
# ----------------------------------------------------------------------
test_safe_table_notifications() {
  local tmp
  tmp=$(mktemp -d)
  local dir="$tmp/backend/internal/adapter/postgres"
  mkdir -p "$dir"
  cat > "$dir/some_repo.go" <<'EOF'
package postgres
const q = `SELECT 1 FROM notifications WHERE user_id = $1`
EOF
  assert_lint_ok "FROM notifications WHERE user_id (safe table)" "$tmp"
  rm -rf "$tmp"
}

# ----------------------------------------------------------------------
# Test 4: a query against proposals → FAIL (business state)
# ----------------------------------------------------------------------
test_business_state_proposals() {
  local tmp
  tmp=$(mktemp -d)
  local dir="$tmp/backend/internal/adapter/postgres"
  mkdir -p "$dir"
  cat > "$dir/proposal_repo.go" <<'EOF'
package postgres
const q = `SELECT * FROM proposals WHERE user_id = $1`
EOF
  assert_lint_fail "FROM proposals WHERE user_id (business state, must use org_id)" "$tmp"
  rm -rf "$tmp"
}

# ----------------------------------------------------------------------
# Test 5: a marker comment exempts the line → OK
# ----------------------------------------------------------------------
test_marker_comment() {
  local tmp
  tmp=$(mktemp -d)
  local dir="$tmp/backend/internal/adapter/postgres"
  mkdir -p "$dir"
  cat > "$dir/proposal_repo.go" <<'EOF'
package postgres
const q = `SELECT * FROM audit_logs WHERE user_id = $1` // authorship-by-user-ok
EOF
  assert_lint_ok "marker comment exempts the line" "$tmp"
  rm -rf "$tmp"
}

# ----------------------------------------------------------------------
# Test 6: an allowed file is never flagged
# ----------------------------------------------------------------------
test_allowed_file() {
  local tmp
  tmp=$(mktemp -d)
  local dir="$tmp/backend/internal/adapter/postgres"
  mkdir -p "$dir"
  cat > "$dir/notification_queries.go" <<'EOF'
package postgres
const q = `SELECT * FROM notifications WHERE user_id = $1`
EOF
  assert_lint_ok "allowed file (notification_queries.go) is skipped" "$tmp"
  rm -rf "$tmp"
}

# ----------------------------------------------------------------------
# Test 7: a _test.go file is skipped
# ----------------------------------------------------------------------
test_test_file_skipped() {
  local tmp
  tmp=$(mktemp -d)
  local dir="$tmp/backend/internal/adapter/postgres"
  mkdir -p "$dir"
  cat > "$dir/proposal_test.go" <<'EOF'
package postgres
const q = `SELECT * FROM proposals WHERE user_id = $1`
EOF
  assert_lint_ok "_test.go file skipped" "$tmp"
  rm -rf "$tmp"
}

# ----------------------------------------------------------------------
# Test 8: organization_members subquery within a business-state UPDATE → OK
# ----------------------------------------------------------------------
test_org_subquery_safe() {
  local tmp
  tmp=$(mktemp -d)
  local dir="$tmp/backend/internal/adapter/postgres"
  mkdir -p "$dir"
  cat > "$dir/job_queries.go" <<'EOF'
package postgres
const q = `INSERT INTO jobs (organization_id) VALUES (
  (SELECT organization_id FROM organization_members WHERE user_id = $1 LIMIT 1)
)`
EOF
  assert_lint_ok "organization_members subquery is allowed" "$tmp"
  rm -rf "$tmp"
}

# ----------------------------------------------------------------------
# Test 9: missing adapter directory → exits 1
# ----------------------------------------------------------------------
test_missing_dir() {
  local tmp
  tmp=$(mktemp -d)
  # No backend/internal/adapter/postgres directory.
  assert_lint_fail "missing adapter directory" "$tmp"
  rm -rf "$tmp"
}

# Run all tests.
echo "Running test-lint-org-scoping.sh tests:"
test_empty
test_safe_table_org_members
test_safe_table_notifications
test_business_state_proposals
test_marker_comment
test_allowed_file
test_test_file_skipped
test_org_subquery_safe
test_missing_dir

echo
echo "Result: $passes passed, $fails failed"

if [ "$fails" -gt 0 ]; then
  exit 1
fi
exit 0
