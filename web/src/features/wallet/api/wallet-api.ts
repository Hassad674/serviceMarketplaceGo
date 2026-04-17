import { apiClient } from "@/shared/lib/api-client"

export type WalletRecord = {
  proposal_id: string
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
  return apiClient<WalletOverview>("/api/v1/wallet")
}

export function requestPayout(): Promise<PayoutResult> {
  return apiClient<PayoutResult>("/api/v1/wallet/payout", { method: "POST" })
}
