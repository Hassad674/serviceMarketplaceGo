# Agent Robustness Rules

> Source: hard-learned lesson from agent `a853cbad084cb5c8c` (Mobile dashboard parity), 2026-05-10 — ran 4086s / 67 tool uses with **zero commits** before stream idle timeout. All work lost.

## Why this exists

When an agent runs in worktree isolation and hits a runtime failure (stream idle timeout, context overflow, host crash), the worktree may be auto-cleaned. Anything not pushed to `origin/main` or to a remote branch is **gone forever**.

A single hour of agent work with no commits = a single hour of agent work permanently lost.

## The 5 hard rules — bake these into EVERY agent brief

### 1. First commit ≤ 10 tool uses (HARD)

> "Your 2nd or 3rd tool use MUST be a `git commit`. Create a placeholder file or stub if needed. Commit message: `wip(<scope>): agent <id> in flight`. Push immediately. ONLY THEN start the real work."

Rationale: guarantees the branch exists with at least one commit, so a follow-up agent can pick up from there.

### 2. Commit each file as soon as it compiles

> "After every file you create or substantially modify, the immediate next action is `git commit` + `git push`. NOT after multiple files. NOT after a logical chunk. After EACH file."

Rationale: 5 separate commits over an hour means 80% of work survives a mid-flight failure. One big commit means 0% survives.

### 3. Heavy tests run ONCE, at the very end

> "DO NOT run `flutter test`, `playwright test`, `npm run test`, or any test suite during incremental development. Write the tests in files, commit them, and run the suite ONLY in the very last commit before the final report. If the test suite times out, you've still pushed all the test files — partial work survives."

Rationale: Long-running tests are the #1 cause of stream idle timeouts. They produce no output for minutes at a time.

### 4. Wrap every potentially-long command with `timeout`

> "EVERY command that could hang gets `timeout 90 <cmd>`. Examples:
> - `timeout 60 npm install --no-audit --no-fund <pkg>`
> - `timeout 120 flutter test test/<single-file>` (one file, never the full suite during dev)
> - `timeout 30 go test ./internal/<single-package>/... -count=1 -short`
> - `timeout 90 npx tsc --noEmit`
>
> If a command hits its timeout, that's a signal to commit-and-stop, not retry."

Rationale: If a command hangs, the agent never produces output, the stream goes idle, the runtime kills the agent.

### 5. Push after every commit, never batched

> "Pattern: `commit -m 'X' && git push` as a single bash invocation. Never `commit && commit && commit && push` — push each time. If protection rejects, fall back to `git push origin <branch>` and continue. NEVER hold commits locally for later."

Rationale: A pushed commit on a feature branch survives the worktree being deleted.

---

## Brief template snippet — copy-paste into every agent dispatch

```markdown
## Robustness rules (NON-NEGOTIABLE — see `.claude/agent-robustness.md`)

You are working in an isolated worktree that may be deleted on failure. To prevent total work loss:

1. **First commit ≤ 10 tool uses**: your 2nd/3rd tool use is a `git commit` of a placeholder/stub. Push it. Branch: `wip/<scope>` if main is protected.
2. **Commit each file as soon as it compiles** — `git add <file> && git commit -m '...' && git push` after EACH file. No batching.
3. **No test suite mid-flight**: write tests in files + commit them, but DO NOT run `flutter test` / `playwright test` / `npm run test` until the final commit. Run targeted single-file tests with `timeout 30 ...` if you really must verify.
4. **Wrap long commands with `timeout`**: `timeout 90 npx tsc --noEmit`, `timeout 60 npm install --no-audit --no-fund`, etc.
5. **Push immediately after each commit** — never hold commits locally.

If you hit any timeout, push everything you have and stop. Do not retry — report what's done.
```

---

## Sizing rule — keep briefs realistic

A single agent brief should be:

- **Estimated effort ≤ 3-4h** (real work, not pad)
- **≤ 10 new files** to create
- **≤ 5 existing files** to substantially modify
- **One concern at a time** — don't ask for "feature X + tests + mobile parity" in one agent. Split.

If a feature is big enough to need 2+ agents, plan them sequentially, with the second agent depending on the first agent's commit being on `main`.

---

## Recovery pattern when an agent dies mid-flight

1. `git fetch origin --quiet`
2. `git ls-remote origin | grep <agent-branch-name>` — check if anything was pushed
3. `git log --all --oneline --since='2 hours ago'` — find any orphan commits
4. `git reflog show --all | grep <agent-id>` — find branches the agent created locally
5. If commits exist: cherry-pick or merge them, continue from there
6. If zero commits: dispatch a fresh agent with a tighter brief, mention what failed last time
