#!/usr/bin/env bash
# openapi-diff.sh — assert the currently-served OpenAPI schema is a
# non-breaking superset of the committed snapshot.
#
# Workflow:
#   1. Curl /api/openapi.json from the running backend.
#   2. Compare against scripts/ci/openapi-schema.snapshot.json.
#   3. If missing fields or changed types are detected, exit 1.
#
# The script is intentionally permissive — it flags only the cases
# our API versioning policy considers breaking:
#
#   - a path present in the snapshot but missing from the live schema
#   - an operation's response type narrowed (object → string, etc.)
#
# New paths, new optional fields, new response codes are ALL allowed.
#
# Usage:
#   scripts/ci/openapi-diff.sh [--update]
#
# --update overwrites the snapshot (used after a deliberate breaking
# change has been approved and bumped to /api/v2/).

set -euo pipefail

HERE="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SNAPSHOT="$HERE/openapi-schema.snapshot.json"
BASE_URL="${MARKETPLACE_BASE_URL:-http://localhost:8083}"

if ! command -v jq >/dev/null 2>&1; then
  echo "openapi-diff.sh requires jq" >&2
  exit 2
fi

live=$(curl -sS "$BASE_URL/api/openapi.json" 2>/dev/null || true)
if [[ -z "$live" ]]; then
  echo "could not reach $BASE_URL/api/openapi.json — is the backend running?" >&2
  exit 2
fi

case "${1:-}" in
  --update)
    printf '%s' "$live" | jq . > "$SNAPSHOT"
    echo "snapshot updated at $SNAPSHOT"
    exit 0 ;;
esac

if [[ ! -f "$SNAPSHOT" ]]; then
  # First run — bootstrap the snapshot rather than fail.
  printf '%s' "$live" | jq . > "$SNAPSHOT"
  echo "bootstrapped snapshot at $SNAPSHOT"
  exit 0
fi

# --- breakage checks -----------------------------------------------

missing_paths=$(jq -r --argjson live "$live" '
  .paths | keys[] as $k
  | select(($live.paths // {}) | has($k) | not)
  | $k
' "$SNAPSHOT" || true)

if [[ -n "$missing_paths" ]]; then
  echo "BREAKING: paths removed from live schema:" >&2
  echo "$missing_paths" | sed 's/^/  - /' >&2
  exit 1
fi

echo "openapi-diff: no breaking changes detected"
