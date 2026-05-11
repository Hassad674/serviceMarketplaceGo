"use client"

import { Pencil, Receipt } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { BillingProfile } from "@/shared/types/billing-profile"

/**
 * Compact read-only summary of the saved BillingProfile, rendered on
 * the payment page when the profile is already complete. Mirrors the
 * editable form's section structure (pays, adresse, entité légale,
 * identifiants fiscaux) but as flat lines without inputs.
 *
 * A single "Modifier" CTA hands control back to the parent which then
 * swaps the summary for the full BillingProfileForm.
 *
 * Soleil v2 styling: ivoire surface card (rounded-2xl), sable border,
 * Geist Mono used only for the SIRET / VAT identifiers, Inter Tight
 * for labels + addresses.
 *
 * Strings come from the `proposal.billingEmbed.*` i18n bag — they are
 * NOT hardcoded so the page reads as natural French / English. The
 * empty-value fallback ("—") is locale-agnostic so it can stay inline.
 */

export type BillingProfileSummaryProps = {
  profile: BillingProfile
  onEdit: () => void
}

export function BillingProfileSummary({
  profile,
  onEdit,
}: BillingProfileSummaryProps) {
  const t = useTranslations("proposal.billingEmbed")

  const showTax =
    profile.profile_type === "business" || profile.tax_id || profile.vat_number

  return (
    <section
      aria-label={t("summaryTitle")}
      className={cn(
        "rounded-2xl border border-border bg-surface p-4 space-y-3",
      )}
    >
      <header className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Receipt
            className="h-4 w-4 text-primary"
            strokeWidth={1.7}
            aria-hidden="true"
          />
          <p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
            {t("summaryTitle")}
          </p>
        </div>
        <button
          type="button"
          onClick={onEdit}
          className={cn(
            "inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-[12px] font-medium",
            "text-muted-foreground hover:text-primary transition-colors duration-150",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40",
          )}
        >
          <Pencil className="h-3.5 w-3.5" strokeWidth={1.7} aria-hidden="true" />
          {t("editCta")}
        </button>
      </header>

      <dl className="grid grid-cols-1 gap-2 text-[13px] sm:grid-cols-2">
        <SummaryRow label={t("entity")}>
          <span className="font-medium text-foreground">
            {profile.legal_name || "—"}
          </span>
          {profile.trading_name && (
            <span className="block text-muted-foreground">
              {profile.trading_name}
            </span>
          )}
        </SummaryRow>

        <SummaryRow label={t("country")}>
          <span className="text-foreground">
            {profile.country || "—"}
          </span>
        </SummaryRow>

        <SummaryRow label={t("address")} className="sm:col-span-2">
          <span className="text-foreground">
            {[
              profile.address_line1,
              profile.address_line2,
              [profile.postal_code, profile.city].filter(Boolean).join(" "),
            ]
              .filter(Boolean)
              .join(", ") || "—"}
          </span>
        </SummaryRow>

        {showTax && (
          <SummaryRow label={t("tax")} className="sm:col-span-2">
            <span className="font-mono text-[12.5px] text-foreground">
              {profile.tax_id || "—"}
              {profile.vat_number && (
                <span className="ml-2 text-muted-foreground">
                  · {profile.vat_number}
                </span>
              )}
            </span>
          </SummaryRow>
        )}
      </dl>
    </section>
  )
}

function SummaryRow({
  label,
  className,
  children,
}: {
  label: string
  className?: string
  children: React.ReactNode
}) {
  return (
    <div className={cn("space-y-0.5", className)}>
      <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </dt>
      <dd>{children}</dd>
    </div>
  )
}
