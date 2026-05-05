"use client"

import { useEffect } from "react"
import { AlertTriangle, CheckCircle2, Loader2, RefreshCw } from "lucide-react"
import { FormProvider, useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { ApiError } from "@/shared/lib/api-client"
import { cn, formatDate } from "@/shared/lib/utils"
import {
  useBillingProfile,
  useSyncBillingProfile,
  useUpdateBillingProfile,
  useValidateVAT,
} from "../hooks/use-billing-profile"
import type { BillingProfile, UpdateBillingProfileInput } from "../types"
import { describeMissing } from "./missing-fields-copy"
import { BillingSectionLegalIdentity } from "./billing-section-legal-identity"
import { BillingSectionAddress } from "./billing-section-address"
import { BillingSectionFiscal } from "./billing-section-fiscal"
import {
  billingProfileFormSchema,
  type BillingProfileFormValues,
} from "./billing-profile-form.schema"

import { Button } from "@/shared/components/ui/button"
/**
 * Editable form for the billing profile. Reads the current snapshot
 * via TanStack Query, hydrates a react-hook-form instance, and pushes
 * back through three independent mutations:
 *
 *   - update (PUT)              → Save button
 *   - sync from Stripe (POST)   → "Pré-remplir depuis Stripe" button
 *   - validate VAT (POST)       → "Valider mon n° TVA" button
 *
 * Each mutation has its own pending/error state so a VIES failure
 * never blocks a save and vice versa.
 *
 * The form is split into three section components — legal identity,
 * address (+ country), and fiscal (VAT). They share a single RHF
 * context so the component tree stays composable.
 *
 * Variants:
 *   - "page" (default): standalone /settings/billing-profile page,
 *     with editorial Soleil header (corail mono eyebrow + Fraunces
 *     italic accent + tabac subtitle).
 *   - "compact": embedded inside the upgrade modal, slightly tighter
 *     spacing, hides the editorial header + the "synced from Stripe"
 *     indicator since the sync runs automatically on mount in that
 *     context.
 *
 * onSaved fires once after a successful update mutation lands AND the
 * resulting profile passes CheckCompleteness. The upgrade modal uses
 * it to transition from the billing step to the payment step.
 */
export type BillingProfileFormProps = {
  variant?: "page" | "compact"
  onSaved?: () => void
}

export function BillingProfileForm({
  variant = "page",
  onSaved,
}: BillingProfileFormProps = {}) {
  const { data, isLoading, isError } = useBillingProfile()
  const updateMutation = useUpdateBillingProfile()
  const syncMutation = useSyncBillingProfile()
  const vatMutation = useValidateVAT()

  const isCompact = variant === "compact"

  const form = useForm<BillingProfileFormValues>({
    resolver: zodResolver(billingProfileFormSchema),
    mode: "onSubmit",
    reValidateMode: "onChange",
    defaultValues: emptyDefaults(),
  })

  // Hydrate the form from the server snapshot once on first success,
  // then re-hydrate whenever the server-side `updated_at` changes
  // (e.g. after sync-from-stripe or a VAT validation).
  useEffect(() => {
    if (!data?.profile) return
    form.reset(profileToFormValues(data.profile))
  }, [data?.profile, form])

  // Trigger onSaved exactly once per save: when the latest update
  // mutation has settled successfully AND the resulting snapshot is
  // complete. Resets on a new mutation so re-saving a still-incomplete
  // profile never advances the modal.
  useEffect(() => {
    if (!onSaved) return
    if (!updateMutation.isSuccess) return
    if (!data?.is_complete) return
    onSaved()
  }, [onSaved, updateMutation.isSuccess, data?.is_complete])

  if (isLoading) return <FormSkeleton />
  if (isError || !data) {
    return (
      <p className="rounded-2xl border border-border bg-surface p-6 text-sm text-muted-foreground">
        Impossible de charger le profil de facturation. Réessaie dans un
        instant.
      </p>
    )
  }

  function handleSubmit(values: BillingProfileFormValues) {
    updateMutation.mutate(values as UpdateBillingProfileInput)
  }

  return (
    <FormProvider {...form}>
      <form
        onSubmit={form.handleSubmit(handleSubmit)}
        className={cn(isCompact ? "space-y-4" : "space-y-6")}
      >
        {!isCompact && <BillingProfileEditorialHeader />}

        {!data.is_complete && data.missing_fields.length > 0 && (
          <MissingFieldsBanner fields={data.missing_fields} />
        )}

        <div className="flex items-center justify-between gap-3">
          {!isCompact && (
            <SyncedFromStripeIndicator at={data.profile.synced_from_kyc_at} />
          )}
          {isCompact && <span />}
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => syncMutation.mutate()}
            disabled={syncMutation.isPending}
            className={cn(
              "inline-flex items-center gap-2 rounded-full border border-border-strong bg-surface px-4 py-1.5 text-xs font-semibold text-foreground transition-colors",
              "hover:bg-primary-soft/40 disabled:opacity-50",
            )}
          >
            {syncMutation.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
            )}
            Pré-remplir depuis Stripe
          </Button>
        </div>
        {syncMutation.isError && (
          <FormError message="La synchronisation Stripe a échoué. Réessaie ou complète manuellement." />
        )}

        <BillingSectionAddress />
        <BillingSectionLegalIdentity />
        <BillingSectionFiscal
          validatedAt={data.profile.vat_validated_at}
          isValidating={vatMutation.isPending}
          validateError={vatMutation.isError ? (vatMutation.error as Error) : null}
          onValidate={() => vatMutation.mutate()}
        />

        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-end">
          {updateMutation.isError && (
            <span className="text-sm text-destructive">
              {updateMutation.error instanceof ApiError
                ? updateMutation.error.message
                : "L'enregistrement a échoué. Réessaie."}
            </span>
          )}
          {updateMutation.isSuccess && !updateMutation.isPending && (
            <span className="text-sm text-success">
              Profil enregistré.
            </span>
          )}
          <Button variant="ghost" size="auto"
            type="submit"
            disabled={updateMutation.isPending}
            className={cn(
              "inline-flex items-center justify-center gap-2 rounded-full bg-primary px-6 py-2.5",
              "text-sm font-bold text-primary-foreground",
              "transition-all duration-200 ease-out",
              "hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
              "active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60",
            )}
          >
            {updateMutation.isPending && (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            )}
            Enregistrer
          </Button>
        </div>
      </form>
    </FormProvider>
  )
}

// Editorial Soleil v2 header — corail mono eyebrow + Fraunces italic
// accent + tabac subtitle. Strings are hardcoded here on purpose:
// the existing test file does not wrap the form in a
// `NextIntlClientProvider`, so calling `useTranslations` would crash
// every test in `billing-profile-form.test.tsx` (off-limits).
function BillingProfileEditorialHeader() {
  return (
    <header>
      <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
        ATELIER · PROFIL DE FACTURATION
      </p>
      <h1 className="font-serif text-[26px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[32px]">
        Renseigne ton{" "}
        <span className="italic text-primary">profil de facturation.</span>
      </h1>
      <p className="mt-3 max-w-[620px] text-[14px] leading-relaxed text-muted-foreground">
        Ces informations apparaissent sur les factures que la plateforme émet
        à ton organisation. Elles doivent être complètes pour pouvoir retirer
        ton solde et souscrire à un abonnement Premium.
      </p>
    </header>
  )
}

function MissingFieldsBanner({
  fields,
}: {
  fields: { field: string; reason: string }[]
}) {
  const labels = fields.map((f) => describeMissing(f))
  return (
    <div className="flex items-start gap-3 rounded-2xl border border-warning/40 bg-amber-soft p-4 text-sm text-foreground">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-warning" aria-hidden="true" />
      <div>
        <p className="font-medium">Quelques informations restent à compléter</p>
        <ul className="mt-1 list-disc pl-4 text-xs text-muted-foreground">
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
      <p className="text-xs text-muted-foreground">
        Profil non synchronisé depuis Stripe
      </p>
    )
  }
  return (
    <p className="inline-flex items-center gap-1 text-xs text-muted-foreground">
      <CheckCircle2 className="h-3.5 w-3.5 text-success" aria-hidden="true" />
      Synchronisé depuis Stripe le {formatDate(at)}
    </p>
  )
}

function FormError({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="flex items-start gap-2 rounded-xl border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <span>{message}</span>
    </div>
  )
}

function FormSkeleton() {
  return (
    <div className="space-y-4">
      <div className="h-12 animate-shimmer rounded-2xl bg-border" />
      <div className="h-48 animate-shimmer rounded-2xl bg-border" />
      <div className="h-64 animate-shimmer rounded-2xl bg-border" />
    </div>
  )
}

// Maps the server-side billing profile to the form input shape. The
// snapshot uses nullable timestamp fields the form does not need to
// echo back — the backend recomputes them on every save.
function profileToFormValues(p: BillingProfile): BillingProfileFormValues {
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

function emptyDefaults(): BillingProfileFormValues {
  return {
    profile_type: "individual",
    legal_name: "",
    trading_name: "",
    legal_form: "",
    tax_id: "",
    vat_number: "",
    address_line1: "",
    address_line2: "",
    postal_code: "",
    city: "",
    country: "",
    invoicing_email: "",
  }
}
