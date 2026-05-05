"use client"

import { CheckCircle2, Loader2, XCircle } from "lucide-react"
import { useFormContext } from "react-hook-form"
import { cn, formatDate } from "@/shared/lib/utils"
import { isEUCountry } from "./eu-countries"
import type { BillingProfileFormValues } from "./billing-profile-form.schema"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
// "Identifiants fiscaux" — VAT applicability section.
// Currently only handles the intracom VAT input + VIES validate
// button (EU non-FR). The brief reserves space here for a future
// "régime fiscal" toggle that the backend does not yet expose, so
// the section is only rendered when the country is EU.
//
// Soleil v2: ivoire surface card (rounded-2xl), Fraunces section
// title, sable border, corail focus + corail-soft validate pill.
//
// VAT validation runs through a dedicated mutation (`onValidateVat`)
// that the parent owns — this component only reports the outcome.

const inputClasses = cn(
  "w-full rounded-xl border border-border bg-surface px-3 py-2 text-sm text-foreground",
  "transition-colors duration-200 placeholder:text-subtle-foreground",
  "focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary/15",
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
          <Input
            id="vat_number"
            type="text"
            {...register("vat_number")}
            className={inputClasses}
            placeholder="FR12345678901"
            autoComplete="off"
          />
        </Field>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="ghost" size="auto"
            type="button"
            onClick={onValidate}
            disabled={!canValidate || isValidating}
            className={cn(
              "inline-flex items-center gap-2 rounded-full border border-border-strong bg-surface px-4 py-1.5 text-xs font-semibold text-foreground transition-colors",
              "hover:bg-primary-soft/40 disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {isValidating ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
            ) : (
              <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
            )}
            Valider mon n° TVA
          </Button>
          {validatedAt && !validateError && (
            <span className="inline-flex items-center gap-1 text-xs font-medium text-success">
              <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
              Validé le {formatDate(validatedAt)}
            </span>
          )}
          {validateError && (
            <span className="inline-flex items-center gap-1 text-xs font-medium text-destructive">
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
    <div className="rounded-2xl border border-border bg-surface p-6">
      <h2 className="font-serif text-[18px] font-semibold tracking-[-0.01em] text-foreground">
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
        className="mb-1.5 block text-xs font-medium text-foreground"
      >
        {label}
      </label>
      {children}
      {error && (
        <p
          role="alert"
          className="mt-1 text-[11px] text-destructive"
        >
          {error}
        </p>
      )}
    </div>
  )
}
