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
      className="bg-card border border-border rounded-2xl p-7 shadow-[var(--shadow-card)]"
    >
      <header className="mb-4">
        <h2
          id={`public-pricing-title-${titleKey}`}
          className="font-serif text-xl font-medium tracking-[-0.005em] text-foreground"
        >
          {t(titleKey)}
        </h2>
      </header>

      <div className="flex items-baseline flex-wrap gap-3">
        <span className="font-serif text-[30px] font-medium leading-none tracking-[-0.02em] text-foreground">
          {formatPricing(pricing, locale)}
        </span>
        {pricing.negotiable ? (
          <span className="inline-flex items-center rounded-full border border-success bg-success-soft px-3 py-1 text-xs font-semibold text-success">
            {t("negotiableYes")}
          </span>
        ) : null}
      </div>

      {pricing.note.trim() !== "" ? (
        <p className="mt-3 font-serif text-sm italic text-muted-foreground">
          {pricing.note}
        </p>
      ) : null}
    </section>
  )
}
