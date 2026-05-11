# WALLET-UNIFY Run A — Plan

Run A of 4 sequential runs. Migration 153 (referral_attributions.ended_at) is already applied. This run delivers the backend foundation for "End an intro" so subsequent runs (B/C/D) can extend it to web, mobile, and wallet.

## Scope items vs file plan

### A.1 — Domain field `EndedAt`
- `backend/internal/domain/referral/attribution.go`
  - Add `EndedAt *time.Time` to `Attribution`.
  - Add method `(*Attribution).IsEnded() bool`.
- `backend/internal/domain/referral/attribution_test.go`
  - Add `TestAttribution_IsEnded` (table-driven: nil vs set).

### A.2 — Adapter refactor (scanAttribution + INSERT extension)
- `backend/internal/adapter/postgres/referral_repository.go`
  - Centralise the 4 scan call-sites (FindAttributionByProposal, FindAttributionByID, ListAttributionsByReferral, ListAttributionsByReferralIDs) via a new `scanAttribution(row attributionScanner)` helper.
  - INSERT statement updated to write `ended_at` (always NULL on create).
- `backend/internal/adapter/postgres/referral_queries.go`
  - Add `ended_at` to every attribution SELECT column list + INSERT VALUES.
  - Add `queryEndAttribution` (UPDATE with WHERE rbac via JOIN to referrals.referrer_id).

### A.3 — Repository method: `EndAttribution(ctx, id, referrerID)`
- `backend/internal/adapter/postgres/referral_repository.go`
  - SQL: `UPDATE referral_attributions a SET ended_at = NOW() FROM referrals r WHERE a.referral_id = r.id AND a.id = $1 AND r.referrer_id = $2 AND a.ended_at IS NULL`.
  - Disambiguates: not found (no row matching id+referrer) vs already-ended (row exists with `ended_at NOT NULL`). Achieved by a follow-up `SELECT ended_at FROM referral_attributions WHERE id = $1` when UPDATE affected 0 rows, then checking ownership.
  - Returns `referral.ErrAttributionNotFound` if attribution does not exist or is not owned by `referrerID`.
  - Returns new sentinel `referral.ErrAttributionAlreadyEnded` if already ended.

### A.4 — Port repository interface
- `backend/internal/port/repository/referral_repository.go`
  - Add `EndAttribution(ctx context.Context, attributionID, referrerID uuid.UUID) error` to `ReferralRepository`.
- `backend/internal/app/referral/mocks_test.go`
  - Add `EndAttribution` to `fakeReferralRepo` (matches in-memory map semantics + already-ended check).

### A.5 — App service `EndIntroAttribution`
- `backend/internal/app/referral/service_lifecycle.go` (already small at 63 lines, room to add).
  - Add `EndIntroAttribution(ctx, attributionID, actorUserID uuid.UUID) (*referral.Attribution, error)`.
  - 1. Load attribution by id, then parent referral, verify actorUserID == referral.ReferrerID; otherwise return `ErrNotAuthorized`. (RBAC primary defense remains the repo UPDATE.)
  - 2. Call `EndAttribution`. If `ErrAttributionAlreadyEnded`, reload and return idempotently with no extra notif/audit.
  - 3. Reload attribution to get the timestamp.
  - 4. Emit audit (new `ActionReferralIntroAttributionEnded`).
  - 5. Notify both provider + client parties (existing `TypeReferralIntroTerminated` reused — same human meaning) — best-effort.
- `backend/internal/domain/audit/entity.go`
  - Add `ActionReferralIntroAttributionEnded` constant.
  - Add `ResourceTypeReferralAttribution` constant.
- `backend/internal/app/referral/service_test.go`
  - 4 tests: success+idempotency, not-owner, not-found, already-ended idempotent path.

### A.6 — Handler endpoint
- `backend/internal/handler/referral_handler.go`
  - Add `EndAttribution(w, r)` handler:
    - Parse `{id}` from URL.
    - Call `svc.EndIntroAttribution(ctx, id, userID)`.
    - Return 200 `{"data":{"id":"...","ended_at":"..."}}`.
- `backend/internal/handler/routes_referral_dispute.go`
  - Wire `r.With(idem).Post("/attributions/{id}/end", deps.Referral.EndAttribution)` inside `/referrals`. Note: this lives under `/referrals` root so the URL is `/api/v1/referrals/attributions/{id}/end`.
- `backend/internal/handler/dto/response/...` — already exposed via the attribution struct; we add a minimal response struct only if needed (will reuse `response.NewAttributionResponse` if exists or inline a small `endAttributionResponse`).
- Handler error mapping in `handleReferralError`: map `ErrAttributionAlreadyEnded` to a 200 idempotent path (handled by service idempotency, so handler doesn't need a special branch).

### A.7 — Commission distributor gate
- `backend/internal/app/referral/commission_distributor.go` (DistributeIfApplicable and PrepareCommissionForMilestone)
  - After `FindAttributionByProposal`, if `att.IsEnded() && !approvedAt.Before(*att.EndedAt)`, skip commission creation. The brief specifies `milestone.ApprovedAt`. Inputs to both methods do NOT currently carry `ApprovedAt`. Let me re-read the input struct to confirm.
  - **Deviation flag**: We need a milestone approval timestamp. `service.ReferralCommissionDistributorInput` is the input. If it has no ApprovedAt, we use `time.Now()` as the proxy ONLY in this run — call it out as a tech debt for Run B+. (Will check input struct before coding.)

### A.8 — Notifications
- Reuse `notification.TypeReferralIntroTerminated` for the end-intro event (same human meaning, fits in i18n already). No new constant required — the brief says "Add a notification.Kind constant for the new event" but the existing one is a 100% fit; reusing avoids a duplicate. Document the choice inline as a justified deviation.

## Test plan (counts per file)

| File | Test count |
|------|------------|
| domain/referral/attribution_test.go | +1 (IsEnded) |
| adapter/postgres/referral_repository_test.go | +2 (EndAttribution round-trip, EndAttribution already-ended) + 1 helper-coverage (scanAttribution roundtrip via existing Find) |
| app/referral/service_test.go (new file or extension) | +4 (EndIntroAttribution_Success, _Idempotent, _NotOwner, _NotFound) + 1 (notifs+audit) |
| app/referral/commission_distributor_gate_test.go | +8-case table (existing file extended) |
| handler/referral_handler_unit_test.go | +1 (404 not found, 403 cross-tenant, 200 success, 200 idempotent) — total 4 subtests |

## Commit sequence

1. `_plan_run_a.md` (this file).
2. Adapter refactor (centralise scan + queries include ended_at, INSERT, EndAttribution method) — repository tests.
3. Domain `EndedAt` + `IsEnded` + audit Action constant + ResourceType.
4. Port interface `EndAttribution` + fake mock implementation.
5. App service `EndIntroAttribution` + unit tests.
6. Handler endpoint + route wiring + handler unit tests.
7. Commission distributor gate + distributor tests.

## Deviations / open questions

- **A.3 RBAC schema**: `referrals.referrer_id` is a `user_id` (per migration 105), NOT `referrer_org_id`. The brief says "verify referrer_org_id is on referrals or referrer_user_id". Verified: it is `referrer_id` (user). The repository method takes a `referrerID uuid.UUID` and matches `r.referrer_id`. No org-level RBAC needed since the referral is per-user-owned currently (org-scoping is a future migration per `project_org_based_model.md`).
- **A.7 milestone ApprovedAt**: input contracts for the distributor + preparer ports may not include the approval timestamp. The gate will be expressed as "if attribution is ended, the milestone is being approved NOW, so compare `time.Now()` against `att.EndedAt`". This is consistent because both `PrepareCommissionForMilestone` and `DistributeIfApplicable` are called synchronously from the milestone-approval flow. Will verify before coding.
- **A.8 notification constant**: reusing `TypeReferralIntroTerminated` — flagged here, no new constant.
