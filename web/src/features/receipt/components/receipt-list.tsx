"use client"

import { useState } from "react"
import { Download, Loader2, ReceiptText } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { Button } from "@/shared/components/ui/button"
import { cn, formatCurrency } from "@/shared/lib/utils"
import { getReceiptPdfUrl } from "../api/receipt-api"
import { useReceipts } from "../hooks/use-receipts"
import type { Receipt, ReceiptPdfLanguage } from "../types"
import { ReceiptDetail } from "./receipt-detail"
import { ReceiptEmptyState } from "./receipt-empty-state"

/**
 * Soleil v2 receipts list. Mirrors the editorial grammar of
 * `InvoiceList` (corail-soft icon plate, Geist Mono identifiers,
 * tabac dates, full-pill download CTA) so the two tabs feel like
 * siblings.
 *
 * Behavior:
 * - Cursor-paginated via `useReceipts()` — "Voir plus" appends the
 *   next page in place; the page is intentionally truncated to one
 *   page in V1 to keep the UI calm.
 * - Each row exposes a "Voir détails" button that opens
 *   `ReceiptDetail` (modal) — the modal is keyed by the active
 *   receipt id, which lives in component-local state.
 * - The "Télécharger PDF" link points at the localized PDF
 *   endpoint and opens in a new tab.
 * - When `snapshot_available === false`, the row carries a
 *   corail-soft "Reçu antérieur" badge.
 *
 * On small viewports (< sm) the table-like row collapses into a
 * stacked card so the column-based layout never overflows.
 */
export function ReceiptList() {
  const t = useTranslations("receipts")
  const { data, isLoading, isError, hasMore, loadMore, isFetching } =
    useReceipts()
  const [activeReceiptId, setActiveReceiptId] = useState<string | null>(null)

  if (isLoading) return <ListSkeleton />
  if (isError || !data) {
    return (
      <p className="rounded-2xl border border-border bg-card p-6 text-[14px] text-muted-foreground shadow-[var(--shadow-card)]">
        {t("loadError")}
      </p>
    )
  }

  const items = data.data
  if (items.length === 0) return <ReceiptEmptyState />

  return (
    <>
      <section className="overflow-hidden rounded-2xl border border-border bg-card shadow-[var(--shadow-card)]">
        <header className="border-b border-border px-6 py-4">
          <p className="font-mono text-[10px] font-bold uppercase tracking-[0.12em] text-primary">
            {t("listEyebrow")}
          </p>
          <p className="mt-1 text-[13px] text-muted-foreground">
            {t("listSubtitle")}
          </p>
        </header>
        <ul role="list" className="divide-y divide-border">
          {items.map((receipt) => (
            <li key={receipt.id}>
              <ReceiptRow
                receipt={receipt}
                onShowDetail={() => setActiveReceiptId(receipt.id)}
              />
            </li>
          ))}
        </ul>
        {hasMore ? <LoadMoreBar onClick={loadMore} loading={isFetching} /> : null}
      </section>
      <ReceiptDetail
        receiptId={activeReceiptId}
        onClose={() => setActiveReceiptId(null)}
      />
    </>
  )
}

function ReceiptRow({
  receipt,
  onShowDetail,
}: {
  receipt: Receipt
  onShowDetail: () => void
}) {
  const t = useTranslations("receipts")
  const locale = useLocale()
  const pdfLang: ReceiptPdfLanguage = locale === "en" ? "en" : "fr"
  const counterpartyName = pickCounterpartyName(receipt)

  return (
    <div className="flex flex-col gap-3 px-4 py-4 sm:flex-row sm:items-center sm:gap-4 sm:px-6">
      <div className="flex shrink-0 items-center gap-3 sm:gap-4">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary-soft text-primary">
          <ReceiptText
            className="h-4 w-4"
            strokeWidth={1.6}
            aria-hidden="true"
          />
        </div>
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center rounded-full bg-background px-2 py-0.5 font-mono text-[11px] font-semibold tracking-tight text-foreground">
            {formatShortId(receipt.id)}
          </span>
          {!receipt.snapshot_available ? (
            <span
              className="inline-flex items-center rounded-full bg-primary-soft px-2 py-0.5 text-[11px] font-semibold text-[var(--primary-deep)]"
              title={t("snapshotMissingBody")}
            >
              {t("snapshotMissingBadge")}
            </span>
          ) : null}
        </div>
        <p className="mt-1 truncate text-[13.5px] text-foreground">
          {counterpartyName ?? t("counterpartyUnknown")}
        </p>
        <p className="mt-0.5 font-mono text-[11.5px] uppercase tracking-wide text-subtle-foreground">
          {formatRelativeDate(receipt.created_at)}
        </p>
      </div>
      <p className="shrink-0 font-mono text-[15px] font-semibold tracking-tight text-foreground sm:text-right">
        {formatCurrency(receipt.amount_cents / 100)}
      </p>
      <div className="flex shrink-0 flex-wrap items-center gap-2">
        <Button
          variant="ghost"
          size="auto"
          type="button"
          onClick={onShowDetail}
          className={cn(
            "inline-flex items-center gap-1.5 rounded-full border border-border-strong bg-card px-3 py-1.5",
            "text-[12px] font-semibold text-foreground transition-colors hover:border-primary hover:text-primary",
          )}
        >
          {t("viewDetails")}
        </Button>
        <a
          href={getReceiptPdfUrl(receipt.id, pdfLang)}
          target="_blank"
          rel="noopener noreferrer"
          className={cn(
            "inline-flex items-center gap-1.5 rounded-full border border-border-strong bg-card px-3 py-1.5",
            "text-[12px] font-semibold text-foreground transition-colors hover:border-primary hover:text-primary",
          )}
          aria-label={t("downloadPdfLabel")}
        >
          <Download className="h-3.5 w-3.5" aria-hidden="true" />
          {t("downloadPdf")}
        </a>
      </div>
    </div>
  )
}

function LoadMoreBar({
  onClick,
  loading,
}: {
  onClick: () => void
  loading: boolean
}) {
  const t = useTranslations("receipts")
  return (
    <div className="flex justify-center border-t border-border px-6 py-4">
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={onClick}
        disabled={loading}
        className={cn(
          "inline-flex items-center gap-2 rounded-full border border-border-strong bg-card px-5 py-2",
          "text-[13px] font-semibold text-foreground transition-colors hover:border-primary",
          "disabled:opacity-50",
        )}
      >
        {loading ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
        ) : null}
        {t("loadMore")}
      </Button>
    </div>
  )
}

function ListSkeleton() {
  return (
    <section className="overflow-hidden rounded-2xl border border-border bg-card shadow-[var(--shadow-card)]">
      <ul role="list" className="divide-y divide-border">
        {[1, 2, 3].map((i) => (
          <li key={i} className="flex items-center gap-4 px-6 py-4">
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

/**
 * Picks the counterparty's display name for the row.
 *
 * On a receipt the user belongs to one of the parties (client OR
 * provider OR referrer). The list view shows the *other* side of
 * the transaction so the user immediately recognises who paid them
 * (or who they paid). Without org-side context we fall back to the
 * client name first (most common case in invoicing UI).
 */
function pickCounterpartyName(receipt: Receipt): string | null {
  return (
    receipt.client?.name ||
    receipt.provider?.name ||
    receipt.referrer?.name ||
    null
  )
}

/**
 * Conversational FR relative date. Mirrors the helper from
 * `invoice-list.tsx` so the two tabs share a single date language.
 * Kept as a small inline copy (rule of three not yet reached) — a
 * shared util will be extracted at the next occurrence.
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

/** First 8 chars of the UUID rendered as a humane reference token. */
function formatShortId(id: string): string {
  const head = id.split("-")[0] ?? id
  return head.slice(0, 8).toUpperCase()
}
