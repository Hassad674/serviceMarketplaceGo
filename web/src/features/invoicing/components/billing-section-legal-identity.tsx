"use client"

import { useFormContext } from "react-hook-form"
import { cn } from "@/shared/lib/utils"
import type { ProfileType } from "../types"
import type { BillingProfileFormValues } from "./billing-profile-form.schema"

// "Identité légale" + identifiants fiscaux:
//   - profile_type radio (individual / business)
//   - legal_name (always)
//   - trading_name + legal_form (business only)
//   - tax_id — labelled "Numéro SIRET" for FR (14 digits), generic
//     "Identifiant fiscal" otherwise
//
// Form state lives in the react-hook-form context owned by the
// orchestrator — this component only registers fields and reads the
// current `profile_type` + `country` to pick the right inputs.

const inputClasses = cn(
  "w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-xs",
  "focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
  "dark:border-slate-700 dark:bg-slate-900 dark:text-white",
)

export function BillingSectionLegalIdentity() {
  const {
    register,
    watch,
    formState: { errors },
  } = useFormContext<BillingProfileFormValues>()
  const profileType = watch("profile_type")
  const country = watch("country")
  const isFR = country === "FR"

  return (
    <>
      <Section title="Type de profil">
        <ProfileTypeRadio />
      </Section>

      <Section title="Identité légale">
        <Field
          label="Raison sociale ou nom légal"
          htmlFor="legal_name"
          error={errors.legal_name?.message}
        >
          <input
            id="legal_name"
            type="text"
            {...register("legal_name")}
            className={inputClasses}
            autoComplete="organization"
          />
        </Field>
        {profileType === "business" && (
          <>
            <Field label="Nom commercial (optionnel)" htmlFor="trading_name">
              <input
                id="trading_name"
                type="text"
                {...register("trading_name")}
                className={inputClasses}
              />
            </Field>
            <Field
              label="Forme juridique"
              htmlFor="legal_form"
              error={errors.legal_form?.message}
            >
              <input
                id="legal_form"
                type="text"
                {...register("legal_form")}
                className={inputClasses}
                placeholder="SAS, SARL, EURL, etc."
              />
            </Field>
          </>
        )}
      </Section>

      <Section title="Identifiants fiscaux">
        {isFR ? (
          <Field
            label="Numéro SIRET"
            htmlFor="tax_id"
            hint="14 chiffres, sans espace"
            error={errors.tax_id?.message}
          >
            <input
              id="tax_id"
              type="text"
              inputMode="numeric"
              {...register("tax_id")}
              className={inputClasses}
              maxLength={14}
            />
          </Field>
        ) : (
          <Field label="Identifiant fiscal" htmlFor="tax_id">
            <input
              id="tax_id"
              type="text"
              {...register("tax_id")}
              className={inputClasses}
            />
          </Field>
        )}
      </Section>
    </>
  )
}

function ProfileTypeRadio() {
  const { register, watch } = useFormContext<BillingProfileFormValues>()
  const value = watch("profile_type")
  // RHF controls the radio input — we must NOT set `checked` ourselves
  // because that breaks the controlled flow. The `selected` state is
  // only used to drive the highlight styles on the wrapping label.
  return (
    <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
      {(["individual", "business"] as ProfileType[]).map((option) => {
        const selected = value === option
        const label = option === "individual" ? "Particulier" : "Entreprise"
        return (
          <label
            key={option}
            className={cn(
              "flex cursor-pointer items-center gap-3 rounded-lg border px-4 py-2.5 text-sm transition-colors",
              selected
                ? "border-rose-500 bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300"
                : "border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800",
            )}
          >
            <input
              type="radio"
              value={option}
              {...register("profile_type")}
              className="h-4 w-4 accent-rose-500"
            />
            {label}
          </label>
        )
      })}
    </div>
  )
}

function Section({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle?: string
  children: React.ReactNode
}) {
  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900">
      <h2 className="text-sm font-semibold text-slate-900 dark:text-white">
        {title}
      </h2>
      {subtitle && (
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          {subtitle}
        </p>
      )}
      <div className="mt-4 space-y-3">{children}</div>
    </div>
  )
}

function Field({
  label,
  htmlFor,
  hint,
  error,
  children,
}: {
  label: string
  htmlFor: string
  hint?: string
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
      {hint && !error && (
        <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">
          {hint}
        </p>
      )}
    </div>
  )
}
