"use client"

import { useState } from "react"
import { Pencil } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import type { Pricing, PricingKind } from "../api/profile-api"
import { formatPricing, type PricingLocale } from "../lib/pricing-format"
import { usePricing } from "../hooks/use-pricing"
import { PricingEditorModal } from "./pricing-editor-modal"
import { Button } from "@/shared/components/ui/button"

// PricingVariant determines which kind of pricing the section
// surfaces. The profile page renders variant="direct" (freelance
// pricing), the referral page renders variant="referral"
// (apporteur d'affaires commission). The modal opened from this
// section only edits its own kind — the two flows never overlap.
export type PricingVariant = "direct" | "referral"

interface PricingSectionProps {
  variant: PricingVariant
  orgType: string | undefined
  referrerEnabled: boolean | undefined
  readOnly?: boolean
}

export function PricingSection({
  variant,
  orgType,
  referrerEnabled,
  readOnly = false,
}: PricingSectionProps) {
  const t = useTranslations("profile.pricing")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: pricing } = usePricing()
  const [modalOpen, setModalOpen] = useState(false)

  const rows = pricing ?? []
  const row = rows.find((r) => r.kind === variant) ?? null

  if (!isVariantVisible(variant, orgType, referrerEnabled)) return null
  if (readOnly && !row) return null

  const titleKey = variant === "direct" ? "directSectionTitle" : "referralSectionTitle"
  const subtitleKey =
    variant === "direct" ? "directSectionSubtitle" : "referralSectionSubtitle"

  return (
    <section
      aria-labelledby={`pricing-section-title-${variant}`}
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id={`pricing-section-title-${variant}`}
          className="text-lg font-semibold text-foreground"
        >
          {t(titleKey)}
        </h2>
        {!readOnly ? (
          <p className="text-sm text-muted-foreground">{t(subtitleKey)}</p>
        ) : null}
      </header>

      <PricingRow row={row} locale={locale} />

      {!readOnly ? (
        <Button variant="ghost" size="auto"
          type="button"
          onClick={() => setModalOpen(true)}
          className="mt-4 inline-flex items-center gap-2 rounded-md border border-border h-9 px-4 text-sm font-medium text-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
        >
          <Pencil className="h-4 w-4" aria-hidden="true" />
          {t("editButton")}
        </Button>
      ) : null}

      {modalOpen ? (
        <PricingEditorModal
          open={modalOpen}
          variant={variant}
          orgType={orgType}
          referrerEnabled={referrerEnabled}
          onClose={() => setModalOpen(false)}
        />
      ) : null}
    </section>
  )
}

// isVariantVisible gates rendering per org capability: enterprise
// orgs show neither variant, agencies only see direct, providers
// see direct always and referral only when referrer_enabled.
function isVariantVisible(
  variant: PricingVariant,
  orgType: string | undefined,
  referrerEnabled: boolean | undefined,
): boolean {
  if (orgType === "enterprise") return false
  if (variant === "direct") {
    return orgType === "agency" || orgType === "provider_personal"
  }
  return orgType === "provider_personal" && referrerEnabled === true
}

interface PricingRowProps {
  row: Pricing | null
  locale: PricingLocale
}

function PricingRow({ row, locale }: PricingRowProps) {
  const t = useTranslations("profile.pricing")
  if (!row) {
    return (
      <p className="text-sm italic text-muted-foreground">{t("empty")}</p>
    )
  }
  return (
    <div
      key={row.kind}
      className="inline-flex flex-col rounded-lg border border-border bg-muted/30 px-3 py-2 text-sm"
    >
      <span className="font-semibold text-foreground">
        {formatPricing(row, locale)}
      </span>
      <div className="mt-1 flex flex-wrap items-center gap-2">
        {row.negotiable ? (
          <span className="inline-flex items-center rounded-full border border-primary/30 bg-primary/10 px-2 py-0.5 text-[11px] font-medium text-primary">
            {t("negotiableBadge")}
          </span>
        ) : null}
        {row.note ? (
          <span className="text-xs text-muted-foreground italic truncate max-w-[220px]">
            {row.note}
          </span>
        ) : null}
      </div>
    </div>
  )
}

// Re-exported so consumers that need to distinguish the two kinds
// (e.g. a future A/B test toggle) do not duplicate the union.
export type { PricingKind }
