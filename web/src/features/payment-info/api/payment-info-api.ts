import { apiClient } from "@/shared/lib/api-client"

export type PaymentInfoResponse = {
  id: string
  user_id: string
  stripe_account_id: string
  stripe_verified: boolean
  charges_enabled: boolean
  payouts_enabled: boolean
  stripe_business_type: string
  stripe_country: string
  stripe_display_name: string
  created_at: string
  updated_at: string
}

export type PaymentInfoStatusResponse = {
  complete: boolean
}

export async function getPaymentInfo(): Promise<PaymentInfoResponse | null> {
  return apiClient<PaymentInfoResponse | null>("/api/v1/payment-info")
}

export async function getPaymentInfoStatus(): Promise<PaymentInfoStatusResponse> {
  return apiClient<PaymentInfoStatusResponse>("/api/v1/payment-info/status")
}

export type AccountSessionResponse = {
  client_secret: string
  stripe_account_id: string
}

export async function createAccountSession(email: string): Promise<AccountSessionResponse> {
  return apiClient<AccountSessionResponse>("/api/v1/payment-info/account-session", {
    method: "POST",
    body: { email },
  })
}
