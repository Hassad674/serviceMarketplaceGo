# Migrations

Powered by [`golang-migrate`](https://github.com/golang-migrate/migrate). One numbered pair `XXX_name.up.sql` + `XXX_name.down.sql` per migration, applied in lexical order.

## Numbering gap: 024 and 025

The numbering jumps from `023_create_reports` to `026_remove_off_platform_payment_reason`. Versions `024` and `025` are intentionally unused — they were never created during early development. `golang-migrate` tolerates non-contiguous numbering, so the gap is harmless. Do **not** retroactively fill these slots; that would conflict with environments that already migrated past `023`.

## Convention for new migrations

1. **Idempotency** — every new migration must use `IF NOT EXISTS` / `IF EXISTS` clauses on `CREATE TABLE / CREATE INDEX / CREATE TYPE / DROP *`. This makes `make migrate-up` safe to retry on a partially-applied state.
2. **Up + down required** — every `XXX_name.up.sql` ships with a working `XXX_name.down.sql`. The down must be tested locally (`make migrate-down && make migrate-up`) before committing.
3. **Immutability** — once a migration is applied in production it is **never edited**. Mistakes are corrected by a *new* forward migration (e.g. `0NN_fix_xyz.up.sql`).
4. **Concurrency for index creation** — on tables that grow (`messages`, `proposals`, `payment_records`, `audit_logs`, `notifications`, `search_queries`), use `CREATE INDEX CONCURRENTLY` so the migration does not hold an `ACCESS EXCLUSIVE` lock during the build.
5. **Long-running backfills** — split bulk `UPDATE` statements into chunks committed separately; do not mix a schema change and a 10M-row backfill in the same transaction.
6. **Cross-feature foreign keys (revised rule, phase 5)** — the original rule was "only reference `users(id)`". The current schema admits a small set of business-driven FK between features because the linked entities cannot exist independently. Do not add new cross-feature FK casually — only when the ownership relationship is genuinely required by the business model. The full list of accepted FKs is below.
7. **Org-scoped ownership** — new tables holding business state always reference `organizations(id)`, never `users(id)` for ownership. `user_id` columns survive on some legacy tables (`proposals`, `disputes`, `reviews`, `payment_records`, `conversations`) as **write-only authorship** (audit / created_by). Reads must always filter by `organization_id` — enforced by `scripts/ci/lint-org-scoping.sh` on every PR.

## Accepted cross-feature foreign keys

The following cross-feature FKs are explicitly allowed because the linked entities cannot exist without their parent. Anything not on this list requires a written justification in the migration's docstring AND owner sign-off before being added.

| Source table              | FK column            | Target table       | Reason                                                                                  |
|---------------------------|----------------------|--------------------|-----------------------------------------------------------------------------------------|
| `disputes`                | `proposal_id`        | `proposals`        | A dispute literally cannot exist without the proposal it disputes.                      |
| `disputes`                | `job_id` (when set)  | `jobs`             | Job-stage disputes (pre-proposal phase) reference the originating job.                  |
| `reviews`                 | `proposal_id`        | `proposals`        | A review is the post-mortem of a single proposal/mission.                               |
| `reports`                 | `conversation_id`    | `conversations`    | Most reports originate from a conversation context. NULL allowed for global reports.    |
| `reports`                 | `message_id`         | `messages`         | Message-level reports point at the offending message.                                   |
| `payment_records`         | `proposal_id`        | `proposals`        | A payment record is created per milestone of a proposal.                                |
| `proposals`               | `conversation_id`    | `conversations`    | Proposals are sent within a conversation thread.                                        |
| `proposals`               | `parent_id` (self)   | `proposals`        | Modify-flow versioning chain (same feature, listed for completeness).                   |
| `invoices`                | `payment_record_id`  | `payment_records`  | Invoice is generated when payment_record completes (m.121).                             |
| `proposal_milestones`     | `proposal_id`        | `proposals`        | Same feature.                                                                           |
| `dispute_evidence`        | `dispute_id`         | `disputes`         | Same feature.                                                                           |
| `counter_proposals`       | `dispute_id`         | `disputes`         | Same feature.                                                                           |
| `dispute_ai_chat_messages`| `dispute_id`         | `disputes`         | Same feature.                                                                           |
| `messages`                | `conversation_id`    | `conversations`    | Same feature.                                                                           |
| `messages`                | `reply_to_id` (self) | `messages`         | Same feature.                                                                           |
| `message_history`         | `message_id`         | `messages`         | Same feature.                                                                           |
| `conversation_read_state` | `conversation_id`    | `conversations`    | Same feature.                                                                           |
| `job_applications`        | `job_id`             | `jobs`             | Same feature.                                                                           |
| `job_views`               | `job_id`             | `jobs`             | Same feature.                                                                           |
| `milestone_transitions`   | `proposal_id`        | `proposals`        | Same feature.                                                                           |

**To propose a new entry**: open a PR that updates this table together with the migration. Reviewer checklist:
- confirm the parent entity is required (deleting it should orphan the child),
- confirm `ON DELETE` semantics are documented,
- confirm the lint script in `scripts/ci/lint-org-scoping.sh` does not flag the new query patterns.

## Workflow

```
1. Author migration files            ->  XXX_name.up.sql + XXX_name.down.sql
2. Test locally on isolated DB        ->  createdb -T marketplace_go marketplace_go_<feat>
3. make migrate-up + verify schema    ->  psql \d <table>
4. Test rollback                      ->  make migrate-down + verify
5. Re-apply                           ->  make migrate-up
6. Commit + push                      ->  git commit + git push
7. Apply to prod                      ->  DATABASE_URL=<prod> make migrate-up
8. Drop the throwaway DB              ->  dropdb marketplace_go_<feat>
```

## Multi-agent safety

When several agents work in parallel on different feature branches, each agent that touches migrations **must** use its own DB copy (step 2 above). Running `make migrate-down` on the shared `marketplace_go` would roll back another agent's migrations and lose data — `migrate-down` is for the per-agent DB only. Forward-only fixes (new corrective migration) are the only safe pattern on the shared DB.
