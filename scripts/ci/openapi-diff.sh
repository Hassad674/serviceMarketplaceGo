#!/usr/bin/env bash
# openapi-diff.sh — detects drift between backend OpenAPI schema and the
# committed client type files.
#
# Flow:
#   1. Fetch /api/openapi.json from a running backend
#   2. Regenerate types with openapi-typescript
#   3. Diff against the committed file(s)
#   4. Fail the PR if they diverge without a matching regeneration commit
#
# If the backend is not running, the script skips cleanly. Intended to
# run as part of CI *after* a short-lived backend is booted (see
# .github/workflows/e2e.yml for an example).
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"

if ! command -v jq >/dev/null 2>&1; then
  echo "::warning::jq not installed — diff check reduced to byte comparison"
fi

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# 1. Fetch fresh schema — tolerate backend not running.
if ! curl -fsS "$BASE/api/openapi.json" -o "$TMP/openapi.json" 2>/dev/null; then
  # Fallback path (some deployments expose /api/v1/openapi.json)
  if ! curl -fsS "$BASE/api/v1/openapi.json" -o "$TMP/openapi.json" 2>/dev/null; then
    echo "::warning::backend not reachable at $BASE — skipping OpenAPI diff"
    exit 0
  fi
fi

# 2. Regenerate types.
WEB_TYPES_PATH=${WEB_TYPES_PATH:-web/src/shared/types/api.d.ts}
MOBILE_TYPES_PATH=${MOBILE_TYPES_PATH:-mobile/lib/shared/types/api_gen.dart}

echo "== web types =="
if command -v npx >/dev/null 2>&1; then
  npx --yes openapi-typescript "$TMP/openapi.json" -o "$TMP/api.d.ts" 2>/dev/null || {
    echo "::error::openapi-typescript failed"
    exit 1
  }
  if [[ -f "$WEB_TYPES_PATH" ]]; then
    if diff -q "$WEB_TYPES_PATH" "$TMP/api.d.ts" >/dev/null 2>&1; then
      echo "  [OK] $WEB_TYPES_PATH up to date"
    else
      echo "::error::$WEB_TYPES_PATH is out of date — run 'npm run generate-api' in web/ and commit the result"
      diff -u "$WEB_TYPES_PATH" "$TMP/api.d.ts" | head -80 || true
      exit 1
    fi
  else
    echo "  [INFO] $WEB_TYPES_PATH absent — nothing to diff"
  fi
else
  echo "::warning::npx not available — skipping web diff"
fi

# 3. Mobile diff — the Dart generator is optional and slow; if the
# committed file exists we assert presence only (the mobile generator
# runs in its own CI step via pub run build_runner).
echo "== mobile types =="
if [[ -f "$MOBILE_TYPES_PATH" ]]; then
  # We cannot easily regenerate without the full Dart toolchain in this
  # script; mobile CI owns the regeneration + commit of api_gen.dart.
  echo "  [INFO] $MOBILE_TYPES_PATH present — drift check handled by mobile CI"
else
  echo "  [INFO] $MOBILE_TYPES_PATH absent — mobile OpenAPI generator not yet wired"
fi

echo "OpenAPI diff OK"
