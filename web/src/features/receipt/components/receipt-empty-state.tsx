"use client"

import { ReceiptText } from "lucide-react"
import { useTranslations } from "next-intl"

/**
 * Soleil v2 empty state for the receipts tab. Mirrors the editorial
 * grammar of the sibling InvoiceList empty state: corail-soft icon
 * plate, Fraunces title, italic subtitle in tabac.
 */
export function ReceiptEmptyState() {
  const t = useTranslations("receipts")
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl border border-border bg-card px-6 py-16 text-center shadow-[var(--shadow-card)]">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary-soft text-primary">
        <ReceiptText className="h-5 w-5" strokeWidth={1.6} aria-hidden="true" />
      </div>
      <h2 className="font-serif text-[20px] font-medium leading-tight tracking-[-0.01em] text-foreground">
        {t("emptyTitle")}
      </h2>
      <p className="max-w-md font-serif italic text-[14px] leading-relaxed text-muted-foreground">
        {t("emptyBody")}
      </p>
    </div>
  )
}
