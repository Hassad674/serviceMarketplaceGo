"use client"

import { Download, FileText, Loader2 } from "lucide-react"
import { cn, formatCurrency, formatDate } from "@/shared/lib/utils"
import { getInvoicePDFURL } from "../api/invoicing-api"
import { useInvoices } from "../hooks/use-invoices"
import type { Invoice, InvoiceSourceType } from "../types"

import { Button } from "@/shared/components/ui/button"
/**
 * Cursor-paginated list of invoices issued to the current org.
 * Empty state has its own visual treatment (icon + explainer)
 * rather than a bare "No data" line, to match the rest of the
 * dashboard surfaces (e.g. wallet history).
 */
export function InvoiceList() {
  const { data, isLoading, isError, hasMore, loadMore, isFetching } =
    useInvoices()

  if (isLoading) return <ListSkeleton />
  if (isError || !data) {
    return (
      <p className="rounded-2xl border border-slate-100 bg-white p-6 text-sm text-slate-500 shadow-sm dark:border-slate-700 dark:bg-slate-900">
        Impossible de charger les factures.
      </p>
    )
  }

  const items = data.data
  if (items.length === 0) return <EmptyState />

  return (
    <section className="overflow-hidden rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-900">
      <div className="border-b border-slate-100 px-5 py-3 dark:border-slate-700">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-white">
          Mes factures
        </h2>
        <p className="text-xs text-slate-500 dark:text-slate-400">
          Toutes les factures émises par la plateforme à ton organisation
        </p>
      </div>
      <ul
        role="list"
        className="divide-y divide-slate-100 dark:divide-slate-800"
      >
        {items.map((invoice) => (
          <li key={invoice.id}>
            <InvoiceRow invoice={invoice} />
          </li>
        ))}
      </ul>
      {hasMore && (
        <div className="flex justify-center border-t border-slate-100 p-3 dark:border-slate-700">
          <Button variant="ghost" size="auto"
            type="button"
            onClick={loadMore}
            disabled={isFetching}
            className={cn(
              "inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2",
              "text-xs font-medium text-slate-700 transition-colors hover:bg-slate-50",
              "disabled:opacity-50",
              "dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700",
            )}
          >
            {isFetching ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
            ) : null}
            Voir plus
          </Button>
        </div>
      )}
    </section>
  )
}

function InvoiceRow({ invoice }: { invoice: Invoice }) {
  return (
    <div className="flex items-center gap-4 px-5 py-3">
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-300">
        <FileText className="h-4 w-4" aria-hidden="true" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-slate-900 dark:text-white">
          {invoice.number}
        </p>
        <div className="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
          <span>{formatDate(invoice.issued_at)}</span>
          <span aria-hidden="true">·</span>
          <span>{sourceLabel(invoice.source_type)}</span>
        </div>
      </div>
      <p className="font-mono text-sm font-semibold text-slate-900 dark:text-white">
        {formatCurrency(invoice.amount_incl_tax_cents / 100)}
      </p>
      <a
        href={getInvoicePDFURL(invoice.id)}
        // The backend now signs the presigned URL with
        // `Content-Disposition: attachment; filename=<number>.pdf`,
        // which forces the browser to save the file. The `download`
        // attribute is defense-in-depth (browsers ignore it for
        // cross-origin links, so the server header is the actual fix).
        download={`${invoice.number}.pdf`}
        rel="noopener noreferrer"
        className={cn(
          "inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-1.5",
          "text-xs font-medium text-slate-700 transition-colors hover:bg-slate-50",
          "dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700",
        )}
        aria-label={`Télécharger la facture ${invoice.number} au format PDF`}
      >
        <Download className="h-3.5 w-3.5" aria-hidden="true" />
        Télécharger PDF
      </a>
    </div>
  )
}

function sourceLabel(source: InvoiceSourceType): string {
  switch (source) {
    case "subscription":
      return "Abonnement Premium"
    case "monthly_commission":
      return "Commission mensuelle"
    case "credit_note":
      return "Avoir"
    default:
      return source
  }
}

function EmptyState() {
  return (
    <div className="rounded-2xl border border-dashed border-slate-200 bg-white p-10 text-center dark:border-slate-700 dark:bg-slate-900">
      <FileText
        className="mx-auto h-10 w-10 text-slate-300 dark:text-slate-600"
        aria-hidden="true"
      />
      <h2 className="mt-3 text-base font-semibold text-slate-900 dark:text-white">
        Aucune facture archivée
      </h2>
      <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
        La facture consolidée des commissions du mois en cours sera émise
        automatiquement le 1er du mois suivant. Les factures d&apos;abonnement
        Premium apparaîtront dès le premier paiement.
      </p>
    </div>
  )
}

function ListSkeleton() {
  return (
    <div className="space-y-3">
      {[1, 2, 3].map((i) => (
        <div
          key={i}
          className="h-16 animate-shimmer rounded-xl bg-slate-100 dark:bg-slate-800"
        />
      ))}
    </div>
  )
}
