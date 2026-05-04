#!/usr/bin/env bash
# design/scripts/check-imports-stable.sh
#
# Defense-in-depth on top of check-api-untouched.sh.
# Counts imports of api / hooks / zod schemas in the touchable surface
# (web/src/app, web/src/features/*/components, etc.) before and after
# the agent's diff. Fails if any counter changed materially.
#
# Why: a sneaky agent could add `import { useFoo } from '@/features/foo/hooks/use-foo'`
# inside a component and silently introduce business logic into UI code.
# This script catches that even if the hook file itself wasn't modified.
#
# The counters are:
#   - imports from `*/api/*`
#   - imports from `*/hooks/use-*`
#   - imports from `*/schemas/*`
#   - imports of `from "zod"` directly (new zod schemas being created in UI)
#
# A material change = the count went UP. Going down (refactoring) is OK,
# the agent flags it in the report.

set -uo pipefail

BASE_REF="${BASE_REF:-origin/main}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${REPO_ROOT}"

# Pattern groups
patterns=(
  '/api/'              # imports of any api module
  '/hooks/use-'        # imports of any TanStack Query hook
  '/schemas/'          # imports of any zod schema module
  'from "zod"'         # direct zod usage
)

# Surfaces to scan (where UI lives)
surfaces=(
  'web/src/app/'
  'web/src/features/'
  'web/src/shared/components/'
  'admin/src/'
  'mobile/lib/features/'
)

count_imports() {
  local ref="$1"
  local pattern="$2"
  local total=0
  for surface in "${surfaces[@]}"; do
    if [[ "${ref}" == "WORKING" ]]; then
      [[ -d "${surface}" ]] || continue
      n=$(grep -rE "import.*${pattern}" "${surface}" --include="*.ts" --include="*.tsx" --include="*.dart" 2>/dev/null | wc -l)
    else
      # Use git grep against a ref
      n=$(git grep -rE "import.*${pattern}" "${ref}" -- "${surface}" 2>/dev/null | wc -l)
    fi
    total=$(( total + n ))
  done
  echo "${total}"
}

# Verify the base ref exists
if ! git rev-parse --verify "${BASE_REF}" >/dev/null 2>&1; then
  echo "check-imports-stable: BASE_REF=${BASE_REF} not found; skipping (treat as clean)"
  exit 0
fi

echo "check-imports-stable: comparing against ${BASE_REF}"

failed=0
for pattern in "${patterns[@]}"; do
  before=$(count_imports "${BASE_REF}" "${pattern}")
  after=$(count_imports "WORKING" "${pattern}")
  delta=$(( after - before ))
  if [[ ${delta} -gt 0 ]]; then
    printf '  \033[1;31mFAIL\033[0m  pattern "%s"  before=%d  after=%d  delta=+%d\n' \
      "${pattern}" "${before}" "${after}" "${delta}"
    failed=1
  else
    printf '  \033[1;32mOK\033[0m    pattern "%s"  before=%d  after=%d  delta=%d\n' \
      "${pattern}" "${before}" "${after}" "${delta}"
  fi
done

if [[ ${failed} -eq 1 ]]; then
  echo
  echo "Imports of api/hooks/schemas grew. UI batches must NOT introduce new"
  echo "data dependencies. If you're consuming an existing hook from a screen"
  echo "that didn't have it before, that's normal — but flag it explicitly"
  echo "in the batch report. Otherwise revert."
  exit 1
fi

exit 0
