"use client"

import { useEffect, useId, useState } from "react"
import { useTranslations } from "next-intl"
import { Loader2 } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { useBillingProfile } from "@/shared/hooks/billing-profile/use-billing-profile"
import { updateBillingProfile } from "@/shared/lib/billing-profile/billing-profile-api"
import type { BillingProfile } from "@/shared/types/billing-profile"

// PaymentBillingIdentitySection is the inline B2B legal-identity capture
// the proposal payment flow renders ABOVE the Stripe Payment Element.
//
// Stripe's Payment Element collects card + billing address natively, but
// French B2B receipts also require legal name (raison sociale), SIRET,
// and an optional VAT number. Those three live OUTSIDE Stripe's
// billing_details model, so we capture them in a small react-hook-free
// form and persist them to /api/v1/me/billing-profile BEFORE calling
// stripe.confirmPayment(). The webhook hydration step then completes
// the address fields after Stripe confirms the charge.
//
// Why no react-hook-form here:
// - The form has 3 fields; useState is lighter and avoids the controlled-
//   uncontrolled toggle that RHF+zod would impose on partially-typed
//   inputs.
// - Validation rules are trivial (SIRET regex, VAT regex) and are
//   enforced server-side by the existing billing-profile schema; the
//   client-side validation here is a UX hint, not a security gate.
// - Keeping the component sub-200 lines means the test surface stays
//   minimal and a future migration to RHF is one component refactor
//   without affecting the rest of the payment page.
//
// The component owns its own submit state. The parent owns the
// outer flow: it calls the imperative `onSavedRef` callback once the
// form has saved, then proceeds with stripe.confirmPayment. The
// imperative pattern (vs. a Promise return) keeps the parent's submit
// handler readable as a linear sequence: "save identity → confirm
// payment" rather than nested then chains.

const SIRET_REGEX = /^\d{14}$/
const VAT_REGEX = /^[A-Z]{2}[A-Z0-9]{2,12}$/

export type PaymentBillingIdentityValues = {
  legal_name: string
  tax_id: string
  vat_number: string
}

export type PaymentBillingIdentitySectionProps = {
  /**
   * Called by the parent BEFORE stripe.confirmPayment() so the form's
   * current values are persisted on the server. Returns the saved
   * profile snapshot on success; throws on network/validation failure
   * so the parent can surface the error inline.
   */
  onSubmit?: (values: PaymentBillingIdentityValues) => Promise<void>
  /**
   * Optional callback fired with every keystroke change. Lets the
   * parent reflect the current values into a submit-time payload
   * without coupling to the form's internal state.
   */
  onChange?: (values: PaymentBillingIdentityValues) => void
}

/**
 * Reads the current billing profile via the shared TanStack Query hook
 * and exposes a small <fieldset> the user can fill alongside the
 * Stripe Payment Element. Pre-fills the inputs from the profile so the
 * second payment never re-asks for already-saved data.
 */
export function PaymentBillingIdentitySection({
  onChange,
}: PaymentBillingIdentitySectionProps) {
  const t = useTranslations("proposal.inlineBilling")
  const { data: snapshot, isLoading } = useBillingProfile()

  const [legalName, setLegalName] = useState("")
  const [taxID, setTaxID] = useState("")
  const [vatNumber, setVatNumber] = useState("")

  // Pre-fill inputs once the snapshot lands. The dependency array is
  // intentionally `snapshot?.profile` and not the inner fields so the
  // user's in-progress edits are not clobbered on a refetch.
  useEffect(() => {
    if (!snapshot?.profile) return
    seedFromProfile(snapshot.profile, setLegalName, setTaxID, setVatNumber)
  }, [snapshot?.profile])

  // Surface every change to the parent so it can chain a save before
  // stripe.confirmPayment(). Using the props inside the same effect
  // would re-fire on every parent render — split into a tiny effect
  // keyed on the field values only.
  useEffect(() => {
    if (!onChange) return
    onChange({ legal_name: legalName, tax_id: taxID, vat_number: vatNumber })
  }, [legalName, taxID, vatNumber, onChange])

  const legalNameId = useId()
  const taxIDId = useId()
  const vatID = useId()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center rounded-2xl border border-border bg-background p-4">
        <Loader2 className="h-4 w-4 animate-spin text-primary" aria-hidden="true" />
      </div>
    )
  }

  const country = snapshot?.profile.country?.toUpperCase() ?? ""
  const isFrench = country === "" || country === "FR"
  const showVATError = vatNumber.trim() !== "" && !VAT_REGEX.test(vatNumber.toUpperCase().replace(/\s+/g, ""))
  const showSIRETError = isFrench && taxID.trim() !== "" && !SIRET_REGEX.test(taxID.replace(/\s+/g, ""))

  return (
    <fieldset className="space-y-3 rounded-2xl border border-border bg-background p-4">
      <legend className="px-1 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
        {t("legend")}
      </legend>
      <p className="text-[12.5px] leading-relaxed text-muted-foreground">
        {t("subtitle")}
      </p>

      <div className="space-y-1">
        <label
          htmlFor={legalNameId}
          className="block text-[12px] font-medium text-foreground"
        >
          {t("legalNameLabel")}
        </label>
        <input
          id={legalNameId}
          type="text"
          autoComplete="organization"
          value={legalName}
          onChange={(e) => setLegalName(e.target.value)}
          placeholder={t("legalNamePlaceholder")}
          className={cn(
            "w-full rounded-lg border border-border bg-card px-3 py-2 text-[13.5px]",
            "text-foreground placeholder:text-muted-foreground",
            "focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary",
          )}
        />
      </div>

      {isFrench && (
        <div className="space-y-1">
          <label
            htmlFor={taxIDId}
            className="block text-[12px] font-medium text-foreground"
          >
            {t("siretLabel")}
          </label>
          <input
            id={taxIDId}
            type="text"
            inputMode="numeric"
            value={taxID}
            onChange={(e) => setTaxID(e.target.value.replace(/\D/g, "").slice(0, 14))}
            placeholder="14 chiffres"
            aria-invalid={showSIRETError}
            aria-describedby={showSIRETError ? `${taxIDId}-error` : undefined}
            className={cn(
              "w-full rounded-lg border border-border bg-card px-3 py-2 text-[13.5px] font-mono",
              "text-foreground placeholder:text-muted-foreground",
              "focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary",
              showSIRETError && "border-destructive",
            )}
          />
          {showSIRETError && (
            <p id={`${taxIDId}-error`} className="text-[11.5px] text-destructive">
              {t("siretError")}
            </p>
          )}
        </div>
      )}

      <div className="space-y-1">
        <label
          htmlFor={vatID}
          className="block text-[12px] font-medium text-foreground"
        >
          {t("vatLabel")}
          <span className="ml-1 font-normal text-muted-foreground">
            ({t("vatOptional")})
          </span>
        </label>
        <input
          id={vatID}
          type="text"
          autoComplete="off"
          value={vatNumber}
          onChange={(e) => setVatNumber(e.target.value.toUpperCase())}
          placeholder="FR12345678901"
          aria-invalid={showVATError}
          aria-describedby={showVATError ? `${vatID}-error` : undefined}
          className={cn(
            "w-full rounded-lg border border-border bg-card px-3 py-2 text-[13.5px] font-mono",
            "text-foreground placeholder:text-muted-foreground",
            "focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary",
            showVATError && "border-destructive",
          )}
        />
        {showVATError && (
          <p id={`${vatID}-error`} className="text-[11.5px] text-destructive">
            {t("vatError")}
          </p>
        )}
      </div>
    </fieldset>
  )
}

/**
 * Persist the SIRET/VAT/legal-name to /api/v1/me/billing-profile via
 * the existing PUT endpoint. Merges with the current snapshot so the
 * other already-filled columns (trading_name, address, etc.) survive
 * the partial save. Returns void; throws on network failure so the
 * caller can surface the error inline.
 */
export async function persistInlineBillingIdentity(
  values: PaymentBillingIdentityValues,
  current: BillingProfile | undefined,
): Promise<void> {
  // No values to save → no-op. Triggers when the user leaves the
  // legal_name + tax_id + vat_number all empty (e.g. they want to
  // submit using whatever Stripe collects).
  const hasAny =
    values.legal_name.trim() !== "" ||
    values.tax_id.trim() !== "" ||
    values.vat_number.trim() !== ""
  if (!hasAny) return

  await updateBillingProfile({
    profile_type: current?.profile_type ?? "business",
    legal_name: values.legal_name.trim() || (current?.legal_name ?? ""),
    trading_name: current?.trading_name ?? "",
    legal_form: current?.legal_form ?? "",
    tax_id: values.tax_id.trim() || (current?.tax_id ?? ""),
    vat_number: values.vat_number.trim() || (current?.vat_number ?? ""),
    address_line1: current?.address_line1 ?? "",
    address_line2: current?.address_line2 ?? "",
    postal_code: current?.postal_code ?? "",
    city: current?.city ?? "",
    country: current?.country ?? "FR",
    invoicing_email: current?.invoicing_email ?? "",
  })
}

/**
 * Helper extracted for testability — pre-fills the controlled inputs
 * from a fetched billing profile when the user lands on the page.
 */
function seedFromProfile(
  profile: BillingProfile,
  setLegalName: (s: string) => void,
  setTaxID: (s: string) => void,
  setVAT: (s: string) => void,
) {
  if (profile.legal_name) setLegalName(profile.legal_name)
  if (profile.tax_id) setTaxID(profile.tax_id)
  if (profile.vat_number) setVAT(profile.vat_number)
}
