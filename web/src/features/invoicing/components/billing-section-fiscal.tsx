"use client"

import { CheckCircle2, Loader2, XCircle } from "lucide-react"
import { useFormContext } from "react-hook-form"
import { cn, formatDate } from "@/shared/lib/utils"
import { isEUCountry } from "./eu-countries"
import type { BillingProfileFormValues } from "./billing-profile-form.schema"

// "Identifiants fiscaux" — VAT applicability section.
// Currently only handles the intracom VAT input + VIES validate
// button (EU non-FR). The brief reserves space here for a future
// "régime fiscal" toggle that the backend does not yet expose, so
// the section is only rendered when the country is EU.
//
// VAT validation runs through a dedicated mutation (`onValidateVat`)
// that the parent owns — this component only reports the outcome.

const inputClasses = cn(
  "w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-xs",
  "focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
  "dark:border-slate-700 dark:bg-slate-900 dark:text-white",
)

export interface BillingSectionFiscalProps {
  validatedAt: string | null
  isValidating: boolean
  validateError: Error | null
  onValidate: () => void
}

export function BillingSectionFiscal({
  validatedAt,
  isValidating,
  validateError,
  onValidate,
}: BillingSectionFiscalProps) {
  const {
    register,
    watch,
    formState: { errors },
  } = useFormContext<BillingProfileFormValues>()
  const country = watch("country")
  const vatNumber = watch("vat_number")

  // Section is hidden for non-EU countries — VAT does not apply.
  // FR is also EU but uses SIRET in the legal-identity section, so
  // the VAT input is shown only for EU non-FR.
  if (!isEUCountry(country)) return null

  const canValidate = vatNumber.trim().length > 0

  return (
    <Section title="Identifiants fiscaux">
      <div className="space-y-2">
        <Field
          label="Numéro de TVA intracommunautaire"
          htmlFor="vat_number"
          error={errors.vat_number?.message}
        >
          <input
            id="vat_number"
            type="text"
            {...register("vat_number")}
            className={inputClasses}
            placeholder="FR12345678901"
            autoComplete="off"
          />
        </Field>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={onValidate}
            disabled={!canValidate || isValidating}
            className={cn(
              "inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition-colors",
              "hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed",
              "dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700",
            )}
          >
            {isValidating ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
            ) : (
              <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
            )}
            Valider mon n° TVA
          </button>
          {validatedAt && !validateError && (
            <span className="inline-flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
              <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
              Validé le {formatDate(validatedAt)}
            </span>
          )}
          {validateError && (
            <span className="inline-flex items-center gap-1 text-xs text-red-600 dark:text-red-400">
              <XCircle className="h-3.5 w-3.5" aria-hidden="true" />
              Numéro non reconnu par VIES
            </span>
          )}
        </div>
      </div>
    </Section>
  )
}

function Section({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900">
      <h2 className="text-sm font-semibold text-slate-900 dark:text-white">
        {title}
      </h2>
      <div className="mt-4 space-y-3">{children}</div>
    </div>
  )
}

function Field({
  label,
  htmlFor,
  error,
  children,
}: {
  label: string
  htmlFor: string
  error?: string
  children: React.ReactNode
}) {
  return (
    <div>
      <label
        htmlFor={htmlFor}
        className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300"
      >
        {label}
      </label>
      {children}
      {error && (
        <p
          role="alert"
          className="mt-1 text-[11px] text-red-600 dark:text-red-400"
        >
          {error}
        </p>
      )}
    </div>
  )
}
