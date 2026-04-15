"use client"

import { useLocale, useTranslations } from "next-intl"
import {
  formatPricing,
  type FormattablePricing,
} from "@/shared/lib/profile/pricing-format"

interface PricingDisplayCardProps {
  pricing: FormattablePricing & { note: string; negotiable: boolean } | null
  titleKey: "directSectionTitle" | "referralSectionTitle"
}

// PricingDisplayCard renders a pricing row in read-only mode for
// public profile pages. Reads the row from props (no hook call) so
// visitors always see the profile owner's rate — not their own.
// Collapses to null when no pricing row exists so the public viewer
// never sees an empty placeholder.
export function PricingDisplayCard({
  pricing,
  titleKey,
}: PricingDisplayCardProps) {
  const t = useTranslations("profile.pricing")
  const locale = useLocale() === "fr" ? "fr" : "en"
  if (!pricing) return null

  return (
    <section
      aria-labelledby={`public-pricing-title-${titleKey}`}
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4">
        <h2
          id={`public-pricing-title-${titleKey}`}
          className="text-lg font-semibold text-foreground"
        >
          {t(titleKey)}
        </h2>
      </header>

      <div className="flex items-baseline flex-wrap gap-2">
        <span className="text-2xl font-bold text-foreground">
          {formatPricing(pricing, locale)}
        </span>
        {pricing.negotiable ? (
          <span className="inline-flex items-center rounded-full bg-emerald-50 text-emerald-700 border border-emerald-200 px-2.5 py-0.5 text-xs font-medium dark:bg-emerald-500/10 dark:text-emerald-300 dark:border-emerald-500/30">
            {t("negotiableYes")}
          </span>
        ) : null}
      </div>

      {pricing.note.trim() !== "" ? (
        <p className="mt-3 text-sm text-muted-foreground italic">
          {pricing.note}
        </p>
      ) : null}
    </section>
  )
}
