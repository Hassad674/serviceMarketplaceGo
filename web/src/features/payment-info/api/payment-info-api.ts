import { apiClient } from "@/shared/lib/api-client"
import type { PaymentInfoFormData } from "../types"

export type PaymentInfoResponse = {
  id: string
  user_id: string
  first_name: string
  last_name: string
  date_of_birth: string
  nationality: string
  address: string
  city: string
  postal_code: string
  is_business: boolean
  business_name: string
  business_address: string
  business_city: string
  business_postal_code: string
  business_country: string
  tax_id: string
  vat_number: string
  role_in_company: string
  phone: string
  activity_sector: string
  iban: string
  bic: string
  account_number: string
  routing_number: string
  account_holder: string
  bank_country: string
  stripe_account_id: string
  stripe_verified: boolean
  created_at: string
  updated_at: string
}

export type PaymentInfoStatusResponse = {
  complete: boolean
}

export async function getPaymentInfo(): Promise<PaymentInfoResponse | null> {
  return apiClient<PaymentInfoResponse | null>("/api/v1/payment-info")
}

export async function savePaymentInfo(data: PaymentInfoFormData, email?: string): Promise<PaymentInfoResponse> {
  return apiClient<PaymentInfoResponse>("/api/v1/payment-info", {
    method: "PUT",
    body: {
      email: email ?? "",
      first_name: data.firstName,
      last_name: data.lastName,
      date_of_birth: data.dateOfBirth,
      nationality: data.nationality,
      address: data.address,
      city: data.city,
      postal_code: data.postalCode,
      is_business: data.isBusiness,
      business_name: data.businessName,
      business_address: data.businessAddress,
      business_city: data.businessCity,
      business_postal_code: data.businessPostalCode,
      business_country: data.businessCountry,
      tax_id: data.taxId,
      vat_number: data.vatNumber,
      role_in_company: data.businessRole,
      phone: data.phone,
      activity_sector: data.activitySector,
      iban: data.iban,
      bic: data.bic,
      account_number: data.accountNumber,
      routing_number: data.routingNumber,
      account_holder: data.accountHolder,
      bank_country: data.bankCountry,
    },
  })
}

export async function getPaymentInfoStatus(): Promise<PaymentInfoStatusResponse> {
  return apiClient<PaymentInfoStatusResponse>("/api/v1/payment-info/status")
}
