#!/usr/bin/env bash
# scripts/test-pre-commit.sh — smoke test the pre-commit hook.
#
# Stages a deliberately mis-formatted Go file, invokes the hook
# directly, and asserts:
#   1. the hook exits non-zero (commit would have been refused),
#   2. the failure message points at gofmt.
#
# Run from the repo root:
#
#   $ ./scripts/test-pre-commit.sh
#
# Used by maintainers to verify the hook still fails closed when
# the format check is broken — a regression on the hook itself
# would silently let unformatted code into main otherwise.
#
# Exit codes:
#   0 — hook correctly rejected the bad input.
#   1 — hook accepted the bad input (unexpected).

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# Sanity: the hook must exist and be executable.
if [ ! -x ".githooks/pre-commit" ]; then
  echo "error: .githooks/pre-commit not executable" >&2
  exit 1
fi

# Generate a temporary mis-formatted Go file at a safe path and
# stage it. The file is removed (and the index restored) at the
# end no matter what.
TMP_GO="backend/internal/.precommit_test_FIXTURE.go"
trap 'rm -f "$TMP_GO" 2>/dev/null || true; git reset HEAD -- "$TMP_GO" 2>/dev/null || true' EXIT

cat > "$TMP_GO" <<'EOF'
package internal

func   bad_format ( ) string  {return    "spaces galore"}
EOF

git add "$TMP_GO"

# Invoke the hook. Capture exit code without `set -e` interfering.
set +e
HOOK_OUTPUT="$(.githooks/pre-commit 2>&1)"
HOOK_EXIT=$?
set -e

# Assertion 1 — the hook must have failed.
if [ "$HOOK_EXIT" -eq 0 ]; then
  echo "FAIL: pre-commit hook accepted unformatted Go (expected exit != 0)" >&2
  printf '%s\n' "$HOOK_OUTPUT" >&2
  exit 1
fi

# Assertion 2 — the failure message must mention gofmt.
if ! printf '%s' "$HOOK_OUTPUT" | grep -qi "gofmt"; then
  echo "FAIL: hook failed but did not cite gofmt — likely a different check fired" >&2
  printf '%s\n' "$HOOK_OUTPUT" >&2
  exit 1
fi

echo "PASS: pre-commit hook correctly rejected unformatted Go"
exit 0
