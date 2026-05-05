"use client"

import { Download, FileText, Loader2 } from "lucide-react"
import { cn, formatCurrency } from "@/shared/lib/utils"
import { getInvoicePDFURL } from "../api/invoicing-api"
import { useInvoices } from "../hooks/use-invoices"
import type { Invoice, InvoiceSourceType } from "../types"

import { Button } from "@/shared/components/ui/button"

/**
 * W-19 — Soleil v2 invoice list.
 *
 * Cursor-paginated list of invoices issued to the current org. Each
 * row pairs a corail-soft icon plate, the Geist Mono invoice number,
 * a relative date in tabac, the amount in Geist Mono semibold, a
 * source-type pill, and a download icon button. The empty state and
 * skeleton mirror the editorial Soleil card grammar.
 *
 * The user-facing literal strings ("Aucune facture archivée",
 * "Abonnement Premium", "Commission mensuelle", "Voir plus",
 * "Télécharger PDF") are intentionally kept as JSX literals here.
 * The existing test (`__tests__/invoice-list.test.tsx`, OFF-LIMITS for
 * this batch) renders the component without a `NextIntlClientProvider`
 * and queries those literals via `getByText`/`getByRole`. Wiring
 * `useTranslations` here would crash the test (next-intl 4 throws when
 * a hook runs outside an `IntlProvider`). A follow-up batch can swap
 * them to `invoicesList.*` keys once the test wraps the component in a
 * provider — see design/rules.md §7.
 */
export function InvoiceList() {
  const { data, isLoading, isError, hasMore, loadMore, isFetching } =
    useInvoices()

  if (isLoading) return <ListSkeleton />
  if (isError || !data) {
    return (
      <p
        className="rounded-2xl border border-border bg-card p-6 text-[14px] text-muted-foreground"
        style={{ boxShadow: "var(--shadow-card)" }}
      >
        Impossible de charger les factures.
      </p>
    )
  }

  const items = data.data
  if (items.length === 0) return <EmptyState />

  return (
    <section
      className="overflow-hidden rounded-2xl border border-border bg-card"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <header className="border-b border-border px-6 py-4">
        <p className="font-mono text-[10px] font-bold uppercase tracking-[0.12em] text-primary">
          Toutes les factures
        </p>
        <p className="mt-1 text-[13px] text-muted-foreground">
          Toutes les factures émises par la plateforme à ton organisation
        </p>
      </header>
      <ul role="list" className="divide-y divide-border">
        {items.map((invoice) => (
          <li key={invoice.id}>
            <InvoiceRow invoice={invoice} />
          </li>
        ))}
      </ul>
      {hasMore && (
        <div className="flex justify-center border-t border-border px-6 py-4">
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={loadMore}
            disabled={isFetching}
            className={cn(
              "inline-flex items-center gap-2 rounded-full border border-border-strong bg-card px-5 py-2",
              "text-[13px] font-semibold text-foreground transition-colors hover:border-primary",
              "disabled:opacity-50",
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
  const status = sourceStatus(invoice.source_type)
  return (
    <div className="flex items-center gap-4 px-6 py-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary-soft text-primary">
        <FileText className="h-4 w-4" strokeWidth={1.6} aria-hidden="true" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center rounded-full bg-background px-2 py-0.5 font-mono text-[11px] font-semibold tracking-tight text-foreground">
            {invoice.number}
          </span>
          <span
            className={cn(
              "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-semibold",
              status.pillClass,
            )}
          >
            {status.label}
          </span>
        </div>
        <p className="mt-1 truncate text-[13.5px] text-foreground">
          {sourceLabel(invoice.source_type)}
        </p>
        <p className="mt-0.5 font-mono text-[11.5px] uppercase tracking-wide text-subtle-foreground">
          {formatRelativeDate(invoice.issued_at)}
        </p>
      </div>
      <p className="shrink-0 font-mono text-[15px] font-semibold tracking-tight text-foreground">
        {formatCurrency(invoice.amount_incl_tax_cents / 100)}
      </p>
      <a
        href={getInvoicePDFURL(invoice.id)}
        // The backend signs the presigned URL with
        // `Content-Disposition: attachment; filename=<number>.pdf`,
        // which forces the browser to save the file. The `download`
        // attribute is defense-in-depth (browsers ignore it for
        // cross-origin links, so the server header is the actual fix).
        download={`${invoice.number}.pdf`}
        rel="noopener noreferrer"
        className={cn(
          "inline-flex shrink-0 items-center gap-1.5 rounded-full border border-border-strong bg-card px-3 py-1.5",
          "text-[12px] font-semibold text-foreground transition-colors hover:border-primary hover:text-primary",
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

type StatusPill = { label: string; pillClass: string }

function sourceStatus(source: InvoiceSourceType): StatusPill {
  switch (source) {
    case "subscription":
      // Subscription invoices are issued only after a successful
      // Stripe payment, so they are always "Payée".
      return {
        label: "Payée",
        pillClass: "bg-success-soft text-success",
      }
    case "monthly_commission":
      // Monthly commission invoices are emitted on the 1st but paid
      // later by SEPA debit; surface them as "En attente" until the
      // backend exposes a real `status` field.
      return {
        label: "En attente",
        pillClass: "bg-amber-soft text-foreground",
      }
    case "credit_note":
      return {
        label: "Avoir",
        pillClass: "bg-border text-muted-foreground",
      }
    default:
      return {
        label: "Brouillon",
        pillClass: "bg-border text-muted-foreground",
      }
  }
}

/**
 * Conversational FR relative date for a timestamp.
 *
 * The spec ("dates FR conversationnelles") wants short labels like
 * "à l'instant" / "il y a 1h" / "ce matin" / "hier" rather than the
 * verbose `1 avril 2026` `formatDate` output. Older issuance dates
 * fall back to a short `1 avr.` style.
 */
function formatRelativeDate(iso: string): string {
  const date = new Date(iso)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60_000)
  if (diffMin < 1) return "à l'instant"
  if (diffMin < 60) return `il y a ${diffMin} min`
  const diffHours = Math.floor(diffMin / 60)
  if (diffHours < 24 && date.toDateString() === now.toDateString()) {
    return `il y a ${diffHours} h`
  }
  const yesterday = new Date(now)
  yesterday.setDate(yesterday.getDate() - 1)
  if (date.toDateString() === yesterday.toDateString()) return "hier"
  const diffDays = Math.floor(diffMs / 86_400_000)
  if (diffDays < 7) return `il y a ${diffDays} j`
  return date.toLocaleDateString("fr-FR", { day: "numeric", month: "short" })
}

function EmptyState() {
  return (
    <div
      className="flex flex-col items-center gap-3 rounded-2xl border border-border bg-card px-6 py-16 text-center"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary-soft text-primary">
        <FileText className="h-5 w-5" strokeWidth={1.6} aria-hidden="true" />
      </div>
      <h2 className="font-serif text-[20px] font-medium leading-tight tracking-[-0.01em] text-foreground">
        Aucune facture archivée
      </h2>
      <p className="max-w-md font-serif italic text-[14px] leading-relaxed text-muted-foreground">
        La facture consolidée des commissions du mois en cours sera émise
        automatiquement le 1er du mois suivant. Les factures d&apos;abonnement
        Premium apparaîtront dès le premier paiement.
      </p>
    </div>
  )
}

function ListSkeleton() {
  return (
    <section
      className="overflow-hidden rounded-2xl border border-border bg-card"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <ul role="list" className="divide-y divide-border">
        {[1, 2, 3].map((i) => (
          <li
            key={i}
            className="flex items-center gap-4 px-6 py-4"
          >
            <div className="h-10 w-10 shrink-0 rounded-2xl bg-border animate-shimmer" />
            <div className="flex-1 space-y-2">
              <div className="h-3.5 w-32 rounded-full bg-border animate-shimmer" />
              <div className="h-3 w-48 rounded-full bg-border animate-shimmer" />
            </div>
            <div className="h-3.5 w-16 rounded-full bg-border animate-shimmer" />
          </li>
        ))}
      </ul>
    </section>
  )
}
