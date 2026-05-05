import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { CurrentMonthAggregate } from "@/features/invoicing/components/current-month-aggregate"
import { InvoiceList } from "@/features/invoicing/components/invoice-list"

// W-19 · Factures — Soleil v2.
// Editorial header (corail mono eyebrow + Fraunces serif title with
// italic corail accent + tabac subtitle) wraps the existing
// CurrentMonthAggregate (running commissions) and InvoiceList
// (paginated archive). The page stays a Server Component — only the
// two child sections need client interactivity.
//
// Out-of-scope flagged (NOT shipped this batch — features absent from repo):
//   - Filter pills (sent / received / all): the /api/v1/me/invoices
//     endpoint exposes no status/direction filter; rendering a pill
//     would require a new hook. SKIP+FLAG per design/rules.md §3.

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
      <CurrentMonthAggregate />
      <InvoiceList />
    </div>
  )
}
