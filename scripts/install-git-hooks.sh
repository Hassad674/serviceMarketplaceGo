#!/usr/bin/env bash
# scripts/install-git-hooks.sh — wire the in-repo .githooks
# directory into the local clone's .git/hooks via git's
# `core.hooksPath` config.
#
# Why core.hooksPath rather than per-hook symlinks:
#   - One config line; works on every Unix and on Windows-Git-Bash.
#   - Updates to .githooks/ are picked up automatically (no
#     reinstall after `git pull`).
#   - `git config --unset core.hooksPath` cleanly reverts.
#
# Run once after the first clone:
#
#   $ ./scripts/install-git-hooks.sh
#
# After this, every `git commit` runs the pre-commit hook unless
# you pass `--no-verify`. See `.githooks/pre-commit` for the list
# of checks.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$REPO_ROOT" ]; then
  echo "error: not inside a git repository" >&2
  exit 1
fi

cd "$REPO_ROOT"

if [ ! -d ".githooks" ]; then
  echo "error: .githooks/ directory missing — are you on the right branch?" >&2
  exit 1
fi

if [ ! -x ".githooks/pre-commit" ]; then
  echo "warning: .githooks/pre-commit is not executable; fixing..." >&2
  chmod +x .githooks/pre-commit
fi

git config core.hooksPath .githooks

echo "[install-git-hooks] core.hooksPath set to .githooks/"
echo "[install-git-hooks] Pre-commit will now run gofmt + tsc + flutter analyze on staged files."
echo "[install-git-hooks] Skip with 'git commit --no-verify' (use sparingly)."
echo "[install-git-hooks] Revert with 'git config --unset core.hooksPath'."
