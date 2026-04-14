"use client"

import { useState } from "react"
import { Pencil } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import type { Pricing } from "../api/profile-api"
import { formatPricing, type PricingLocale } from "../lib/pricing-format"
import { usePricing } from "../hooks/use-pricing"
import { PricingEditorModal } from "./pricing-editor-modal"

interface PricingSectionProps {
  orgType: string | undefined
  referrerEnabled: boolean | undefined
  readOnly?: boolean
}

// Compact card mirrors the Skills section pattern: it renders the
// current pricing rows as chips and opens a modal editor on click.
// The modal owns the entire edit session; this file stays skimmable.
export function PricingSection({
  orgType,
  referrerEnabled,
  readOnly = false,
}: PricingSectionProps) {
  const t = useTranslations("profile.pricing")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: pricing } = usePricing()
  const [modalOpen, setModalOpen] = useState(false)

  const rows = pricing ?? []

  if (orgType === "enterprise") return null
  if (readOnly && rows.length === 0) return null

  return (
    <section
      aria-labelledby="pricing-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="pricing-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
        {!readOnly ? (
          <p className="text-sm text-muted-foreground">
            {t("sectionSubtitle")}
          </p>
        ) : null}
      </header>

      <PricingSummary rows={rows} locale={locale} />

      {!readOnly ? (
        <button
          type="button"
          onClick={() => setModalOpen(true)}
          className="mt-4 inline-flex items-center gap-2 rounded-md border border-border h-9 px-4 text-sm font-medium text-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
        >
          <Pencil className="h-4 w-4" aria-hidden="true" />
          {t("editButton")}
        </button>
      ) : null}

      {modalOpen ? (
        <PricingEditorModal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          orgType={orgType}
          referrerEnabled={referrerEnabled}
        />
      ) : null}
    </section>
  )
}

interface PricingSummaryProps {
  rows: Pricing[]
  locale: PricingLocale
}

function PricingSummary({ rows, locale }: PricingSummaryProps) {
  const t = useTranslations("profile.pricing")
  if (rows.length === 0) {
    return (
      <p className="text-sm italic text-muted-foreground">{t("empty")}</p>
    )
  }
  return (
    <ul className="flex flex-wrap gap-2" aria-label={t("sectionTitle")}>
      {rows.map((row) => (
        <li
          key={row.kind}
          className="inline-flex flex-col rounded-lg border border-border bg-muted/30 px-3 py-2 text-sm"
        >
          <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
            {row.kind === "direct" ? t("kindDirect") : t("kindReferral")}
          </span>
          <span className="font-semibold text-foreground">
            {formatPricing(row, locale)}
          </span>
          {row.note ? (
            <span className="text-xs text-muted-foreground italic truncate max-w-[220px]">
              {row.note}
            </span>
          ) : null}
        </li>
      ))}
    </ul>
  )
}
