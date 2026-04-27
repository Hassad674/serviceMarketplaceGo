"use client"

import { useEffect, useMemo, useState } from "react"
import {
  AlertTriangle,
  CheckCircle2,
  Loader2,
  RefreshCw,
  XCircle,
} from "lucide-react"
import { ApiError } from "@/shared/lib/api-client"
import { cn, formatDate } from "@/shared/lib/utils"
import {
  useBillingProfile,
  useSyncBillingProfile,
  useUpdateBillingProfile,
  useValidateVAT,
} from "../hooks/use-billing-profile"
import type {
  BillingProfile,
  ProfileType,
  UpdateBillingProfileInput,
} from "../types"
import {
  REGION_LABELS,
  STRIPE_CONNECT_COUNTRIES,
  type SupportedCountry,
} from "@/shared/lib/stripe-countries"
import { isEUCountry } from "./eu-countries"
import { describeMissing } from "./missing-fields-copy"

/**
 * Editable form for the billing profile. Reads the current snapshot
 * via TanStack Query, keeps a local controlled copy while editing,
 * and pushes back through three independent mutations:
 *
 *   - update (PUT)              → Save button
 *   - sync from Stripe (POST)   → "Pré-remplir depuis Stripe" button
 *   - validate VAT (POST)       → "Valider mon n° TVA" button
 *
 * Each mutation has its own pending/error state so a VIES failure
 * never blocks a save and vice versa.
 *
 * Variants:
 *   - "page" (default): standalone /settings/billing-profile page
 *   - "compact": embedded inside the upgrade modal, slightly tighter
 *     spacing, hides the "synced from Stripe" indicator since the
 *     sync runs automatically on mount in that context.
 *
 * onSaved fires once after a successful update mutation lands AND the
 * resulting profile passes CheckCompleteness. The upgrade modal uses
 * it to transition from the billing step to the payment step.
 */
export type BillingProfileFormProps = {
  variant?: "page" | "compact"
  onSaved?: () => void
}

export function BillingProfileForm({ variant = "page", onSaved }: BillingProfileFormProps = {}) {
  const { data, isLoading, isError } = useBillingProfile()
  const updateMutation = useUpdateBillingProfile()
  const syncMutation = useSyncBillingProfile()
  const vatMutation = useValidateVAT()

  const [form, setForm] = useState<UpdateBillingProfileInput | null>(null)
  const isCompact = variant === "compact"

  // Hydrate the local form from the server snapshot once on first
  // success, then re-hydrate whenever the server-side `updated_at`
  // changes (e.g. after sync-from-stripe or a VAT validation).
  useEffect(() => {
    if (!data?.profile) return
    setForm(profileToInput(data.profile))
  }, [data?.profile])

  // Trigger onSaved exactly once per save: when the latest update
  // mutation has settled successfully AND the resulting snapshot is
  // complete. We watch isSuccess + the freshest is_complete flag.
  // Resets on a new mutation so re-saving a still-incomplete profile
  // never advances the modal.
  useEffect(() => {
    if (!onSaved) return
    if (!updateMutation.isSuccess) return
    if (!data?.is_complete) return
    onSaved()
  }, [onSaved, updateMutation.isSuccess, data?.is_complete])

  if (isLoading) return <FormSkeleton />
  if (isError || !data) {
    return (
      <p className="rounded-xl border border-slate-200 bg-white p-6 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900">
        Impossible de charger le profil de facturation. Réessaie dans un
        instant.
      </p>
    )
  }
  if (!form) return <FormSkeleton />

  const isEU = isEUCountry(form.country)
  const isFR = form.country === "FR"
  const canValidateVAT = isEU && form.vat_number.trim().length > 0

  function patch<K extends keyof UpdateBillingProfileInput>(
    key: K,
    value: UpdateBillingProfileInput[K],
  ) {
    setForm((prev) => (prev ? { ...prev, [key]: value } : prev))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    updateMutation.mutate(form)
  }

  return (
    <form onSubmit={handleSubmit} className={cn(isCompact ? "space-y-4" : "space-y-6")}>
      {!data.is_complete && data.missing_fields.length > 0 && (
        <MissingFieldsBanner fields={data.missing_fields} />
      )}

      {!isCompact && (
        <div className="flex items-center justify-between gap-3">
          <SyncedFromStripeIndicator at={data.profile.synced_from_kyc_at} />
          <button
            type="button"
            onClick={() => syncMutation.mutate()}
            disabled={syncMutation.isPending}
            className={cn(
              "inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition-colors",
              "hover:bg-slate-50 disabled:opacity-50",
              "dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700",
            )}
          >
            {syncMutation.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
            )}
            Pré-remplir depuis Stripe
          </button>
        </div>
      )}
      {syncMutation.isError && (
        <FormError message="La synchronisation Stripe a échoué. Réessaie ou complète manuellement." />
      )}

      <Section
        title="Pays"
        subtitle="Choisis d'abord ton pays — les autres champs s'adaptent en conséquence (SIRET pour la France, n° TVA intracom pour l'UE, adresse seule ailleurs)."
      >
        <Field label="Pays de facturation" htmlFor="country">
          <CountrySelect
            value={form.country}
            onChange={(v) => patch("country", v)}
          />
        </Field>
      </Section>

      <Section title="Type de profil">
        <ProfileTypeRadio
          value={form.profile_type}
          onChange={(v) => patch("profile_type", v)}
        />
      </Section>

      <Section title="Identité légale">
        <Field label="Raison sociale ou nom légal" htmlFor="legal_name">
          <input
            id="legal_name"
            type="text"
            value={form.legal_name}
            onChange={(e) => patch("legal_name", e.target.value)}
            className={inputClasses}
            autoComplete="organization"
          />
        </Field>
        {form.profile_type === "business" && (
          <>
            <Field label="Nom commercial (optionnel)" htmlFor="trading_name">
              <input
                id="trading_name"
                type="text"
                value={form.trading_name}
                onChange={(e) => patch("trading_name", e.target.value)}
                className={inputClasses}
              />
            </Field>
            <Field label="Forme juridique" htmlFor="legal_form">
              <input
                id="legal_form"
                type="text"
                value={form.legal_form}
                onChange={(e) => patch("legal_form", e.target.value)}
                className={inputClasses}
                placeholder="SAS, SARL, EURL, etc."
              />
            </Field>
          </>
        )}
      </Section>

      <Section title="Identifiants fiscaux">
        {isFR && (
          <Field
            label="Numéro SIRET"
            htmlFor="tax_id"
            hint="14 chiffres, sans espace"
          >
            <input
              id="tax_id"
              type="text"
              inputMode="numeric"
              value={form.tax_id}
              onChange={(e) => patch("tax_id", e.target.value)}
              className={inputClasses}
              maxLength={14}
            />
          </Field>
        )}
        {!isFR && (
          <Field label="Identifiant fiscal" htmlFor="tax_id">
            <input
              id="tax_id"
              type="text"
              value={form.tax_id}
              onChange={(e) => patch("tax_id", e.target.value)}
              className={inputClasses}
            />
          </Field>
        )}
        {isEU && (
          <VATRow
            value={form.vat_number}
            onChange={(v) => patch("vat_number", v)}
            validatedAt={data.profile.vat_validated_at}
            canValidate={canValidateVAT}
            isValidating={vatMutation.isPending}
            validateError={
              vatMutation.isError ? (vatMutation.error as Error) : null
            }
            onValidate={() => vatMutation.mutate()}
          />
        )}
      </Section>

      <Section title="Adresse">
        <Field label="Adresse" htmlFor="address_line1">
          <input
            id="address_line1"
            type="text"
            value={form.address_line1}
            onChange={(e) => patch("address_line1", e.target.value)}
            className={inputClasses}
            autoComplete="address-line1"
          />
        </Field>
        <Field
          label="Complément d'adresse (optionnel)"
          htmlFor="address_line2"
        >
          <input
            id="address_line2"
            type="text"
            value={form.address_line2}
            onChange={(e) => patch("address_line2", e.target.value)}
            className={inputClasses}
            autoComplete="address-line2"
          />
        </Field>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Code postal" htmlFor="postal_code">
            <input
              id="postal_code"
              type="text"
              value={form.postal_code}
              onChange={(e) => patch("postal_code", e.target.value)}
              className={inputClasses}
              autoComplete="postal-code"
            />
          </Field>
          <Field label="Ville" htmlFor="city">
            <input
              id="city"
              type="text"
              value={form.city}
              onChange={(e) => patch("city", e.target.value)}
              className={inputClasses}
              autoComplete="address-level2"
            />
          </Field>
        </div>
      </Section>

      <Section title="Email de facturation">
        <Field
          label="Email où envoyer les factures"
          htmlFor="invoicing_email"
        >
          <input
            id="invoicing_email"
            type="email"
            value={form.invoicing_email}
            onChange={(e) => patch("invoicing_email", e.target.value)}
            className={inputClasses}
            autoComplete="email"
          />
        </Field>
      </Section>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-end">
        {updateMutation.isError && (
          <span className="text-sm text-red-600 dark:text-red-400">
            {updateMutation.error instanceof ApiError
              ? updateMutation.error.message
              : "L'enregistrement a échoué. Réessaie."}
          </span>
        )}
        {updateMutation.isSuccess && !updateMutation.isPending && (
          <span className="text-sm text-emerald-600 dark:text-emerald-400">
            Profil enregistré.
          </span>
        )}
        <button
          type="submit"
          disabled={updateMutation.isPending}
          className={cn(
            "inline-flex items-center justify-center gap-2 rounded-lg px-5 py-2.5",
            "text-sm font-semibold text-white",
            "gradient-primary hover:shadow-glow active:scale-[0.98]",
            "disabled:opacity-50 disabled:cursor-not-allowed transition-all",
          )}
        >
          {updateMutation.isPending && (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          )}
          Enregistrer
        </button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Sub-components — kept inside the same file because none of them are reused.
// ---------------------------------------------------------------------------

const inputClasses = cn(
  "w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-xs",
  "focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
  "dark:border-slate-700 dark:bg-slate-900 dark:text-white",
)

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

/**
 * Country dropdown grouped by region (EU / Europe / Americas / APAC /
 * MENA), backed by the same Stripe Connect supported list as the
 * payment-info onboarding. Native <select> + <optgroup> keeps the
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
    <select
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
    </select>
  )
}

function Field({
  label,
  htmlFor,
  hint,
  children,
}: {
  label: string
  htmlFor: string
  hint?: string
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
      {hint && (
        <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">
          {hint}
        </p>
      )}
    </div>
  )
}

function ProfileTypeRadio({
  value,
  onChange,
}: {
  value: ProfileType
  onChange: (v: ProfileType) => void
}) {
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
              name="profile_type"
              value={option}
              checked={selected}
              onChange={() => onChange(option)}
              className="h-4 w-4 accent-rose-500"
            />
            {label}
          </label>
        )
      })}
    </div>
  )
}

function VATRow({
  value,
  onChange,
  validatedAt,
  canValidate,
  isValidating,
  validateError,
  onValidate,
}: {
  value: string
  onChange: (v: string) => void
  validatedAt: string | null
  canValidate: boolean
  isValidating: boolean
  validateError: Error | null
  onValidate: () => void
}) {
  return (
    <div className="space-y-2">
      <Field label="Numéro de TVA intracommunautaire" htmlFor="vat_number">
        <input
          id="vat_number"
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
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
  )
}

function MissingFieldsBanner({
  fields,
}: {
  fields: { field: string; reason: string }[]
}) {
  const labels = useMemo(() => fields.map((f) => describeMissing(f)), [fields])
  return (
    <div className="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div>
        <p className="font-medium">Quelques informations restent à compléter</p>
        <ul className="mt-1 list-disc pl-4 text-xs">
          {labels.map((label) => (
            <li key={label}>{label}</li>
          ))}
        </ul>
      </div>
    </div>
  )
}

function SyncedFromStripeIndicator({ at }: { at: string | null }) {
  if (!at) {
    return (
      <p className="text-xs text-slate-500 dark:text-slate-400">
        Profil non synchronisé depuis Stripe
      </p>
    )
  }
  return (
    <p className="inline-flex items-center gap-1 text-xs text-slate-500 dark:text-slate-400">
      <CheckCircle2
        className="h-3.5 w-3.5 text-emerald-500"
        aria-hidden="true"
      />
      Synchronisé depuis Stripe le {formatDate(at)}
    </p>
  )
}

function FormError({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <span>{message}</span>
    </div>
  )
}

function FormSkeleton() {
  return (
    <div className="space-y-4">
      <div className="h-12 animate-shimmer rounded-xl bg-slate-100 dark:bg-slate-800" />
      <div className="h-48 animate-shimmer rounded-xl bg-slate-100 dark:bg-slate-800" />
      <div className="h-64 animate-shimmer rounded-xl bg-slate-100 dark:bg-slate-800" />
    </div>
  )
}

// Maps the server-side billing profile to the form input shape. The
// snapshot uses nullable timestamp fields the form does not need to
// echo back — the backend recomputes them on every save.
function profileToInput(p: BillingProfile): UpdateBillingProfileInput {
  return {
    profile_type: p.profile_type,
    legal_name: p.legal_name,
    trading_name: p.trading_name,
    legal_form: p.legal_form,
    tax_id: p.tax_id,
    vat_number: p.vat_number,
    address_line1: p.address_line1,
    address_line2: p.address_line2,
    postal_code: p.postal_code,
    city: p.city,
    country: p.country,
    invoicing_email: p.invoicing_email,
  }
}
