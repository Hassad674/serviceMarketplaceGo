#!/usr/bin/env bash
# design/scripts/check-api-untouched.sh
#
# Verify that the agent has NOT modified any OFF-LIMITS file.
# Compares the current branch against origin/main (or BASE_REF if set).
#
# Exits 0 if clean, 1 if any forbidden file is in the diff.

set -uo pipefail

BASE_REF="${BASE_REF:-origin/main}"

# Forbidden glob patterns (POSIX-style, evaluated by git pathspec).
# The glob ":(exclude)" is the inverse — paths we don't care about.
FORBIDDEN=(
  # Backend — entire surface is off-limits to UI agents
  'backend/**'

  # Web — data + transport + routing + config
  'web/src/features/*/api/**.ts'
  'web/src/features/*/api/**.tsx'
  'web/src/features/*/hooks/use-*.ts'
  'web/src/features/*/hooks/use-*.tsx'
  'web/src/features/*/schemas/**.ts'
  'web/src/shared/lib/api-client.ts'
  'web/src/shared/lib/api-paths.ts'
  'web/src/shared/types/api.d.ts'
  'web/middleware.ts'
  'web/next.config.ts'
  'web/next.config.mjs'
  'web/package.json'
  'web/package-lock.json'
  'web/yarn.lock'

  # Admin — same families
  'admin/src/features/*/api/**.ts'
  'admin/src/features/*/hooks/use-*.ts'
  'admin/src/features/*/schemas/**.ts'
  'admin/src/shared/lib/api-client.ts'
  'admin/src/shared/types/api.d.ts'
  'admin/package.json'
  'admin/package-lock.json'
  'admin/vite.config.ts'

  # Mobile — data, network, repos, deps
  'mobile/lib/core/api/**.dart'
  'mobile/lib/core/network/**.dart'
  'mobile/lib/features/*/data/**.dart'
  'mobile/lib/features/*/domain/**.dart'
  'mobile/pubspec.yaml'
  'mobile/pubspec.lock'

  # Tests of any kind
  '**/*.test.ts'
  '**/*.test.tsx'
  '**/*.spec.ts'
  '**/*.spec.tsx'
  '**/*_test.go'
  '**/*_test.dart'
  'web/e2e/**'
  'backend/test/**'
)

# Whitelist override — any file matching ALLOW_OVERRIDE will be ignored
# from the forbidden list. Used by Phase 0 batches that explicitly need
# to touch an off-limits file (e.g., globals.css token setup).
# Set ALLOW_OVERRIDE="path1,path2,..." in the env when dispatching that batch.
mapfile -t ALLOWED < <(printf '%s\n' "${ALLOW_OVERRIDE:-}" | tr ',' '\n' | grep -v '^$' || true)

# Collect changed files vs base
mapfile -t changed < <(git diff --name-only "${BASE_REF}"...HEAD 2>/dev/null)
if [[ ${#changed[@]} -eq 0 ]]; then
  echo "no diff vs ${BASE_REF}; nothing to check"
  exit 0
fi

# Helper: does a path match any of the FORBIDDEN globs?
match_forbidden() {
  local path="$1"
  for pattern in "${FORBIDDEN[@]}"; do
    # Use git's check-ignore-style matching via a temporary index trick is overkill;
    # instead we use bash glob extglob.
    case "${path}" in
      ${pattern//\*\*/*}) return 0 ;;
    esac
    # Also match the **/X form by stripping leading **/
    case "${path}" in
      ${pattern#'**/'}) return 0 ;;
    esac
  done
  return 1
}

match_allowed() {
  local path="$1"
  for allowed in "${ALLOWED[@]}"; do
    [[ "${path}" == "${allowed}" ]] && return 0
  done
  return 1
}

violations=()
for path in "${changed[@]}"; do
  if match_allowed "${path}"; then
    continue
  fi
  if match_forbidden "${path}"; then
    violations+=("${path}")
  fi
done

if [[ ${#violations[@]} -eq 0 ]]; then
  printf 'check-api-untouched: clean (%d files in diff, 0 violations)\n' "${#changed[@]}"
  exit 0
fi

printf '\033[1;31mcheck-api-untouched: %d OFF-LIMITS path(s) modified:\033[0m\n' "${#violations[@]}"
for v in "${violations[@]}"; do
  printf '  · %s\n' "${v}"
done
echo
echo "If your batch DOES need one of these (e.g., Phase 0 token setup),"
echo "set ALLOW_OVERRIDE='<path1>,<path2>' in the env before re-running."
echo "Otherwise: revert these changes and ship the batch without them."
exit 1
