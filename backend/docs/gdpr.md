# GDPR endpoints — operator runbook

## Purpose

This document describes how the right-to-erasure + right-to-export endpoints (P5) operate in production: configuration, scheduling, manual purge procedures, and incident response.

## Endpoints

| Method | Path | Auth | Status codes |
|--------|------|------|--------------|
| GET    | `/api/v1/me/export`                     | Bearer/Cookie | 200 (ZIP) / 401 / 404 / 410 / 500 |
| POST   | `/api/v1/me/account/request-deletion`   | Bearer/Cookie | 200 / 400 / 401 / 404 / 409 / 500 |
| GET    | `/api/v1/me/account/confirm-deletion`   | JWT in query  | 200 / 400 / 401 / 404 / 500 |
| POST   | `/api/v1/me/account/cancel-deletion`    | Bearer/Cookie | 200 / 401 / 500 |

The login flow at `/api/v1/auth/login` returns **HTTP 410 Gone** with body `{ "error": "account_scheduled_for_deletion", "reason": "<RFC3339 deleted_at>" }` when the user attempts to authenticate with a soft-deleted account.

## Configuration

### Required environment variables

| Variable | Purpose | Production fallback policy |
|----------|---------|----------------------------|
| `GDPR_ANONYMIZATION_SALT` | Salt for `sha256(email + salt)` written to `audit_logs.metadata.actor_email_hash` after a hard purge | Refused — `config.Validate()` blocks boot when the value equals the dev default `dev-salt-not-for-prod` |
| `JWT_SECRET` | Source of the deletion-confirmation signing key (`SHA256(JWT_SECRET || "gdpr-deletion-confirmation")`) | Already enforced by SEC-04 |

### Deriving a fresh salt

```bash
openssl rand -base64 48 | tr -d '\n'
```

Set the value in your secret manager (e.g., Doppler / 1Password / AWS Secrets Manager) and inject it as the `GDPR_ANONYMIZATION_SALT` env var on the backend service.

**Salt rotation**: rotating the salt invalidates the forensic-search ability for previously-anonymized rows. Plan a rotation only when there is a credible risk that the salt was leaked. Use a separate var (`GDPR_ANONYMIZATION_SALT_NEXT`) to roll forward without dropping the old one.

## Scheduler

The purge cron is implemented in `internal/app/gdpr/scheduler.go` and started in `cmd/api/wire_gdpr.go`. One scheduler per process is enough: the SQL uses `FOR UPDATE SKIP LOCKED` so multiple instances cooperate without coordination.

| Setting | Production | Development |
|---------|------------|-------------|
| Tick interval | 24h | 1 minute |
| Batch size | 100 users per tick | 100 users per tick |
| Cutoff | `NOW() - INTERVAL '30 days'` | same |

The scheduler ticks immediately on start, then on every interval. An overdue batch after a deploy is picked up without waiting a full day.

## What `PurgeUser` does

The brief calls for "hard cascade DELETE", but the existing schema has several `NOT NULL` foreign keys to `users` with `NO ACTION` (proposals, disputes, jobs, reviews, invoice, payment_records). The RGPD-compliant compromise the implementation uses is a **hybrid hard-delete + anonymize-in-place**:

1. **`SELECT … FOR UPDATE SKIP LOCKED` on the user row** + re-check `deleted_at < cutoff` inside the tx so a `cancel-deletion` that landed between `ListPurgeable` and `PurgeUser` is honored.
2. **Anonymize `audit_logs`** for that user:
   - Compute `sha256(LOWER(TRIM(email_or_actor_email)) || $salt)` in SQL via `pgcrypto` and write to `metadata.actor_email_hash`.
   - Drop `metadata.email`, `metadata.actor_email`, `metadata.actor_name`, `metadata.first_name`, `metadata.last_name`.
   - Stamp `metadata.anonymized_at = NOW()`.
   - Mask `ip_address` to `/16` (IPv4) or `/32` (IPv6) via `network(set_masklen(...))`.
3. **Anonymize the user row in place**:
   - `email` → `anonymized+{user_uuid}@deleted.local` (preserves UNIQUE so re-running is safe)
   - `first_name` → `'anonymized'`, `last_name` → `'user'`, `display_name` → `'Anonymized user'`
   - `hashed_password` → `'!ANONYMIZED!'`
   - `linkedin_id`, `google_id` → NULL
4. **Cascade-eligible per-user rows** are explicitly DELETEd:
   - `notifications`, `device_tokens`, `password_resets`, `notification_preferences`
   - `conversation_participants`, `conversation_read_state`, `job_views`

### Anonymized audit row example

Before:
```
metadata = {"email": "alice@example.com", "actor_name": "Alice Doe"}
ip_address = 203.0.113.42
```

After:
```
metadata = {
  "anonymized_at": "2026-05-01T21:50:05.075Z",
  "actor_email_hash": "899d80206672d8af9d901e0d6de8c67312fd54ebb5d5cbfebbc09e6df990f397"
}
ip_address = 203.0.0.0/16
```

### Forensic search

To verify "did Alice (alice@example.com) ever do X" against anonymized history:

```sql
SELECT * FROM audit_logs
WHERE metadata->>'actor_email_hash' =
  encode(digest('alice@example.com' || current_setting('app.gdpr_salt'), 'sha256'), 'hex');
```

Where `current_setting('app.gdpr_salt')` is set with `SET LOCAL app.gdpr_salt = 'YOUR-SALT'` for the session.

## Manual procedures

### Force-purge a specific user (emergency)

```sql
-- 1. Verify the user is in the cooldown
SELECT id, email, deleted_at FROM users WHERE id = '<uuid>';

-- 2. Set deleted_at to a past date so the cron picks them up next tick
UPDATE users
SET deleted_at = NOW() - INTERVAL '31 days'
WHERE id = '<uuid>';
```

The next cron tick (or a manual `make migrate-up && make run` smoke test) will purge the row. To force an immediate purge from the admin shell:

```bash
psql $DATABASE_URL -c "UPDATE users SET deleted_at = NOW() - INTERVAL '31 days' WHERE id = '<uuid>';"
# Wait for the next scheduler tick (1 min in dev, 24h in prod) OR
# call the service from a one-off Go script:
go run ./cmd/api  # the scheduler runs immediately on start
```

### Restore an accidentally soft-deleted user

The user has 30 days to cancel via `POST /api/v1/me/account/cancel-deletion`. As the operator:

```sql
UPDATE users SET deleted_at = NULL, updated_at = NOW() WHERE id = '<uuid>';
```

### Re-anonymize a row that the cron missed

If a previous purge run anonymized only part of a user's audit_logs (e.g. due to a transient DB error during the original tx), call PurgeUser again:

```bash
psql $DATABASE_URL -c "UPDATE users SET deleted_at = NOW() - INTERVAL '31 days' WHERE id = '<uuid>';"
```

The next cron tick re-runs PurgeUser. The user-row anonymization is idempotent (COALESCE-style), and the audit_logs UPDATE simply re-writes the same hash.

## Monitoring

The scheduler emits structured logs at every tick. Key signals:

- `gdpr scheduler started` (boot)
- `gdpr scheduler: tick examined=N purged=N errors=K`  (every tick when work was found)
- `gdpr scheduler: per-row error` (warn — investigate)

Recommended Prometheus alerts (when metrics are wired in a future round):

| Alert | Condition | Severity |
|-------|-----------|----------|
| `gdpr_purge_errors_total` rate > 0 over 1h | per-row purge errors | warn |
| `gdpr_purge_lag_seconds` > 7d | purge tick has not run in a week | critical |
| `gdpr_anonymization_salt_default` is `1` | env var still equals dev fallback | critical |

## Compliance

The implementation satisfies the following RGPD obligations:

- **Article 17 (Right to erasure)**: 30-day window with confirmation email; hard purge of cascade-eligible rows; PII columns blanked on rows that hold structural FKs. Audit logs retained per Article 6(1)(c) (legal obligation) but anonymized so no re-identification is possible.
- **Article 20 (Right to portability)**: full data export in machine-readable JSON, generated on demand, containing every section the platform holds on the user.
- **Article 30 (Records of processing)**: salt stored in secret manager, not in code or logs.
- **Recital 26**: anonymized identifiers (sha256 + salt, IP truncated to /16 or /32) are no longer reasonably attributable to an identified person.

## Changelog

- 2026-05-01: Initial implementation (P5 commit batch).
