# P5 — GDPR endpoints (Export + Delete + Cancel + Confirm)

**Phase:** F.1 CRITICAL #5 (final F.1)
**Source audit:** SEC-FINAL (RGPD compliance for B2B EU contracts) — disqualifying without
**Effort:** 1j est.
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p5-gdpr-endpoints`

## Problem

The platform has no GDPR endpoints. Per audit and RGPD obligations:
- Users must be able to export their data
- Users must be able to delete their account (right to erasure)
- Audit logs must be anonymized post-deletion (not erased — legal forensic requirement)
- Cascade deletion must be safe + reversible (30-day window standard)

Without these endpoints, the platform is **disqualifying for any RGPD-conscious B2B enterprise contract in the EU**.

## 6 design decisions (LOCKED — user validated)

These are NOT to be revisited. Implement exactly as specified.

### Decision 1 — Export format
**ZIP file** containing:
```
export-{userID}-{timestamp}.zip
├── manifest.json     # metadata: user_id, timestamp, files included, version
├── README.txt        # human-readable explaining each file
├── profile.json      # user + profile + organization
├── proposals.json    # all proposals (as client + as provider)
├── messages.json     # all messages user sent or received
├── invoices.json     # all invoices
├── reviews.json      # reviews user wrote + received
├── audit_logs.json   # only actions performed BY the user
├── notifications.json
└── ...               # one file per main domain
```

Each `.json` is a JSON array of objects (not NDJSON, not CSV). Excel can open via "Get Data → JSON". The README.txt explains what each file contains in plain English (and French — both languages).

### Decision 2 — Audit logs after deletion
**Keep indefinitely, anonymize PII.** Required for legal forensic + RGPD compatible.

After hard-purge at T+30:
```sql
UPDATE audit_logs
SET
  user_id = user_id,  -- KEPT (forensic anchor, UUID, not PII)
  metadata = jsonb_set(
    jsonb_set(
      jsonb_set(
        jsonb_set(metadata, '{actor_email}', 'null'::jsonb),
        '{actor_name}', 'null'::jsonb
      ),
      '{actor_email_hash}', to_jsonb(encode(sha256((metadata->>'actor_email' || $salt)::bytea), 'hex'))
    ),
    '{ip_address}', to_jsonb(regexp_replace(metadata->>'ip_address', '(\d+\.\d+)\.\d+\.\d+', '\1.x.x'))
  )
WHERE user_id = $deleted_user_id;
```

Salt loaded from env var `GDPR_ANONYMIZATION_SALT` — fail-fast if missing in production. Dev fallback is `dev-salt-not-for-prod`.

### Decision 3 — Cascade strategy
**Soft-delete + 30-day window + cron purge.**

- T0: User clicks delete + password + email link → `users.deleted_at = NOW()` + email confirmation
- T0 to T+30j: User locked out (login refused with `account_scheduled_for_deletion`), all reads filter `WHERE deleted_at IS NULL`
- T+25j: Reminder email
- T+30j: Cron `gdpr_purge_scheduler` runs daily at 03:00 UTC, finds `users WHERE deleted_at < NOW() - INTERVAL '30 days'`, performs hard cascade DELETE + anonymizes audit_logs

User can cancel anytime in the 30-day window via `POST /api/v1/me/account/cancel-deletion` (still logged in via the temporary token in cancel email).

### Decision 4 — Anonymization
sha256(email + salt) → hex string in `audit_logs.metadata.actor_email_hash`. Drop name + phone + ip (truncate IP to first 16 bits). Forensic check possible: `WHERE metadata->>'actor_email_hash' = sha256('jean@x.com' + salt)`.

### Decision 5 — Confirmation deletion
2-step:
1. **Password re-prompt** modal with explicit checkbox "I understand my data will be deleted in 30 days unless I cancel"
2. Backend verifies password, sends email with **signed JWT link (TTL 24h, purpose=account_deletion, sub=user_id)**
3. User clicks link → `GET /api/v1/me/account/confirm-deletion?token=...` validates JWT, sets `users.deleted_at`

No SMS OTP (overkill, no infra). Standard GitHub/Stripe pattern.

### Decision 6 — Org owner edge case
If user is OWNER of an org with >0 other members:
- Backend returns `409 Conflict` with body:
  ```json
  {
    "error": {
      "code": "owner_must_transfer_or_dissolve",
      "message": "You own an organization with active members. Transfer ownership or dissolve before deleting your account.",
      "details": {
        "blocked_orgs": [
          {
            "org_id": "uuid",
            "org_name": "...",
            "member_count": N,
            "available_admins": [{ "user_id": "uuid", "email": "..." }],
            "actions": ["transfer_ownership", "dissolve_org"]
          }
        ]
      }
    }
  }
  ```
- Frontend renders inline remediation flow with action buttons.

## Implementation plan (8 commits)

### Commit 1 — Migration 132 + domain
- `migrations/132_users_deleted_at_for_gdpr.up.sql` (+ `.down.sql`)
  - `ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ NULL`
  - Partial index: `CREATE INDEX idx_users_pending_deletion ON users(deleted_at) WHERE deleted_at IS NOT NULL`
- `internal/domain/gdpr/` — export aggregate type, deletion request type, anonymization helpers
- Domain tests

### Commit 2 — Repository + service
- `internal/port/repository/gdpr_repository.go` — interface (read user data across tables, soft-delete user, hard-purge user, anonymize audit_logs)
- `internal/adapter/postgres/gdpr_repository.go` — SQL implementation
- `internal/app/gdpr/service.go` — orchestrator (ExportData, RequestDeletion, ConfirmDeletion, CancelDeletion)
- Service tests with sqlmock

### Commit 3 — Handler + middleware
- `internal/handler/gdpr_handler.go` :
  - `GET /api/v1/me/export` — sync export (small data) → returns ZIP file
  - `POST /api/v1/me/account/request-deletion` — verify password, send confirmation email
  - `GET /api/v1/me/account/confirm-deletion?token=...` — validate JWT, soft-delete
  - `POST /api/v1/me/account/cancel-deletion` — clear deleted_at
- Auth middleware update: refuse login if `users.deleted_at IS NOT NULL` with code `account_scheduled_for_deletion`
- Handler tests

### Commit 4 — Wire + cron scheduler
- `cmd/api/wire_gdpr.go` — wire helper following the post-P2 pattern
- `internal/scheduler/gdpr_purge.go` — daily cron, scans + purges + anonymizes
- main.go: 1 wire call + 1 scheduler kick
- Tests with testcontainers PG (cron flow integration)

### Commit 5 — Frontend web
- `web/src/features/account/api/gdpr.ts` — typed API client
- `web/src/app/[locale]/(app)/account/delete/page.tsx` — password modal + 30j countdown UI when `deleted_at` set
- `web/src/app/[locale]/(public)/account/confirm-deletion/page.tsx` — landing from email
- `web/src/app/[locale]/(app)/account/cancel-deletion/page.tsx` — landing from cancel email
- Banner on dashboard if `deleted_at` set: "Your account will be deleted on {date}. Click here to cancel."
- Org owner remediation modal with transfer/dissolve buttons
- Vitest tests

### Commit 6 — Mobile parity
- `mobile/lib/features/account/data/gdpr_repository_impl.dart`
- `mobile/lib/features/account/presentation/screens/delete_account_screen.dart`
- `mobile/lib/features/account/presentation/screens/cancel_deletion_screen.dart`
- Banner widget on dashboard
- Tests

### Commit 7 — i18n (FR + EN)
- New keys for all GDPR strings in `web/messages/en.json`, `web/messages/fr.json`
- Same in `mobile/lib/l10n/app_en.arb`, `app_fr.arb`
- Email templates (FR + EN) for: deletion confirmation, scheduled-deletion reminder T+25j, final purge T+30j

### Commit 8 — Integration test + docs
- `backend/test/gdpr_e2e_test.go` — full happy path : request → confirm → soft-delete → 30j skip → purge
- `backend/test/gdpr_owner_block_test.go` — org owner blocked
- `backend/docs/gdpr.md` — operator runbook (env vars, cron schedule, manual purge query for emergency)

## Hard constraints

- **Validation pipeline before EVERY commit**:
  ```bash
  go build ./... && go vet ./... && go test ./... -count=1 -short -race
  cd web && npx tsc --noEmit && npx vitest run
  cd mobile && flutter analyze && flutter test
  ```
- **Migration safety**: `IF NOT EXISTS` partout, `down.sql` symmetric.
- **No PII in logs**: every audit log entry that touches GDPR flow uses `metadata.email_hash` not `metadata.email` from day 1.
- **Idempotency**: confirm-deletion is safe to call multiple times (already-deleted = no-op return 200).
- **Race safety**: `cancel-deletion` + cron-purge race — purge cron must `SELECT ... FOR UPDATE SKIP LOCKED` and re-check `deleted_at IS NOT NULL AND deleted_at < NOW() - 30d` inside the transaction.
- **i18n MANDATORY**: every user-facing string FR + EN.
- **Mobile parity MANDATORY** per `feedback_mobile_parity.md`.
- **a11y**: delete modal accessible (focus trap, ESC, role=dialog, aria-describedby).

## OFF-LIMITS

- LiveKit / call code — never touch
- `.github/workflows/*` — token can't push
- Other plans — never touch

## Branch ownership

Agent creates `fix/p5-gdpr-endpoints` from clean `main` via `git worktree add`. Never touches another branch.

## Final report (under 1000 words)

Lead with PR URL.

1. 8 commits delivered (atomic per spec)
2. Migration 132 verified up + down green
3. Cron schedule details (interval, lookup query, daily run)
4. Anonymization SQL example output (before/after row)
5. i18n keys count (FR + EN)
6. Mobile parity confirmed (file count + test count)
7. Validation pipeline output (full paste)
8. "Branch ownership confirmed: only worked on `fix/p5-gdpr-endpoints`"

GO.
