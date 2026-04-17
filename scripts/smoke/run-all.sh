#!/usr/bin/env bash
# run-all.sh — orchestrator for the three smoke suites (search, ops,
# security). Each sub-script is run sequentially so the output stays
# readable; exit code is the first non-zero exit seen.
#
# Usage:
#   scripts/smoke/run-all.sh [--env local|staging|prod] [--yes-i-know]

set -euo pipefail

HERE="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_common.sh
source "$HERE/_common.sh"
smoke_parse_env "$@"

args=("$@")
status=0

for script in search.sh ops.sh security.sh; do
  printf '\n%s▶ running %s%s\n' "$C_BOLD" "$script" "$C_RESET"
  if ! "$HERE/$script" "${args[@]}"; then
    status=1
  fi
done

exit "$status"
