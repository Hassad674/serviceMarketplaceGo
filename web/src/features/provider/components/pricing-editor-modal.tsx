"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { createPortal } from "react-dom"
import { X } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import type { Pricing, PricingKind } from "../api/profile-api"
import { formatPricing, type PricingLocale } from "../lib/pricing-format"
import { usePricing } from "../hooks/use-pricing"
import { useUpsertPricing } from "../hooks/use-upsert-pricing"
import { useDeletePricing } from "../hooks/use-delete-pricing"
import { PricingKindForm } from "./pricing-kind-form"

interface PricingEditorModalProps {
  open: boolean
  variant: PricingKind
  orgType: string | undefined
  referrerEnabled: boolean | undefined
  onClose: () => void
}

// Modal that owns the edit session for ONE pricing kind. The split
// across two pages (/profile for direct, /referral for referral)
// means a single modal instance never mixes both — the form inside
// is static and the preview strip tracks only one live row.
export function PricingEditorModal({
  open,
  variant,
  orgType,
  referrerEnabled,
  onClose,
}: PricingEditorModalProps) {
  const t = useTranslations("profile.pricing")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: persisted } = usePricing()
  const upsert = useUpsertPricing()
  const remove = useDeletePricing()

  const [preview, setPreviewState] = useState<Pricing | null>(null)
  const [prevOpen, setPrevOpen] = useState(open)
  if (prevOpen !== open) {
    setPrevOpen(open)
    if (!open) setPreviewState(null)
  }

  useEffect(() => {
    if (!open) return
    function handleKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose()
    }
    document.addEventListener("keydown", handleKey)
    return () => document.removeEventListener("keydown", handleKey)
  }, [open, onClose])

  const persistedRow = useMemo(() => {
    for (const row of persisted ?? []) {
      if (row.kind === variant) return row
    }
    return null
  }, [persisted, variant])

  const handleSave = useCallback(
    async (row: Pricing) => {
      await upsert.mutateAsync(row)
    },
    [upsert],
  )

  const handleDelete = useCallback(async () => {
    await remove.mutateAsync(variant)
    setPreviewState(null)
  }, [remove, variant])

  // Stable callback with empty deps — PricingKindForm uses this in
  // an effect. With a single form per modal we can use plain
  // setState which already has a stable reference.
  const setPreview = useCallback((row: Pricing | null) => {
    setPreviewState(row)
  }, [])

  if (!open) return null
  if (typeof window === "undefined") return null

  const title =
    variant === "direct" ? t("directModalTitle") : t("referralModalTitle")

  const body = (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/40 backdrop-blur-sm p-4"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={`pricing-editor-title-${variant}`}
        onClick={(event) => event.stopPropagation()}
        className="flex h-[90vh] max-h-[720px] w-full max-w-2xl flex-col rounded-xl border border-border bg-background shadow-xl"
      >
        <div className="flex items-center justify-between border-b border-border px-6 py-4">
          <h2
            id={`pricing-editor-title-${variant}`}
            className="text-lg font-semibold text-foreground"
          >
            {title}
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
          preview={preview ?? persistedRow}
          variant={variant}
          locale={locale}
        />

        <div className="flex-1 overflow-y-auto px-6 py-4">
          <PricingKindForm
            kind={variant}
            orgType={orgType}
            referrerEnabled={referrerEnabled}
            persisted={persistedRow}
            onSave={handleSave}
            onDelete={handleDelete}
            onPreviewChange={setPreview}
          />
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

interface PreviewStripProps {
  preview: Pricing | null
  variant: PricingKind
  locale: PricingLocale
}

function PreviewStrip({ preview, variant, locale }: PreviewStripProps) {
  const t = useTranslations("profile.pricing")
  if (!preview) return null
  return (
    <div className="border-b border-border px-6 py-3 bg-muted/30">
      <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-2">
        {t("previewHeading")}
      </p>
      <div className="inline-flex flex-col rounded-lg border border-border bg-background px-3 py-1.5 text-sm">
        <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
          {variant === "direct" ? t("kindDirect") : t("kindReferral")}
        </span>
        <span className="font-semibold text-foreground">
          {formatPricing(preview, locale)}
        </span>
        {preview.negotiable ? (
          <span className="mt-1 inline-flex w-fit items-center rounded-full border border-primary/30 bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
            {t("negotiableBadge")}
          </span>
        ) : null}
      </div>
    </div>
  )
}
