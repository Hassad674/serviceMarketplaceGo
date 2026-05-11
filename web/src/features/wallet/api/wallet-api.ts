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
