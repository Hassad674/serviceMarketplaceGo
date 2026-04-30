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
      <p className="rounded-xl border border-slate-200 bg-white p-6 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900">
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
        {!data.is_complete && data.missing_fields.length > 0 && (
          <MissingFieldsBanner fields={data.missing_fields} />
        )}

        <div className="flex items-center justify-between gap-3">
          {!isCompact && (
            <SyncedFromStripeIndicator at={data.profile.synced_from_kyc_at} />
          )}
          {isCompact && <span />}
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
    </FormProvider>
  )
}

function MissingFieldsBanner({
  fields,
}: {
  fields: { field: string; reason: string }[]
}) {
  const labels = fields.map((f) => describeMissing(f))
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
      <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" aria-hidden="true" />
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
