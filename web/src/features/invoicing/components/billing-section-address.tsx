"use client"

import { useFormContext } from "react-hook-form"
import { cn } from "@/shared/lib/utils"
import {
  REGION_LABELS,
  STRIPE_CONNECT_COUNTRIES,
  type SupportedCountry,
} from "@/shared/lib/stripe-countries"
import { AddressAutocomplete } from "./address-autocomplete"
import type { BillingProfileFormValues } from "./billing-profile-form.schema"

import { Input } from "@/shared/components/ui/input"
import { Select } from "@/shared/components/ui/select"
// Address section + country dropdown. The country is bundled here
// because the address autocomplete provider is country-scoped (the
// user picks the country first, then the autocomplete narrows by
// that ISO code).
//
// Soleil v2 styling: ivoire surface card (rounded-2xl), Fraunces
// section title, sable border, corail focus ring on inputs.

const inputClasses = cn(
  "w-full rounded-xl border border-border bg-surface px-3 py-2 text-sm text-foreground",
  "transition-colors duration-200 placeholder:text-subtle-foreground",
  "focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary/15",
)

export function BillingSectionAddress() {
  const {
    register,
    watch,
    setValue,
    formState: { errors },
  } = useFormContext<BillingProfileFormValues>()
  const country = watch("country")

  return (
    <>
      <Section
        title="Pays"
        subtitle="Choisis d'abord ton pays — les autres champs s'adaptent en conséquence (SIRET pour la France, n° TVA intracom pour l'UE, adresse seule ailleurs)."
      >
        <Field
          label="Pays de facturation"
          htmlFor="country"
          error={errors.country?.message}
        >
          <CountrySelect
            value={country}
            onChange={(v) =>
              setValue("country", v, {
                shouldDirty: true,
                shouldValidate: true,
              })
            }
          />
        </Field>
      </Section>

      <Section title="Adresse">
        <div className="mb-3">
          <AddressAutocomplete
            country={country}
            onSelect={(addr) => {
              setValue("address_line1", addr.line1, { shouldDirty: true })
              setValue("postal_code", addr.postalCode, { shouldDirty: true })
              setValue("city", addr.city, { shouldDirty: true })
            }}
          />
        </div>
        <Field
          label="Adresse"
          htmlFor="address_line1"
          error={errors.address_line1?.message}
        >
          <Input
            id="address_line1"
            type="text"
            {...register("address_line1")}
            className={inputClasses}
            autoComplete="address-line1"
          />
        </Field>
        <Field label="Complément d'adresse (optionnel)" htmlFor="address_line2">
          <Input
            id="address_line2"
            type="text"
            {...register("address_line2")}
            className={inputClasses}
            autoComplete="address-line2"
          />
        </Field>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field
            label="Code postal"
            htmlFor="postal_code"
            error={errors.postal_code?.message}
          >
            <Input
              id="postal_code"
              type="text"
              {...register("postal_code")}
              className={inputClasses}
              autoComplete="postal-code"
            />
          </Field>
          <Field label="Ville" htmlFor="city" error={errors.city?.message}>
            <Input
              id="city"
              type="text"
              {...register("city")}
              className={inputClasses}
              autoComplete="address-level2"
            />
          </Field>
        </div>
      </Section>
    </>
  )
}

/**
 * Country dropdown grouped by region (EU / Europe / Americas / APAC /
 * MENA), backed by the same Stripe Connect supported list as the
 * payment-info onboarding. Native <Select> + <optgroup> keeps the
 * markup screen-reader and mobile-keyboard friendly without pulling a
 * combobox dependency.
 */
function CountrySelect({
  value,
  onChange,
}: {
  value: string
  onChange: (code: string) => void
}) {
  const groups: Record<string, SupportedCountry[]> = {
    eu: [],
    europe_other: [],
    americas: [],
    apac: [],
    mena: [],
  }
  for (const c of STRIPE_CONNECT_COUNTRIES) {
    groups[c.region].push(c)
  }
  const order: SupportedCountry["region"][] = [
    "eu",
    "europe_other",
    "americas",
    "apac",
    "mena",
  ]
  return (
    <Select
      id="country"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={cn(inputClasses, "appearance-none")}
      autoComplete="country"
    >
      <option value="">— Sélectionne —</option>
      {order.map((region) =>
        groups[region].length === 0 ? null : (
          <optgroup key={region} label={REGION_LABELS[region]}>
            {groups[region].map((c) => (
              <option key={c.code} value={c.code}>
                {c.flag} {c.labelFr}
              </option>
            ))}
          </optgroup>
        ),
      )}
    </Select>
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
    <div className="rounded-2xl border border-border bg-surface p-6">
      <h2 className="font-serif text-[18px] font-semibold tracking-[-0.01em] text-foreground">
        {title}
      </h2>
      {subtitle && (
        <p className="mt-1.5 text-[12.5px] leading-relaxed text-muted-foreground">
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
