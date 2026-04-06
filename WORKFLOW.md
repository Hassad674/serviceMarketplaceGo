# Workflow — multi-agent revert safety

Practical rules for a solo dev running 3-5 parallel agents in different
worktrees. Keeps work revertable without enterprise-grade overhead.

---

## 1. Default: shared DB, isolated ONLY for destructive work

The default is a **shared** database. Isolate only when you'll break schema.

| What the agent does | DB |
|--------------------|-----|
| Add a table, column, index | **Shared** ✅ |
| Build a new feature (no schema removal) | **Shared** ✅ |
| No DB changes at all | **Shared** ✅ |
| `DROP TABLE` | **Isolated** 🔒 |
| `ALTER TABLE ... RENAME` | **Isolated** 🔒 |
| Major schema refactor | **Isolated** 🔒 |
| Experimenting with migrations (might `migrate-down`) | **Isolated** 🔒 |

**Why:** 90 % of work is additive and safe on shared DB. Isolation only
matters when your experimentation could destroy another agent's tables.

### Creating an isolated DB (only when needed)

```bash
# Template clone from the shared DB
createdb -h localhost -p 5435 -U postgres marketplace_feat_<name> -T marketplace_go

# In the worktree .env
DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_feat_<name>?sslmode=disable

# After merging to main
DATABASE_URL=<main> make migrate-up

# Cleanup
dropdb -h localhost -p 5435 -U postgres marketplace_feat_<name>
```

---

## 2. Before any destructive change: 30-second safety check

```bash
# Who is active right now?
git worktree list

# What just shipped to main?
git log main --oneline -10

# Is the thing I'm about to drop used anywhere else?
grep -rn "payment_info" --include="*.go" --include="*.ts" .
```

If all three are clean (no parallel agent on the same table) → proceed.
If one agent is on a branch that touches the table → coordinate first
(merge, rebase, or just wait).

---

## 3. Forward-only migrations

Migrations are **immutable once merged**. To change schema:

- Need to remove a table? → **new migration** `DROP TABLE`, don't edit the
  original `CREATE TABLE` migration.
- Need to fix a bad column? → **new migration** `ALTER TABLE`, don't edit.
- Need to rename? → **new migration**, don't edit.

**Why:** migration runners record the version applied. Editing a file
already applied in any environment means the change will never execute —
silent drift between environments.

### Before merging a migration

```bash
make migrate-up      # apply
# verify schema with psql
make migrate-down    # verify rollback works
make migrate-up      # re-apply
```

If it doesn't round-trip cleanly, don't merge.

---

## 4. Tag + archive before large deletions

If you're about to delete > 10 files or drop a table: create a restore
point. Costs 30 s, saves hours of reconstruction.

```bash
git tag -a vX.Y-before-big-change -m "Restore point before X"
git branch archive/before-X main
git push origin vX.Y-before-big-change archive/before-X
```

Example done once: `v0.9-kyc-custom-final` + `archive/kyc-custom-api`
before the KYC → Embedded migration.

---

## 5. Git discipline

### Atomic commits
One commit = one unit of logic that can be reverted in isolation. If the
subject contains "and", split it.

### `--no-ff` merges to main
Groups a feature's N commits under a single merge commit — one `git revert`
undoes the whole feature cleanly.

```bash
# Standard merge pattern (from a temp worktree)
git worktree add /tmp/main-merge main
cd /tmp/main-merge
git merge feat/my-branch --no-ff -m "feat: clear subject

Body explaining what + why.
"
git push origin main
cd - && git worktree remove /tmp/main-merge
```

### Never force-push to shared branches
`main` and any branch another agent bases work on: no `--force` /
`--force-with-lease` without explicit agreement.

### Never touch `main` directly from a feature worktree
Always merge via a temp worktree (pattern above). Keeps `main`'s working
tree out of scope for any one feature branch.

---

## 6. Cookies / dev server isolation

When two agents run web dev servers on different ports (e.g. 3000 and
3001), they **share cookies** because browsers scope cookies by domain
(`localhost`), not by port. That causes auth sessions to clash.

**Solution:** open each dev server in a **separate browser profile** or
private window. Cookies isolated by profile, zero config needed.

No need for `/etc/hosts` aliases at this scale — too much setup for the
gain.

---

## 7. Tests

### Broken `go build ./...` = emergency
Never merge, never push. Fix in the next 10 minutes or revert.

### Run the full suite before merging to main
```bash
cd backend && go test ./... -count=1
cd web && npx tsc --noEmit
# + playwright / smoke tests if relevant to the change
```

### Never skip or delete a test to pass the suite
Legitimately obsolete? Delete it in its OWN commit with the reason in
the message. Flaky? Don't `t.Skip()` without a `// TODO: fix — <why>`.

---

## 8. Reverting

### Reverting a feature merge
```bash
git revert -m 1 <merge-sha>
git push origin main
```
Creates a new commit that undoes the merge. History preserved, other
agents unaffected.

### Restoring from an archive
```bash
git checkout <tag-or-archive-branch>
# Or a new branch based on it
git checkout -b fix/restore-X <tag>
```

### Never `git reset --hard` on a shared branch
Use `git revert` on `main`. `reset --hard` rewrites history and breaks
every agent whose work is based on the reset commits.

---

## Pre-merge checklist

Before every push to `main`:

- [ ] Tests pass (`go test ./...`, `npx tsc --noEmit`, relevant E2E)
- [ ] Build passes (`go build ./...`)
- [ ] Migrations reversible (if any)
- [ ] `git worktree list` checked for conflicts on files I changed
- [ ] Large deletions preceded by tag + archive
- [ ] Commits atomic, messages tell the story
- [ ] Merge via `--no-ff` from a temp worktree

When in doubt: create a restore point, commit smaller, document more.
