#!/usr/bin/env bash
# design/scripts/validate-no-regression.sh
#
# Master validation gate. Run before every commit on a design batch.
# Exits non-zero on any failure. Prints noisy output on purpose so you
# notice failures.
#
# Usage:
#   design/scripts/validate-no-regression.sh
#
# Optional env:
#   SKIP_BACKEND=1   skip the backend build/test (rare; default runs them)
#   SKIP_WEB=1       skip web tsc + vitest
#   SKIP_ADMIN=1     skip admin tsc + vitest
#   SKIP_MOBILE=1    skip flutter analyze + test
#
# This is a *gate*, not a *fixer*. It tells you what's wrong; you fix it
# yourself. Never edit this script to make it pass.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${REPO_ROOT}"

failures=()

print_step() {
  printf '\n\033[1;36m===> %s\033[0m\n' "$*"
}

print_ok() {
  printf '\033[1;32m  OK\033[0m %s\n' "$*"
}

print_fail() {
  printf '\033[1;31m  FAIL\033[0m %s\n' "$*"
  failures+=("$*")
}

# ───────────────────────────────────────────────────────────────────
# 1. OFF-LIMITS check (the most important guardrail)
# ───────────────────────────────────────────────────────────────────
print_step "1/6 · check-api-untouched.sh — verify OFF-LIMITS files were not modified"
if "${SCRIPT_DIR}/check-api-untouched.sh"; then
  print_ok "no off-limits paths touched"
else
  print_fail "OFF-LIMITS paths were modified — see output above"
fi

# ───────────────────────────────────────────────────────────────────
# 2. Imports stability — no new imports of api/hooks/zod
# ───────────────────────────────────────────────────────────────────
print_step "2/6 · check-imports-stable.sh — verify import counters stable"
if "${SCRIPT_DIR}/check-imports-stable.sh"; then
  print_ok "import counts unchanged"
else
  print_fail "import count changed (api/hooks/zod) — see output above"
fi

# ───────────────────────────────────────────────────────────────────
# 3. Backend build/vet/test (must stay 100% green)
# ───────────────────────────────────────────────────────────────────
if [[ "${SKIP_BACKEND:-}" != "1" ]]; then
  print_step "3/6 · backend · go build + vet + test"
  if (cd backend && go build ./... 2>&1 && go vet ./... 2>&1 && go test ./... -count=1 -race 2>&1 | tail -50); then
    print_ok "backend green"
  else
    print_fail "backend build/vet/test failure"
  fi
else
  print_step "3/6 · backend SKIPPED (SKIP_BACKEND=1)"
fi

# ───────────────────────────────────────────────────────────────────
# 4. Web tsc + vitest
# ───────────────────────────────────────────────────────────────────
if [[ "${SKIP_WEB:-}" != "1" ]] && [[ -d web ]]; then
  print_step "4/6 · web · tsc --noEmit + vitest run"
  if (cd web && npx --no-install tsc --noEmit 2>&1 && npx --no-install vitest run --reporter=dot 2>&1 | tail -30); then
    print_ok "web green"
  else
    print_fail "web tsc or vitest failure"
  fi
else
  print_step "4/6 · web SKIPPED"
fi

# ───────────────────────────────────────────────────────────────────
# 5. Admin tsc + vitest
# ───────────────────────────────────────────────────────────────────
if [[ "${SKIP_ADMIN:-}" != "1" ]] && [[ -d admin ]]; then
  print_step "5/6 · admin · tsc --noEmit + vitest"
  if (cd admin && npx --no-install tsc --noEmit 2>&1 && npx --no-install vitest run --reporter=dot 2>&1 | tail -30); then
    print_ok "admin green"
  else
    print_fail "admin tsc or vitest failure"
  fi
else
  print_step "5/6 · admin SKIPPED"
fi

# ───────────────────────────────────────────────────────────────────
# 6. Mobile flutter analyze + test (scoped to touched dirs only)
# ───────────────────────────────────────────────────────────────────
if [[ "${SKIP_MOBILE:-}" != "1" ]] && [[ -d mobile ]]; then
  print_step "6/6 · mobile · flutter analyze + test (scoped)"
  # Identify touched mobile dirs from the diff vs origin/main
  mapfile -t mobile_dirs < <(git diff --name-only origin/main...HEAD -- 'mobile/lib/**' 2>/dev/null \
    | sed -E 's#(mobile/lib/[^/]+/[^/]+)/.*#\1#' \
    | sort -u)
  if [[ ${#mobile_dirs[@]} -eq 0 ]]; then
    print_ok "no mobile changes; skipping flutter analyze"
  else
    if (cd mobile && flutter analyze --no-pub "${mobile_dirs[@]#mobile/}" 2>&1 | tail -30 \
        && flutter test --no-pub 2>&1 | tail -30); then
      print_ok "mobile green"
    else
      print_fail "mobile analyze or test failure"
    fi
  fi
else
  print_step "6/6 · mobile SKIPPED"
fi

# ───────────────────────────────────────────────────────────────────
# Summary
# ───────────────────────────────────────────────────────────────────
echo
echo "──────────────────────────────────────────────"
if [[ ${#failures[@]} -eq 0 ]]; then
  printf '\033[1;32mALL GREEN.\033[0m Safe to commit.\n'
  exit 0
else
  printf '\033[1;31mFAILED:\033[0m %d step(s) failed:\n' "${#failures[@]}"
  for f in "${failures[@]}"; do
    printf '  · %s\n' "${f}"
  done
  echo
  echo "Do NOT commit. Fix the underlying issue and re-run."
  exit 1
fi
