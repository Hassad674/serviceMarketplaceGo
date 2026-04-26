import type { Metadata } from "next"
import { CurrentMonthAggregate } from "@/features/invoicing/components/current-month-aggregate"
import { InvoiceList } from "@/features/invoicing/components/invoice-list"

export const metadata: Metadata = {
  title: "Mes factures",
}

export default function InvoicesPage() {
  return (
    <div className="mx-auto max-w-4xl space-y-6 px-4 py-8">
      <header>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
          Mes factures
        </h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          Historique des factures émises par la plateforme et suivi des
          commissions du mois en cours.
        </p>
      </header>
      <CurrentMonthAggregate />
      <InvoiceList />
    </div>
  )
}
