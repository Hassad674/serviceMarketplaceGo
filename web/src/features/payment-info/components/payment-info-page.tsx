"use client"

import { useState, useCallback, useEffect } from "react"
import { AlertTriangle, CheckCircle, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { PersonalInfoSection } from "./personal-info-section"
import { BusinessInfoSection } from "./business-info-section"
import { BankAccountSection } from "./bank-account-section"
import { BusinessPersonsSection } from "./business-persons-section"
import { IdentityVerificationSection } from "./identity-verification-section"
import { StripeRequirementsBanner } from "./stripe-requirements-banner"
import { CountrySelector, detectBrowserCountry } from "./country-selector"
import { ExtraFieldsSection } from "./extra-fields-section"
import { isIbanCountry } from "./country-select"
import type { PaymentInfoFormData, BankAccountMode } from "../types"
import { INITIAL_FORM_DATA } from "../types"
import { useUser } from "@/shared/hooks/use-user"
import { usePaymentInfo, useSavePaymentInfo } from "../hooks/use-payment-info"
import { useCountryFields } from "../hooks/use-country-fields"
import type { PaymentInfoResponse } from "../api/payment-info-api"

function isFormValid(data: PaymentInfoFormData): boolean {
  const personalComplete =
    data.firstName.trim() !== "" &&
    data.lastName.trim() !== "" &&
    data.dateOfBirth !== "" &&
    data.nationality !== "" &&
    data.address.trim() !== "" &&
    data.city.trim() !== "" &&
    data.postalCode.trim() !== ""

  if (!personalComplete) return false

  if (data.isBusiness) {
    const businessComplete =
      data.businessRole !== "" &&
      data.businessName.trim() !== "" &&
      data.businessAddress.trim() !== "" &&
      data.businessCity.trim() !== "" &&
      data.businessPostalCode.trim() !== "" &&
      data.businessCountry !== "" &&
      data.taxId.trim() !== ""
    if (!businessComplete) return false
  }

  const bankComplete =
    data.accountHolder.trim() !== "" &&
    data.bankCountry !== "" &&
    (data.bankMode === "iban"
      ? data.iban.trim() !== ""
      : data.accountNumber.trim() !== "" && data.routingNumber.trim() !== "")

  return bankComplete
}

function responseToFormData(res: PaymentInfoResponse): PaymentInfoFormData {
  const hasIban = res.iban !== ""
  return {
    isBusiness: res.is_business,
    firstName: res.first_name,
    lastName: res.last_name,
    dateOfBirth: res.date_of_birth,
    nationality: res.nationality,
    address: res.address,
    city: res.city,
    postalCode: res.postal_code,
    businessRole: res.role_in_company as PaymentInfoFormData["businessRole"],
    businessName: res.business_name,
    businessAddress: res.business_address,
    businessCity: res.business_city,
    businessPostalCode: res.business_postal_code,
    businessCountry: res.business_country,
    taxId: res.tax_id,
    vatNumber: res.vat_number,
    phone: res.phone ?? "",
    activitySector: res.activity_sector || "8999",
    isSelfRepresentative: res.is_self_representative ?? true,
    isSelfDirector: res.is_self_director ?? true,
    noMajorOwners: res.no_major_owners ?? true,
    isSelfExecutive: res.is_self_executive ?? true,
    businessPersons: [],
    bankMode: hasIban ? "iban" : "local",
    iban: res.iban,
    bic: res.bic,
    accountNumber: res.account_number,
    routingNumber: res.routing_number,
    accountHolder: res.account_holder,
    bankCountry: res.bank_country,
    country: res.country ?? "",
    extraFields: res.extra_fields ?? {},
  }
}

export function PaymentInfoPage() {
  const t = useTranslations("paymentInfo")
  const [data, setData] = useState<PaymentInfoFormData>(INITIAL_FORM_DATA)
  const [saved, setSaved] = useState(false)
  const [initialized, setInitialized] = useState(false)

  const { data: user } = useUser()
  const { data: existing, isLoading } = usePaymentInfo()
  const saveMutation = useSavePaymentInfo()

  const businessType = data.isBusiness ? "company" : "individual"
  const { data: countryFields } = useCountryFields(data.country, businessType)

  // Extract extra fields that the country requires
  const extraFieldSpecs = (countryFields?.sections ?? [])
    .flatMap((s) => s.fields)
    .filter((f) => f.is_extra)

  const documentsRequired = countryFields?.documents_required ?? { individual: true, company: false }
  const personRoles = countryFields?.person_roles ?? undefined

  useEffect(() => {
    if (initialized || isLoading) return
    if (existing) {
      setData(responseToFormData(existing))
      setSaved(true)
    } else {
      // Pre-fill country from browser locale for new users
      const detected = detectBrowserCountry()
      if (detected) {
        setData((prev) => ({ ...prev, country: detected }))
      }
    }
    setInitialized(true)
  }, [existing, isLoading, initialized])

  const handleChange = useCallback(
    (field: keyof PaymentInfoFormData, value: string) => {
      setData((prev) => {
        const next = { ...prev, [field]: value }
        if (field === "nationality") {
          next.bankMode = isIbanCountry(value) ? "iban" : "local"
        }
        return next
      })
      setSaved(false)
    },
    [],
  )

  const handleToggleBusiness = useCallback(() => {
    setData((prev) => ({ ...prev, isBusiness: !prev.isBusiness }))
    setSaved(false)
  }, [])

  const handleChangeAny = useCallback(
    (field: keyof PaymentInfoFormData, value: unknown) => {
      setData((prev) => ({ ...prev, [field]: value }))
      setSaved(false)
    },
    [],
  )

  const handleCountryChange = useCallback((country: string) => {
    setData((prev) => ({ ...prev, country, extraFields: {} }))
    setSaved(false)
  }, [])

  const handleExtraFieldChange = useCallback((key: string, value: string) => {
    setData((prev) => ({
      ...prev,
      extraFields: { ...prev.extraFields, [key]: value },
    }))
    setSaved(false)
  }, [])

  const handleBankModeChange = useCallback((mode: BankAccountMode) => {
    setData((prev) => ({ ...prev, bankMode: mode }))
    setSaved(false)
  }, [])

  const handleSave = useCallback(() => {
    saveMutation.mutate({ data, email: user?.email }, {
      onSuccess: () => setSaved(true),
    })
  }, [data, user, saveMutation])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-6 w-6 animate-spin text-rose-500" />
      </div>
    )
  }

  const valid = isFormValid(data)

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
          {t("title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t("subtitle")}
        </p>
      </div>

      {/* Verification status banner */}
      {saved ? (
        <div className="flex items-center gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10">
          <CheckCircle className="h-5 w-5 shrink-0 text-emerald-600 dark:text-emerald-400" strokeWidth={1.5} />
          <p className="text-sm font-medium text-emerald-700 dark:text-emerald-300">
            {t("saved")}
          </p>
        </div>
      ) : (
        <div className="flex items-center gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-500/30 dark:bg-amber-500/10">
          <AlertTriangle className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" strokeWidth={1.5} />
          <p className="text-sm font-medium text-amber-700 dark:text-amber-300">
            {t("incomplete")}
          </p>
        </div>
      )}

      {/* API error banner */}
      {saveMutation.isError && (
        <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-500/30 dark:bg-red-500/10">
          <AlertTriangle className="h-5 w-5 shrink-0 text-red-600 dark:text-red-400" strokeWidth={1.5} />
          <p className="text-sm font-medium text-red-700 dark:text-red-300">
            {saveMutation.error instanceof Error
              ? saveMutation.error.message
              : t("saveError")}
          </p>
        </div>
      )}

      {/* Country selector - FIRST section */}
      <CountrySelector value={data.country} onChange={handleCountryChange} />

      {/* Business toggle */}
      <div className="flex items-center gap-3">
        <button
          type="button"
          role="switch"
          aria-checked={data.isBusiness}
          onClick={handleToggleBusiness}
          className={cn(
            "relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full transition-colors duration-200",
            data.isBusiness ? "bg-rose-500" : "bg-gray-300 dark:bg-gray-600",
          )}
        >
          <span
            className={cn(
              "inline-block h-4 w-4 rounded-full bg-white transition-transform duration-200 shadow-sm",
              data.isBusiness ? "translate-x-6" : "translate-x-1",
            )}
          />
        </button>
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {t("isBusiness")}
        </span>
      </div>

      {/* Stripe requirements banner */}
      {saved && existing?.stripe_account_id && <StripeRequirementsBanner />}

      {/* Sections */}
      <PersonalInfoSection data={data} onChange={handleChange} />

      {/* Country-specific extra fields */}
      {extraFieldSpecs.length > 0 && (
        <ExtraFieldsSection
          fields={extraFieldSpecs}
          values={data.extraFields}
          onChange={handleExtraFieldChange}
        />
      )}

      {data.isBusiness && (
        <>
          <BusinessInfoSection data={data} onChange={handleChange} />
          <BusinessPersonsSection data={data} onChange={handleChangeAny} requiredRoles={personRoles} />
        </>
      )}

      <BankAccountSection
        data={data}
        onChange={handleChange}
        onChangeBankMode={handleBankModeChange}
      />

      {/* Identity verification — only show if required for this country */}
      {documentsRequired.individual && <IdentityVerificationSection />}

      {/* Save button */}
      <button
        type="button"
        disabled={!valid || saveMutation.isPending}
        onClick={handleSave}
        className={cn(
          "w-full rounded-xl px-6 py-3 text-sm font-semibold text-white transition-all duration-200 sm:w-auto",
          valid && !saveMutation.isPending
            ? "gradient-primary hover:shadow-glow active:scale-[0.98]"
            : "cursor-not-allowed bg-gray-300 dark:bg-gray-700",
        )}
      >
        {saveMutation.isPending ? t("saving") : t("save")}
      </button>
    </div>
  )
}
