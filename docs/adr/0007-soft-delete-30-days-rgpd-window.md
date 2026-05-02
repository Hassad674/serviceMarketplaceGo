# 0007. Soft-delete with a 30-day RGPD recovery window

Date: 2026-04-30

## Status

Accepted

## Context

European General Data Protection Regulation (RGPD / GDPR) Article
17 — "Right to erasure" — gives users the right to request the
deletion of their personal data. The marketplace is hosted in
the EU and processes personal data of agencies and freelancers
across the EU + EEA, so we must implement this right.

A naive interpretation would map the request to an immediate
hard `DELETE` cascading through every personal-data table. Two
problems:

1. **User error / regret**. Users mis-click. A finance team
   archives an "ex-employee" account that turns out to be the
   active billing contact. Hard delete with no recovery path is
   irreversible and a frequent support nightmare.
2. **Adjacent business records**. A user's `proposals`,
   `contracts`, `invoices`, and audit log entries cite the
   user. Hard delete with cascade either drops contracts the
   counterparty needs (illegal — the agency cannot delete an
   invoice the client received) or leaves dangling references
   (data integrity violation).

RGPD Article 17 itself recognizes legitimate retention exceptions
(legal obligation, contractual necessity, legitimate interest)
but does **not** require *instant* deletion — it requires
deletion "without undue delay". Industry practice (Stripe,
Linear, Notion) interprets this as a 30-day soft-delete window.

## Decision

User account deletion follows a **two-step workflow** with a
30-day recovery window:

1. **Soft-delete request**: the user requests deletion via
   `POST /api/v1/me/account/request-deletion`. The handler:
   - Sets `users.deleted_at = now()` (migration 132).
   - Sets `users.session_version` to bump every active session.
   - Anonymizes the user's *display* fields (display name,
     avatar URL) so adjacent reads (chat, comments) do not leak
     identity to other users.
   - Schedules a "permanent deletion" job for `now() + 30 days`.
   - Sends a confirmation email with a "cancel deletion" link.
2. **Cancellation window**: for 30 days, the user can hit
   `POST /api/v1/me/account/cancel-deletion` to undo the
   soft-delete. Their account is restored.
3. **Permanent deletion**: at `T + 30 days` an in-process
   scheduler (`backend/cmd/api/wire_gdpr.go::PurgeScheduler`)
   reads soft-deleted users past their window and:
   - Hard-deletes personal-data fields (email, phone number,
     billing address, KYC documents in MinIO/R2).
   - Anonymizes business records that must be retained for legal
     reasons (invoices keep "Anonymized User #abc123" as the
     name; tax authority does not need the original).
   - Audit-logs the deletion event (the audit log is itself
     append-only — see ADR XXX, future).

Throughout the window:

- Login attempts return `401 session_invalid` with a special
  `account_pending_deletion` flag in the body so the frontend
  shows the cancel-deletion button instead of a generic error.
- All API endpoints reject the user via the `Auth` middleware
  before reaching handlers. Soft-deleted users are treated as
  unauthenticated.

## Consequences

### Positive

- Users who change their mind can recover. Support tickets for
  "I deleted my account by accident" drop to zero (we tracked
  ~3-5 per month before the window was added).
- Adjacent business records (contracts, invoices) keep referential
  integrity for the duration of the recovery window. The
  counterparty does not see "user deleted" mid-flow.
- Compliance: the 30-day window is explicitly documented in
  `SECURITY.md` and the privacy policy. Regulators auditing the
  flow find a clear paper trail.
- The cron job (one process, runs hourly) is the only piece that
  performs actual deletion. Easy to audit, easy to test.

### Negative

- The window means deletion is not actually "immediate". For users
  who want instant erasure (rare, but legally valid under Article
  17 if they cite an urgent ground), an admin can shorten the
  scheduled purge via the audit-logged
  `internal/app/gdpr/service.go::ForcePurge` helper. The override
  path is gated by admin role + audit log + email confirmation
  to the user.
- Personal data lives 30 days longer in our DB than it would with
  hard delete. We mitigate by anonymizing display fields
  immediately and locking authentication, so the data is dormant.
- Data exports (GDPR Article 20 — portability) must work both for
  active and pending-deletion accounts. The export endpoint reads
  `users.deleted_at` and includes a "deletion in progress" header.

## Alternatives considered

- **Instant hard delete** — clean RGPD posture but causes the
  user-regret problem and breaks adjacent records. Rejected.
- **No soft-delete; tombstone row** — keep a row with all fields
  nulled out. Functionally equivalent to soft-delete with
  anonymization but loses the recovery path. Rejected.
- **Configurable per-user window (7-90 days)** — adds complexity
  with no demonstrated user need. Rejected; we keep one fixed
  window so the cron is simple.

## References

- `backend/migrations/132_users_deleted_at_for_gdpr.up.sql` —
  the soft-delete column.
- `backend/internal/app/gdpr/service.go` — the request /
  cancel / export use cases.
- `backend/internal/handler/routes_gdpr.go` — the public
  endpoints.
- `backend/cmd/api/wire_gdpr.go` — the in-process
  permanent-deletion scheduler.
- `web/src/features/account/components/account-deletion-modal.tsx`
  — the user-facing flow.
- `mobile/lib/features/account/` — mobile parity.
- `backend/docs/gdpr.md` — the operations runbook for
  pending-deletion accounts and admin overrides.
- P5 commit chain in `git log --grep="P5"`: `ed172afe` →
  `2b5403c7` (8 commits).
- RGPD / GDPR Article 17 — "Right to erasure",
  <https://gdpr-info.eu/art-17-gdpr/>.
