# D1+D2 — Commission apporteur auto-transfer + Retirer fallback

## Branch
`feat/commission-retire-flow`

## Scope (ni plus, ni moins)

### Backend (Go) — hexagonal
1. **Connect-ready gate** in `commission_distributor.go`:
   - Before `stripe.CreateTransfer`, query account capabilities (`payouts_enabled` + `charges_enabled`).
   - If NOT ready → `pending_kyc`, skip Stripe.
   - If ready → attempt transfer; success → `paid`, failure → `failed`.
2. **New retry endpoint** `POST /api/v1/wallet/commissions/{id}/retry`:
   - Auth + RBAC (apporteur owns commission via org).
   - Idempotency via existing middleware.
   - If `pending_kyc`/`failed` → re-check readiness → attempt transfer → update status.
   - If not ready → 422 `kyc_required` with onboarding URL.
   - If already `paid` → 409 `already_paid`.
3. **Webhook hooks** in stripe handler:
   - `transfer.failed` → mark commission `failed`.
   - `account.updated` → existing path already triggers KYC listener, ensure notification.
4. **Wallet reader extension** — add `GetCommissionsGroupedByStatus(referrerID)` returning groups with retire eligibility.
5. **Audit log** every retry attempt (`commission.retry_attempted`).

### Files to create/modify
- NEW `backend/internal/app/referral/commission_retry.go` — retry orchestrator method
- NEW `backend/internal/app/referral/commission_retry_test.go` — table-driven tests
- MOD `backend/internal/app/referral/commission_distributor.go` — wire payouts_enabled gate
- NEW `backend/internal/app/referral/commission_distributor_gate_test.go` — gate tests
- MOD `backend/internal/app/referral/wallet_reader.go` — add grouped reader
- NEW `backend/internal/port/service/referral_commission_retry.go` — port surface
- MOD `backend/internal/port/service/referral_wallet.go` — group structure
- MOD `backend/internal/port/service/referral_kyc_listener.go` — extend or reuse OnStripeAccountReady
- MOD `backend/internal/domain/audit/entity.go` — `ActionCommissionRetryAttempted`
- NEW `backend/internal/handler/wallet_commission_retry_handler.go` — handler
- NEW `backend/internal/handler/wallet_commission_retry_handler_test.go` — handler tests
- MOD `backend/internal/handler/routes_billing.go` — wire route
- MOD `backend/internal/handler/router_deps.go` — extend WalletHandler deps
- MOD `backend/internal/handler/wallet_handler.go` — inject retry service
- MOD `backend/internal/handler/openapi_catalog3.go` — OpenAPI spec
- MOD `backend/internal/adapter/stripe/webhook.go` — handle `transfer.failed` event projection
- MOD `backend/internal/handler/stripe_handler.go` — dispatch `transfer.failed`
- NEW `backend/internal/handler/stripe_transfer_failed.go` — transfer.failed handler glue
- MOD `backend/cmd/api/wire_referral.go` — wire commission retry service into handler

### Web (Next.js)
- MOD `web/src/features/wallet/api/wallet-api.ts` — add `retryCommission(id)` API call
- MOD `web/src/features/wallet/hooks/use-wallet.ts` — add mutation hook
- MOD `web/src/features/wallet/components/wallet-commission-list.tsx` — show Retirer button on `pending_kyc`/`failed`
- NEW `web/src/features/wallet/components/commission-kyc-required-modal.tsx` — KYC modal
- NEW unit tests
- NEW `web/e2e/wallet-commissions.spec.ts` — e2e
- MOD `web/messages/fr.json`, `web/messages/en.json` — i18n strings

### Mobile (Flutter)
- MOD `mobile/lib/features/wallet/...` (mirror web)
- ARB i18n entries

### Tests
- Backend unit (gate, retry orchestrator, wallet reader): `commission_distributor_gate_test.go`, `commission_retry_test.go`, `wallet_reader_test.go`.
- Backend handler tests: `wallet_commission_retry_handler_test.go` — 200/401/403/404/409/422 paths.
- Web vitest: button render, mutation call.
- Web Playwright: `wallet-commissions.spec.ts`.
- Mobile widget test.

## Approach order
1. Plan commit (this file)
2. Backend domain/port surface
3. Backend app retry + gate + tests
4. Backend handler + route + tests
5. Stripe webhook transfer.failed
6. Web API + UI + tests
7. Mobile parity
8. Validation pipeline + final report
