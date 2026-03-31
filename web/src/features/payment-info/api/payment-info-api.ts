import { apiClient } from "@/shared/lib/api-client"
import type { PaymentInfoFormData } from "../types"

// --- Country fields types ---

export type FieldSpec = {
  path: string
  key: string
  type: string
  label_key: string
  required: boolean
  is_extra: boolean
  placeholder?: string
}

export type FieldSection = {
  id: string
  fields: FieldSpec[]
}

export type CountryFieldsResponse = {
  country: string
  business_type: string
  sections: FieldSection[]
  documents_required: {
    individual: boolean
    company: boolean
  }
  person_roles: string[] | null
}

// --- Payment info types ---

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
  is_self_representative: boolean
  is_self_director: boolean
  no_major_owners: boolean
  is_self_executive: boolean
  iban: string
  bic: string
  account_number: string
  routing_number: string
  account_holder: string
  bank_country: string
  stripe_account_id: string
  stripe_verified: boolean
  country: string
  extra_fields: Record<string, string>
  created_at: string
  updated_at: string
}

export type PaymentInfoStatusResponse = {
  complete: boolean
}

export async function getCountryFields(country: string, businessType: string): Promise<CountryFieldsResponse> {
  return apiClient<CountryFieldsResponse>(
    `/api/v1/payment-info/country-fields?country=${country}&business_type=${businessType}`,
  )
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
      is_self_representative: data.isSelfRepresentative,
      is_self_director: data.isSelfDirector,
      no_major_owners: data.noMajorOwners,
      is_self_executive: data.isSelfExecutive,
      business_persons: data.businessPersons.map((p) => ({
        role: p.role,
        first_name: p.firstName,
        last_name: p.lastName,
        date_of_birth: p.dateOfBirth,
        email: p.email,
        phone: p.phone,
        address: p.address,
        city: p.city,
        postal_code: p.postalCode,
        title: p.title,
      })),
      iban: data.iban,
      bic: data.bic,
      account_number: data.accountNumber,
      routing_number: data.routingNumber,
      account_holder: data.accountHolder,
      bank_country: data.bankCountry,
      country: data.country,
      extra_fields: data.extraFields,
    },
  })
}

export type RequirementsInfo = {
  has_requirements: boolean
  currently_due: string[]
  labels: { code: string; label: string }[]
}

export function getRequirements(lang: string): Promise<RequirementsInfo> {
  return apiClient<RequirementsInfo>(`/api/v1/payment-info/requirements?lang=${lang}`)
}

export function createAccountLink(): Promise<{ url: string }> {
  return apiClient<{ url: string }>("/api/v1/payment-info/account-link", { method: "POST" })
}

export async function getPaymentInfoStatus(): Promise<PaymentInfoStatusResponse> {
  return apiClient<PaymentInfoStatusResponse>("/api/v1/payment-info/status")
}
