import { apiClient } from "@/shared/lib/api-client"

import type { Get, Post } from "@/shared/lib/api-paths"
export type WalletRecord = {
  id: string
  proposal_id: string
  milestone_id?: string
  proposal_amount: number
  platform_fee: number
  provider_payout: number
  payment_status: string
  transfer_status: string
  mission_status: string
  created_at: string
}

export type CommissionWallet = {
  pending_cents: number
  pending_kyc_cents: number
  paid_cents: number
  clawed_back_cents: number
  currency: string
}

export type WalletCommissionRecord = {
  id: string
  referral_id?: string
  proposal_id?: string
  milestone_id?: string
  gross_amount_cents: number
  commission_cents: number
  currency: string
  status: string
  stripe_transfer_id?: string
  paid_at?: string
  clawed_back_at?: string
  created_at: string
  /**
   * retire_eligible = backend-authoritative flag that mirrors the
   * commission retry orchestrator's eligibility rule (status is
   * pending_kyc OR failed). The UI renders the "Retirer" button only
   * when this flag is true so the button shape stays in sync with the
   * backend rules without embedding a status-to-eligibility table on
   * the client side. Optional for forward-compat with older API
   * responses that did not carry the field — fall back to deriving
   * from `status` if missing.
   */
  retire_eligible?: boolean
}

export type WalletOverview = {
  stripe_account_id: string
  charges_enabled: boolean
  payouts_enabled: boolean
  escrow_amount: number
  available_amount: number
  transferred_amount: number
  records: WalletRecord[] | null
  commissions: CommissionWallet
  commission_records: WalletCommissionRecord[] | null
}

export type PayoutResult = {
  status: string
  message: string
}

export function getWallet(): Promise<WalletOverview> {
  return apiClient<Get<"/api/v1/wallet"> & WalletOverview>("/api/v1/wallet")
}

export function requestPayout(): Promise<PayoutResult> {
  return apiClient<Post<"/api/v1/wallet/payout"> & PayoutResult>("/api/v1/wallet/payout", { method: "POST" })
}

/**
 * Re-issues the Stripe transfer for a single record stuck in
 * transfer_status="failed". Takes the payment record id — NOT the
 * proposal id — because a proposal can own N records (one per
 * milestone) and only the record id is unambiguous. The backend
 * enforces the same guards as the global payout (mission completed,
 * Stripe account present) and returns 409 when the row is no longer
 * retriable (e.g. someone else retried it).
 */
export function retryFailedTransfer(recordId: string): Promise<PayoutResult> {
  return apiClient<Post<"/api/v1/wallet/transfers/{record_id}/retry"> & PayoutResult>(
    `/api/v1/wallet/transfers/${recordId}/retry`,
    { method: "POST" },
  )
}

/**
 * Outcome of a commission retry call (D1+D2). The backend returns
 * different status codes for each branch:
 *   - 200 → status="paid" (transfer fired successfully)
 *   - 409 → "already_paid" / "not_retriable"
 *   - 422 → "kyc_required" with `onboarding_url`
 *   - 502 → "retry_failed" with `failure_reason`
 * The api-client wrapper surfaces non-2xx responses as ApiError, so
 * the calling hook can branch on `error.code` to drive the modal /
 * toast UX. The success body is the shape below.
 */
export type CommissionRetryResult = {
  status: string
  message: string
  stripe_account?: string
}

/**
 * Retry the Stripe transfer for an apporteur commission stuck in
 * pending_kyc or failed (D1+D2 "Retirer fallback"). The backend
 * verifies that the caller is the apporteur on the parent referral
 * (403 otherwise) and that the row is retriable (409 otherwise).
 * On a 422 the response carries an `onboarding_url` field — the
 * caller is expected to surface it in a "Termine ton KYC" modal so
 * the apporteur can finish onboarding before retrying.
 */
export function retryCommission(commissionId: string): Promise<CommissionRetryResult> {
  return apiClient<Post<"/api/v1/wallet/commissions/{id}/retry"> & CommissionRetryResult>(
    `/api/v1/wallet/commissions/${commissionId}/retry`,
    { method: "POST" },
  )
}

// ─── WALLET-UNIFY Run C — /wallet/summary + /wallet/withdraw ──────────────

/**
 * One leg of the unified wallet summary (missions side OR commissions
 * side). Same shape on both sides so the UI can iterate without
 * branching.
 */
export type WalletSummaryLeg = {
  total_cents: number
  available_cents: number
  escrowed_cents: number
  transmitted_cents: number
}

/**
 * One row in the unified transaction history. `type` is either
 * "mission" or "commission"; the UI picks the icon + tone from it.
 * `status` is a free-form backend string — the UI maps it to a
 * limited tone palette via `wallet-status-badge`.
 */
export type WalletSummaryTransaction = {
  type: "mission" | "commission"
  amount_cents: number
  currency: string
  status: string
  mission_title?: string
  occurred_at: string
  reference_id: string
}

/**
 * Envelope returned by GET /api/v1/wallet/summary. Mirrors the
 * `summaryResponse` struct in backend/internal/handler/wallet_summary.go.
 * Top-level totals are the sum of `breakdown.missions` and
 * `breakdown.commissions` — they are duplicated for the UI's hero
 * card convenience.
 */
export type WalletSummary = {
  currency: string
  total_cents: number
  available_cents: number
  escrowed_cents: number
  transmitted_cents: number
  breakdown: {
    missions: WalletSummaryLeg
    commissions: WalletSummaryLeg
  }
  recent_transactions: WalletSummaryTransaction[]
  next_cursor?: string
}

type WalletSummaryEnvelope = { data: WalletSummary }

/**
 * Fetches the unified wallet view. Optional cursor for the
 * `recent_transactions` pagination — the totals/breakdown are
 * stable across pages. limit defaults to 20 server-side (max 100).
 */
export async function getWalletSummary(
  cursor?: string,
): Promise<WalletSummary> {
  const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  const envelope = await apiClient<WalletSummaryEnvelope>(
    `/api/v1/wallet/summary${qs}`,
  )
  return envelope.data
}

/**
 * One error sub-entry on a 207 Multi-Status response. Identifies
 * which leg failed (missions vs commissions) plus a machine code +
 * human message. Surfaced in the partial-success modal.
 */
export type WithdrawLegError = {
  source: "missions" | "commissions"
  code: string
  message: string
}

/**
 * Body of the success envelope for POST /api/v1/wallet/withdraw.
 * `errors` is present on a 207 Multi-Status; empty on 200.
 */
export type WithdrawResult = {
  drained_cents: number
  missions_cents: number
  commissions_cents: number
  stripe_transfer_ids: string[]
  currency: string
  errors: WithdrawLegError[]
}

type WithdrawResultEnvelope = { data: WithdrawResult }

/**
 * Unified withdraw — drains BOTH missions and apporteur commissions
 * in a single Stripe orchestration. Pass no amount to drain
 * everything; pass an explicit amount in cents to cap the drain.
 *
 * Branches surfaced to the caller via the ApiError thrown by
 * apiClient on non-2xx responses:
 *   - 200  → full success — return { data: { drained_cents, … } }
 *   - 207  → partial success — apiClient sees 2xx, returns { data }
 *            but `errors[]` is populated for the failed leg
 *   - 422  → kyc_required — ApiError with `code === "kyc_required"`
 *            and `body.onboarding_url`
 *   - 403  → billing_profile_incomplete — ApiError with same code,
 *            `body.missing_fields` describes the gaps
 */
export async function withdrawWallet(
  amountCents?: number,
): Promise<WithdrawResult> {
  const envelope = await apiClient<WithdrawResultEnvelope>(
    "/api/v1/wallet/withdraw",
    {
      method: "POST",
      body: amountCents !== undefined ? { amount_cents: amountCents } : {},
    },
  )
  return envelope.data
}
