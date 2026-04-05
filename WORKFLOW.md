# Workflow — multi-agent revert safety

This repo is worked on by many parallel agents in separate worktrees.
These rules keep work revertable and prevent one agent from breaking
another's in-progress branch.

---

## 1. Git discipline

### Atomic, themed commits
- One commit = one unit of logic that can be reverted in isolation.
- If you find yourself writing "and" in the commit subject, split it.
- Commit message: **what** in the subject, **why** in the body.

### Merge feature branches with `--no-ff`
- Each feature merge to `main` creates a single merge commit that groups
  its N child commits.
- Reverting a feature = `git revert -m 1 <merge-sha>`. One command, all
  related commits gone cleanly.
- Fast-forward merges make `main` history hard to revert — avoid them
  for non-trivial features.

### Never force-push to shared branches
- `main` and any branch another agent may base work on: no `--force` /
  `--force-with-lease` without explicit agreement.
- Never rewrite published history.

### Tag + archive branch before risky migrations
Before deleting a large chunk of code, renaming routes, or dropping
tables, create a restore point:
```bash
git tag -a vX.Y-descriptive-name -m "Restore point before X"
git branch archive/pre-X main
git push origin vX.Y-descriptive-name archive/pre-X
```
Example done once: `v0.9-kyc-custom-final` + `archive/kyc-custom-api`
before migrating KYC to Embedded Components.

---

## 2. Worktree & DB isolation

### Never touch `main` directly from inside a feature worktree
- Merge into `main` via a temporary worktree:
  ```bash
  git worktree add /tmp/main-merge main
  cd /tmp/main-merge
  git merge feat/my-branch --no-ff -m "..."
  git push origin main
  cd - && git worktree remove /tmp/main-merge
  ```
- This keeps `main` outside the scope of any single agent's active
  worktree.

### Each migration-touching worktree MUST use its own DB copy
- The shared `marketplace_go` DB must NEVER be modified by an agent
  working in a worktree.
- Pattern:
  ```bash
  # At worktree creation
  createdb -h localhost -p 5435 -U postgres marketplace_feat_<name> -T marketplace_go

  # In worktree .env
  DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_feat_<name>?sslmode=disable

  # After merge to main
  DATABASE_URL=<main> make migrate-up

  # Cleanup
  dropdb -h localhost -p 5435 -U postgres marketplace_feat_<name>
  ```
- An agent who runs `migrate-down` on a shared DB takes down every
  parallel agent's table. Isolated DB = blast radius = zero.

### Never edit files owned by another worktree's active branch
- Before touching a file used across features, check:
  ```bash
  git worktree list
  # Is another worktree on a branch that will conflict?
  ```
- If yes, coordinate (merge, rebase, or wait) before touching.

---

## 3. Migrations

### Sequential numbering, no gaps
- `040_x.up.sql`, `041_y.up.sql`, ... Never skip or reorder.
- Two agents creating migration 042 in parallel = conflict at merge time.
  Coordinate numbers in a shared tracker if multiple agents touch
  migrations simultaneously.

### Always write a functional `.down.sql`
- Every `up` must be reversible.
- For `DROP TABLE`: the `.down.sql` recreates the schema shell (data
  loss is acceptable on rollback — document it).

### Migrations are immutable once merged
- Applied in `main`? Don't edit — create a NEW corrective migration.
- Example: `040` created a bad column → don't edit 040, add `042_fix_x_column.up.sql`.

### Test rollback before merging
```bash
make migrate-up      # apply
# verify schema
make migrate-down    # rollback
make migrate-up      # re-apply
```
A migration that doesn't round-trip cleanly ≈ a live grenade.

---

## 4. Tests

### `go build ./...` broken = emergency
- Never merge, never push. Fix in the next 10 minutes or revert.
- A broken build blocks every other agent's test runs.

### Run the full test suite before any merge to `main`
```bash
cd backend && go test ./... -count=1
cd web && npx tsc --noEmit
# + relevant playwright/smoke tests
```

### Never skip or delete tests to pass the suite
- If a test is legitimately obsolete, delete it IN ITS OWN COMMIT with
  a clear justification.
- Comment out / `t.Skip()` only with a `// TODO: fix — <reason>` and a
  plan.

---

## 5. Code deletions

### Large deletions (>20 files or >500 lines) need a restore point
- Tag + archive branch BEFORE the delete (see §1).
- Deletion commit title: `refactor: delete X — replaced by Y`.

### Audit cross-references before deleting
```bash
grep -rn "TypeName\|func FuncName" --include="*.go" --include="*.ts"
```
Deleting a type/function used by another agent's branch = silent breakage
on their next rebase.

### Prefer deprecation over immediate delete for shared APIs
When an endpoint, table, or port interface has external consumers:
1. Add deprecation comment + log warning
2. Give agents a window to migrate
3. Delete in a separate commit after the window

---

## 6. Merging your feature into `main`

### Pre-merge checklist
- [ ] All tests green locally
- [ ] Build + TypeScript pass
- [ ] Migrations round-trip (up/down/up)
- [ ] No `console.log`, `TODO: fix`, dead code
- [ ] Commit messages tell the story
- [ ] `git merge-base --is-ancestor main <your-branch>` returns true
      (your branch includes the latest `main`) — if not, rebase first

### Merge command
```bash
git worktree add /tmp/main-merge main
cd /tmp/main-merge
git merge <your-branch> --no-ff -m "feat: <clear subject>

<body explaining what + why + notable changes>
"
git push origin main
cd - && git worktree remove /tmp/main-merge
```

### Post-merge cleanup
```bash
# Remove your feature worktree
git worktree remove .claude/worktrees/<name>

# Drop isolated DB
dropdb -h localhost -p 5435 -U postgres marketplace_feat_<name>

# Keep the remote branch as history (optional to delete)
# git push origin --delete <your-branch>
```

---

## 7. Reverting

### Reverting a feature merge
```bash
git revert -m 1 <merge-sha>
git push origin main
```
Creates a new commit that undoes the merge. Original history preserved.

### Restoring from archive
```bash
# From any worktree
git checkout <tag-or-archive-branch>
# Or create a new branch from it
git checkout -b fix/restore-X <tag>
```

### NEVER `git reset --hard` on a shared branch
- Use `git revert` to undo commits on `main` — creates a new commit, safe.
- `reset --hard` rewrites history and breaks every agent whose work is
  based on the reset commits.

---

## 8. Communication with parallel agents

### Before starting
- `git worktree list` — see who's active on what
- `git log main --oneline -10` — see what just shipped

### During work
- Don't rebase your branch on `main` while another agent is merging —
  races produce duplicate commits.
- If two branches touch the same file, the second to merge handles the
  conflict.

### After a big change
- Bump a note in the team channel: "main now at <sha>, watch for
  rebase conflicts on feature X".

---

## Summary checklist

Before every push to `main`:

- [ ] Tests pass
- [ ] Build passes
- [ ] Migrations reversible + tested
- [ ] `git worktree list` checked for conflicts
- [ ] Big deletions tagged + archived
- [ ] Commits atomic + well-labelled
- [ ] Merge via `--no-ff` from a temp worktree

When in doubt: create a restore point, commit smaller, document more.
