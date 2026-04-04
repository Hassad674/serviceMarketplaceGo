"use client"

import { useState, useCallback, useEffect } from "react"
import { AlertTriangle, CheckCircle, Loader2 } from "lucide-react"
import { useTranslations, useLocale } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { DynamicSection } from "./dynamic-section"
import { ActivitySectorSelect } from "./activity-sector-select"
import { BusinessPersonsSection } from "./business-persons-section"
import { IdentityVerificationSection } from "./identity-verification-section"
import { StripeRequirementsBanner } from "./stripe-requirements-banner"
import { CountrySelector } from "./country-selector"
import type { PaymentInfoFormData } from "../types"
import { INITIAL_FORM_DATA, BUSINESS_ROLES } from "../types"
import { useUser } from "@/shared/hooks/use-user"
import { usePaymentInfo, useSavePaymentInfo, useStripeRequirements } from "../hooks/use-payment-info"
import { useCountryFields } from "../hooks/use-country-fields"
import { useIdentityDocuments } from "../hooks/use-identity-documents"
import type { PaymentInfoResponse, FieldSection, RequirementsResponse } from "../api/payment-info-api"

/** Map locale to default country code. */
function localeToCountry(locale: string): string {
  const map: Record<string, string> = {
    fr: "FR", en: "US", de: "DE", es: "ES", it: "IT", pt: "PT", nl: "NL", ja: "JP",
  }
  return map[locale] ?? "FR"
}

export function PaymentInfoPage() {
  const t = useTranslations("paymentInfo")
  const locale = useLocale()
  const [data, setData] = useState<PaymentInfoFormData>({
    ...INITIAL_FORM_DATA,
    country: localeToCountry(locale),
  })
  const [saved, setSaved] = useState(false)
  const [stripeError, setStripeError] = useState<string | null>(null)
  const [initialized, setInitialized] = useState(false)

  const { data: user } = useUser()
  const { data: existing, isLoading } = usePaymentInfo()
  const saveMutation = useSavePaymentInfo()
  const { data: requirements } = useStripeRequirements()
  const { data: existingDocs } = useIdentityDocuments()

  const businessType = data.isBusiness ? "company" : "individual"
  const { data: countryFields } = useCountryFields(data.country, businessType)

  const documentsRequired = countryFields?.documents_required ?? { individual: false, company: false }
  const personRoles = countryFields?.person_roles ?? undefined

  // Filter sections: separate bank from entity sections
  const allSections = countryFields?.sections ?? []
  const entitySections = allSections.filter((s) => s.id !== "bank")
  const bankSection = allSections.find((s) => s.id === "bank")
  const hasDocumentUploadFields = allSections.some((s) =>
    s.fields.some((f) => f.type === "document_upload"),
  )

  // Build field errors and warnings from requirements
  const { fieldErrors, fieldWarnings, extraSections } = buildRequirementErrors(requirements, allSections, t)

  // Pre-fill document_upload fields with "uploaded" when docs already exist
  const docValues = buildDocumentValues(allSections, existingDocs ?? [])

  useEffect(() => {
    if (initialized || isLoading) return
    const userEmail = user?.email ?? ""
    if (existing) {
      const formData = responseToFormData(existing, locale)
      // Pre-fill email from auth user (not stored in payment entity)
      if (!formData.values["individual.email"] && userEmail) {
        formData.values["individual.email"] = userEmail
      }
      setData(formData)
      setSaved(true)
    } else {
      const detected = localeToCountry(locale)
      setData((prev) => ({
        ...prev,
        country: detected,
        values: {
          ...prev.values,
          "individual.email": userEmail,
          "individual.nationality": detected,
        },
      }))
    }
    setInitialized(true)
  }, [existing, isLoading, initialized, locale, user])

  const handleValueChange = useCallback((key: string, value: string) => {
    setData((prev) => ({ ...prev, values: { ...prev.values, [key]: value } }))
    setSaved(false)
  }, [])

  const handleToggleBusiness = useCallback(() => {
    setData((prev) => ({ ...prev, isBusiness: !prev.isBusiness, values: {} }))
    setSaved(false)
  }, [])

  const handleCountryChange = useCallback((country: string) => {
    setData((prev) => ({ ...prev, country, values: {}, extraFields: {} }))
    setSaved(false)
  }, [])

  const handleChangeAny = useCallback(
    (field: keyof PaymentInfoFormData, value: unknown) => {
      setData((prev) => ({ ...prev, [field]: value }))
      setSaved(false)
    },
    [],
  )

  const handleSave = useCallback(() => {
    const merged = valuesToFlatData(data, countryFields?.sections)
    saveMutation.mutate({ data: merged, email: user?.email }, {
      onSuccess: (response) => {
        setSaved(true)
        setStripeError(response.stripe_error ?? null)
      },
    })
  }, [data, user, saveMutation, countryFields])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-6 w-6 animate-spin text-rose-500" />
      </div>
    )
  }

  // Merge document upload status into values for display
  const mergedValues = { ...docValues, ...data.values }
  const valid = isFormValid(data, allSections)
  const hasRequirements = Object.keys(fieldErrors).length > 0 || Object.keys(fieldWarnings).length > 0

  return (
    <div className="space-y-6">
      <PageHeader t={t} />
      <StatusBanner saved={saved} saveMutation={saveMutation} t={t} />

      {/* Country selector */}
      <CountrySelector value={data.country} onChange={handleCountryChange} />

      {/* Business toggle */}
      <BusinessToggle checked={data.isBusiness} onToggle={handleToggleBusiness} t={t} />

      {/* Stripe requirements */}
      {saved && existing?.stripe_account_id && <StripeRequirementsBanner />}

      {/* Activity sector — always visible (not from country_specs) */}
      <ActivitySectorSelect
        value={data.values["activity_sector"] ?? data.activitySector}
        onChange={(v) => handleValueChange("activity_sector", v)}
      />

      {/* Dynamic entity sections */}
      {entitySections.map((section) => (
        <DynamicSection
          key={section.id}
          section={section}
          values={mergedValues}
          onChange={handleValueChange}
          fieldErrors={fieldErrors}
          fieldWarnings={fieldWarnings}
          documents={existingDocs ?? []}
          countryCode={data.country}
        />
      ))}

      {/* Business persons — only when country selected + business mode + roles beyond representative */}
      {data.country && data.isBusiness && hasPersonRoles(personRoles) && (
        <BusinessPersonsSection data={data} onChange={handleChangeAny} requiredRoles={personRoles} />
      )}

      {/* Bank section — rendered dynamically like entity sections */}
      {bankSection && (
        <DynamicSection
          section={bankSection}
          values={mergedValues}
          onChange={handleValueChange}
          fieldErrors={fieldErrors}
          fieldWarnings={fieldWarnings}
          documents={existingDocs ?? []}
          countryCode={data.country}
        />
      )}

      {/* Extra requirement sections not in the current form */}
      {extraSections.map((section) => (
        <DynamicSection
          key={`req-${section.id}`}
          section={section}
          values={mergedValues}
          onChange={handleValueChange}
          fieldErrors={fieldErrors}
          fieldWarnings={fieldWarnings}
          documents={existingDocs ?? []}
          countryCode={data.country}
        />
      ))}

      {/* Identity verification — only show when country is selected and docs are required but NOT inline */}
      {data.country && documentsRequired.individual && !hasDocumentUploadFields && (
        <IdentityVerificationSection />
      )}

      {/* Stripe error just above save */}
      {stripeError && (
        <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-500/30 dark:bg-red-500/10">
          <AlertTriangle className="h-5 w-5 shrink-0 text-red-600 dark:text-red-400" strokeWidth={1.5} />
          <div>
            <p className="text-sm font-semibold text-red-700 dark:text-red-300">{t("stripeErrorTitle")}</p>
            <p className="mt-0.5 text-xs text-red-600 dark:text-red-400">{stripeError}</p>
          </div>
        </div>
      )}

      {/* Save button */}
      <button
        type="button"
        disabled={!valid || saveMutation.isPending || (saved && !hasRequirements)}
        onClick={handleSave}
        className={cn(
          "w-full rounded-xl px-6 py-3 text-sm font-semibold text-white transition-all duration-200 sm:w-auto",
          valid && !saveMutation.isPending && !(saved && !hasRequirements)
            ? "gradient-primary hover:shadow-glow active:scale-[0.98]"
            : "cursor-not-allowed bg-gray-300 dark:bg-gray-700",
        )}
      >
        {saveMutation.isPending ? t("saving") : t("save")}
      </button>
    </div>
  )
}

// --- Sub-components to keep the main function under 50 lines ---

function PageHeader({ t }: { t: (key: string) => string }) {
  return (
    <div>
      <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
        {t("title")}
      </h1>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {t("subtitle")}
      </p>
    </div>
  )
}

function StatusBanner({ saved, saveMutation, t }: {
  saved: boolean
  saveMutation: { isError: boolean; error: Error | null }
  t: (key: string) => string
}) {
  return (
    <>
      {saved ? (
        <div className="flex items-center gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10">
          <CheckCircle className="h-5 w-5 shrink-0 text-emerald-600 dark:text-emerald-400" strokeWidth={1.5} />
          <p className="text-sm font-medium text-emerald-700 dark:text-emerald-300">{t("saved")}</p>
        </div>
      ) : (
        <div className="flex items-center gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-500/30 dark:bg-amber-500/10">
          <AlertTriangle className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" strokeWidth={1.5} />
          <p className="text-sm font-medium text-amber-700 dark:text-amber-300">{t("incomplete")}</p>
        </div>
      )}
      {saveMutation.isError && (
        <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-500/30 dark:bg-red-500/10">
          <AlertTriangle className="h-5 w-5 shrink-0 text-red-600 dark:text-red-400" strokeWidth={1.5} />
          <p className="text-sm font-medium text-red-700 dark:text-red-300">
            {saveMutation.error instanceof Error ? saveMutation.error.message : t("saveError")}
          </p>
        </div>
      )}
    </>
  )
}

function BusinessToggle({ checked, onToggle, t }: {
  checked: boolean; onToggle: () => void; t: (key: string) => string
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-3">
        <button
          type="button"
          role="switch"
          aria-checked={checked}
          onClick={onToggle}
          className={cn(
            "relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full transition-colors duration-200",
            checked ? "bg-rose-500" : "bg-gray-300 dark:bg-gray-600",
          )}
        >
          <span
            className={cn(
              "inline-block h-4 w-4 rounded-full bg-white transition-transform duration-200 shadow-sm",
              checked ? "translate-x-6" : "translate-x-1",
            )}
          />
        </button>
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("isBusiness")}</span>
      </div>
      <p className="text-xs text-slate-500 dark:text-slate-400 ml-14">{t("isBusinessDesc")}</p>
    </div>
  )
}

// --- Data mapping helpers ---

/** Returns true when personRoles contains at least one role beyond "representative". */
function hasPersonRoles(roles?: string[]): boolean {
  if (!roles || roles.length === 0) return false
  return roles.some((r) => r !== "representative")
}

function isFormValid(data: PaymentInfoFormData, sections: FieldSection[]): boolean {
  for (const section of sections) {
    for (const field of section.fields) {
      // Skip document upload fields — they are validated separately
      if (field.type === "document_upload") continue
      if (field.required && !(data.values[field.key] ?? "").trim()) {
        return false
      }
    }
  }
  return true
}

/** Convert API response to form data with path-keyed values. */
function responseToFormData(res: PaymentInfoResponse, locale: string): PaymentInfoFormData {
  const isBusiness = res.is_business
  const prefix = isBusiness ? "representative" : "individual"

  const values: Record<string, string> = {}
  // Map flat response fields to path-keyed values
  if (res.first_name) values[`${prefix}.first_name`] = res.first_name
  if (res.last_name) values[`${prefix}.last_name`] = res.last_name
  if (res.date_of_birth) values[`${prefix}.dob`] = res.date_of_birth
  if (res.nationality) values[`${prefix}.nationality`] = res.nationality
  if (res.address) values[`${prefix}.address.line1`] = res.address
  if (res.city) values[`${prefix}.address.city`] = res.city
  if (res.postal_code) values[`${prefix}.address.postal_code`] = res.postal_code
  if (res.phone) values[`${prefix}.phone`] = res.phone

  if (isBusiness) {
    if (res.business_name) values["company.name"] = res.business_name
    if (res.business_address) values["company.address.line1"] = res.business_address
    if (res.business_city) values["company.address.city"] = res.business_city
    if (res.business_postal_code) values["company.address.postal_code"] = res.business_postal_code
    if (res.business_country) values["company.address.country"] = res.business_country
    if (res.tax_id) values["company.tax_id"] = res.tax_id
  }

  // Bank
  if (res.iban) values["bank.iban"] = res.iban
  if (res.bic) values["bank.bic"] = res.bic
  if (res.account_number) values["bank.account_number"] = res.account_number
  if (res.routing_number) values["bank.routing_number"] = res.routing_number
  if (res.account_holder) values["bank.account_holder"] = res.account_holder
  if (res.bank_country) values["bank.bank_country"] = res.bank_country

  // Activity sector and business role
  values["activity_sector"] = res.activity_sector || "8999"
  if (res.role_in_company) values["business_role"] = res.role_in_company

  // Extra fields — map them with their original key
  if (res.extra_fields) {
    for (const [key, val] of Object.entries(res.extra_fields)) {
      // Extra fields are stored with entity-prefixed keys
      values[key] = val
    }
  }

  const hasIban = res.iban !== ""
  return {
    ...INITIAL_FORM_DATA,
    isBusiness,
    country: res.country ?? localeToCountryFallback(locale),
    values,
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
    extraFields: res.extra_fields ?? {},
  }
}

function localeToCountryFallback(locale: string): string {
  const map: Record<string, string> = { fr: "FR", en: "US", de: "DE", es: "ES" }
  return map[locale] ?? "FR"
}

/** Build field errors (currently_due/past_due) and warnings (eventually_due) from requirements. */
function buildRequirementErrors(
  requirements: RequirementsResponse | undefined,
  formSections: FieldSection[],
  t: (key: string) => string,
): { fieldErrors: Record<string, string>; fieldWarnings: Record<string, string>; extraSections: FieldSection[] } {
  if (!requirements?.has_requirements || !requirements.sections?.length) {
    return { fieldErrors: {}, fieldWarnings: {}, extraSections: [] }
  }

  const formFieldKeys = new Set<string>()
  const docUploadKeys: string[] = []
  for (const section of formSections) {
    for (const field of section.fields) {
      formFieldKeys.add(field.key)
      if (field.type === "document_upload") docUploadKeys.push(field.key)
    }
  }

  const fieldErrors: Record<string, string> = {}
  const fieldWarnings: Record<string, string> = {}
  const extraSections: FieldSection[] = []

  for (const reqSection of requirements.sections) {
    const extraFields: typeof reqSection.fields = []
    for (const field of reqSection.fields) {
      const urgency = field.urgency ?? "currently_due"
      const isWarning = urgency === "eventually_due"
      const targetMap = isWarning ? fieldWarnings : fieldErrors
      const msg = isWarning ? t("fieldEventuallyDue") : t("fieldMissing")

      if (formFieldKeys.has(field.key)) {
        targetMap[field.key] = msg
      } else {
        const matchedDoc = docUploadKeys.find((dk) => field.key.startsWith(dk))
        if (matchedDoc) {
          targetMap[matchedDoc] = msg
        } else {
          extraFields.push(field)
        }
        targetMap[field.key] = msg
      }
    }
    if (extraFields.length > 0) {
      extraSections.push({ ...reqSection, fields: extraFields })
    }
  }

  return { fieldErrors, fieldWarnings, extraSections }
}

/** Mark document_upload fields as "uploaded" when a matching document exists. */
function buildDocumentValues(
  sections: FieldSection[],
  docs: { category: string; document_type: string }[],
): Record<string, string> {
  if (!docs.length) return {}
  const vals: Record<string, string> = {}
  for (const section of sections) {
    for (const field of section.fields) {
      if (field.type !== "document_upload") continue
      // Match by category + document_type derived from the field path
      const category = field.path.startsWith("company") || field.path.startsWith("documents")
        ? "company" : "identity"
      const docType = deriveDocType(field.path)
      const hasMatch = docs.some((d) => d.category === category && d.document_type === docType)
      if (hasMatch) vals[field.key] = "uploaded"
    }
  }
  return vals
}

function deriveDocType(path: string): string {
  if (path.includes("proof_of_liveness")) return "proof_of_liveness"
  if (path.includes("additional_document")) return "additional_document"
  if (path.includes("company_authorization")) return "company_authorization"
  if (path.includes("passport")) return "passport"
  if (path.includes("bank_account_ownership")) return "bank_account_ownership"
  return "document"
}

/** Convert path-keyed values back to the flat save format. */
function valuesToFlatData(
  data: PaymentInfoFormData, sections?: FieldSection[],
): PaymentInfoFormData {
  const v = data.values
  const isBusiness = data.isBusiness
  const prefix = isBusiness ? "representative" : "individual"

  const extraFields: Record<string, string> = { ...data.extraFields }
  // Collect extra fields from values
  if (sections) {
    for (const section of sections) {
      for (const field of section.fields) {
        if (field.is_extra && v[field.key]) {
          extraFields[field.key] = v[field.key]
        }
      }
    }
  }

  return {
    ...data,
    firstName: v[`${prefix}.first_name`] ?? data.firstName,
    lastName: v[`${prefix}.last_name`] ?? data.lastName,
    dateOfBirth: v[`${prefix}.dob`] ?? data.dateOfBirth,
    nationality: v[`${prefix}.nationality`] ?? (data.nationality || data.country),
    address: v[`${prefix}.address.line1`] ?? data.address,
    city: v[`${prefix}.address.city`] ?? data.city,
    postalCode: v[`${prefix}.address.postal_code`] ?? data.postalCode,
    phone: v[`${prefix}.phone`] ?? data.phone,
    businessName: v["company.name"] ?? data.businessName,
    businessAddress: v["company.address.line1"] ?? data.businessAddress,
    businessCity: v["company.address.city"] ?? data.businessCity,
    businessPostalCode: v["company.address.postal_code"] ?? data.businessPostalCode,
    businessCountry: v["company.address.country"] ?? data.businessCountry,
    taxId: v["company.tax_id"] ?? data.taxId,
    activitySector: v["activity_sector"] ?? data.activitySector,
    businessRole: (v["business_role"] ?? data.businessRole) as PaymentInfoFormData["businessRole"],
    iban: v["bank.iban"] ?? data.iban,
    bic: v["bank.bic"] ?? data.bic,
    accountNumber: v["bank.account_number"] ?? data.accountNumber,
    routingNumber: v["bank.routing_number"] ?? data.routingNumber,
    accountHolder: v["bank.account_holder"] ?? data.accountHolder,
    bankCountry: v["bank.bank_country"] ?? data.bankCountry,
    extraFields,
  }
}
