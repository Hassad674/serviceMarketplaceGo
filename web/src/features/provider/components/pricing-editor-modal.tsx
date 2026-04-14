"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { createPortal } from "react-dom"
import { X } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import type {
  Pricing,
  PricingKind,
  PricingType,
} from "../api/profile-api"
import { formatPricing, type PricingLocale } from "../lib/pricing-format"
import { usePricing } from "../hooks/use-pricing"
import { useUpsertPricing } from "../hooks/use-upsert-pricing"
import { useDeletePricing } from "../hooks/use-delete-pricing"
import { PricingKindForm } from "./pricing-kind-form"

interface PricingEditorModalProps {
  open: boolean
  onClose: () => void
  orgType: string | undefined
  referrerEnabled: boolean | undefined
}

// Modal that owns the edit session. Renders up to two forms side-by-
// side (direct + referral) depending on the org's capabilities. Each
// form is a self-contained sub-editor; this parent just orchestrates
// opening/closing + the preview row at the top.
export function PricingEditorModal({
  open,
  onClose,
  orgType,
  referrerEnabled,
}: PricingEditorModalProps) {
  const t = useTranslations("profile.pricing")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: persisted } = usePricing()
  const upsert = useUpsertPricing()
  const remove = useDeletePricing()

  // Local previews let the user see "how it will render on the card"
  // even for rows that are still unsaved. The previews reset when the
  // modal closes — done during render via a tracked key instead of in
  // an effect to satisfy react-hooks/set-state-in-effect.
  const [previews, setPreviews] = useState<Record<PricingKind, Pricing | null>>(
    { direct: null, referral: null },
  )
  const [prevOpen, setPrevOpen] = useState(open)
  if (prevOpen !== open) {
    setPrevOpen(open)
    if (!open) setPreviews({ direct: null, referral: null })
  }

  useEffect(() => {
    if (!open) return
    function handleKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose()
    }
    document.addEventListener("keydown", handleKey)
    return () => document.removeEventListener("keydown", handleKey)
  }, [open, onClose])

  const canEditDirect = useMemo(
    () => orgType === "provider_personal" || orgType === "agency",
    [orgType],
  )
  const canEditReferral = useMemo(
    () => orgType === "provider_personal" && referrerEnabled === true,
    [orgType, referrerEnabled],
  )

  const persistedByKind = useMemo(() => {
    const map: Partial<Record<PricingKind, Pricing>> = {}
    for (const row of persisted ?? []) map[row.kind] = row
    return map
  }, [persisted])

  const handleSave = useCallback(
    async (row: Pricing) => {
      await upsert.mutateAsync(row)
    },
    [upsert],
  )

  const handleDelete = useCallback(
    async (kind: PricingKind) => {
      await remove.mutateAsync(kind)
      setPreviews((current) => ({ ...current, [kind]: null }))
    },
    [remove],
  )

  const setPreview = useCallback(
    (kind: PricingKind, row: Pricing | null) => {
      setPreviews((current) => ({ ...current, [kind]: row }))
    },
    [],
  )

  if (!open) return null
  if (typeof window === "undefined") return null

  const body = (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/40 backdrop-blur-sm p-4"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="pricing-editor-title"
        onClick={(event) => event.stopPropagation()}
        className="flex h-[90vh] max-h-[720px] w-full max-w-2xl flex-col rounded-xl border border-border bg-background shadow-xl"
      >
        <div className="flex items-center justify-between border-b border-border px-6 py-4">
          <h2
            id="pricing-editor-title"
            className="text-lg font-semibold text-foreground"
          >
            {t("modalTitle")}
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label={t("close")}
            className="rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        <PreviewStrip
          previews={previews}
          persistedDirect={persistedByKind.direct ?? null}
          persistedReferral={persistedByKind.referral ?? null}
          locale={locale}
        />

        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-6">
          {canEditDirect ? (
            <PricingKindForm
              kind="direct"
              persisted={persistedByKind.direct ?? null}
              onSave={handleSave}
              onDelete={handleDelete}
              onPreviewChange={(row) => setPreview("direct", row)}
            />
          ) : null}
          {canEditReferral ? (
            <PricingKindForm
              kind="referral"
              persisted={persistedByKind.referral ?? null}
              onSave={handleSave}
              onDelete={handleDelete}
              onPreviewChange={(row) => setPreview("referral", row)}
            />
          ) : null}
        </div>

        <div className="border-t border-border px-6 py-4 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md h-9 px-4 text-sm font-medium text-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
          >
            {t("close")}
          </button>
        </div>
      </div>
    </div>
  )

  return createPortal(body, document.body)
}

// ----- Preview strip ----------------------------------------------------

const EMPTY_PREVIEWS: Record<PricingKind, Pricing | null> = {
  direct: null,
  referral: null,
}

interface PreviewStripProps {
  previews: Record<PricingKind, Pricing | null>
  persistedDirect: Pricing | null
  persistedReferral: Pricing | null
  locale: PricingLocale
}

function PreviewStrip({
  previews = EMPTY_PREVIEWS,
  persistedDirect,
  persistedReferral,
  locale,
}: PreviewStripProps) {
  const t = useTranslations("profile.pricing")
  const direct = previews.direct ?? persistedDirect
  const referral = previews.referral ?? persistedReferral

  if (!direct && !referral) return null

  return (
    <div className="border-b border-border px-6 py-3 bg-muted/30">
      <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-2">
        {t("previewHeading")}
      </p>
      <div className="flex flex-wrap gap-3">
        {direct ? (
          <PreviewChip
            label={t("kindDirect")}
            value={formatPricing(direct, locale)}
          />
        ) : null}
        {referral ? (
          <PreviewChip
            label={t("kindReferral")}
            value={formatPricing(referral, locale)}
          />
        ) : null}
      </div>
    </div>
  )
}

function PreviewChip({ label, value }: { label: string; value: string }) {
  return (
    <div className="inline-flex flex-col rounded-lg border border-border bg-background px-3 py-1.5 text-sm">
      <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      <span className="font-semibold text-foreground">{value}</span>
    </div>
  )
}

export type { PricingType }
