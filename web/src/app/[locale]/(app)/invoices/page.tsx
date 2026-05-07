import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { InvoicesTabs } from "./invoices-tabs"

// W-19 · Factures + Reçus — Soleil v2.
// Editorial header (corail mono eyebrow + Fraunces serif title with
// italic corail accent + tabac subtitle) wraps a tab switcher
// (URL-driven via `?tab=invoices|receipts`).
//
// Tab "Factures" — existing platform invoices: CurrentMonthAggregate
// (running commissions) + InvoiceList (paginated archive).
// Tab "Reçus" — new transaction-receipt list backed by
// `/api/v1/receipts*` (PR #165).
//
// Composition stays in this page file (per web/CLAUDE.md "features
// never import other features"). The `InvoicesTabs` client wrapper
// is colocated in this route directory because it imports from BOTH
// the invoicing and receipt features and is therefore not feature-scoped.

export const metadata: Metadata = {
  title: "Mes factures",
}

export default async function InvoicesPage() {
  const t = await getTranslations("invoicesList")

  return (
    <div className="mx-auto w-full max-w-4xl space-y-8 px-4 py-8 sm:px-6">
      <header className="space-y-3">
        <p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
          {t("eyebrow")}
        </p>
        <h1 className="font-serif text-[34px] font-normal leading-[1.1] tracking-[-0.025em] text-foreground sm:text-[42px]">
          {t("heroTitlePart1")}{" "}
          <span className="italic text-primary">{t("heroTitleAccent")}</span>
        </h1>
        <p className="max-w-xl text-[15px] leading-relaxed text-muted-foreground">
          {t("heroSubtitle")}
        </p>
      </header>
      <InvoicesTabs />
    </div>
  )
}
