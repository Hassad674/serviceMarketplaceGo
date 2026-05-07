"use client"

import { Download, Loader2, AlertTriangle } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { Modal } from "@/shared/components/ui/modal"
import { cn, formatCurrency, formatDate } from "@/shared/lib/utils"
import { getReceiptPdfUrl } from "../api/receipt-api"
import { useReceipt } from "../hooks/use-receipts"
import type { Receipt, ReceiptParty, ReceiptPdfLanguage } from "../types"

/**
 * Soleil v2 receipt-detail modal.
 *
 * Renders the full snapshot of a receipt: client billing block,
 * provider billing block, optional referrer block, total amount,
 * referrer commission share, and proposal/milestone IDs. Includes a
 * "Télécharger PDF" CTA that opens the localized PDF in a new tab.
 *
 * The component is a thin presenter: data lives in `useReceipt(id)`
 * which is keyed by the modal-controlled `receiptId`. Closing the
 * modal returns control to the list and the query stays cached.
 */
type ReceiptDetailProps = {
  receiptId: string | null
  onClose: () => void
}

export function ReceiptDetail({ receiptId, onClose }: ReceiptDetailProps) {
  const t = useTranslations("receipts")
  const open = receiptId !== null
  const { data, isLoading, isError } = useReceipt(receiptId)

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t("detailTitle")}
      maxWidthClassName="max-w-2xl"
    >
      <ReceiptDetailBody isLoading={isLoading} isError={isError} data={data} />
    </Modal>
  )
}

type ReceiptDetailBodyProps = {
  isLoading: boolean
  isError: boolean
  data: Receipt | undefined
}

function ReceiptDetailBody({ isLoading, isError, data }: ReceiptDetailBodyProps) {
  const t = useTranslations("receipts")
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-10 text-muted-foreground">
        <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
      </div>
    )
  }
  if (isError || !data) {
    return (
      <p className="rounded-2xl bg-amber-soft p-4 text-[14px] text-foreground">
        {t("detailError")}
      </p>
    )
  }
  return <ReceiptDetailContent receipt={data} />
}

function ReceiptDetailContent({ receipt }: { receipt: Receipt }) {
  const t = useTranslations("receipts")
  const locale = useLocale()
  const pdfLang: ReceiptPdfLanguage = locale === "en" ? "en" : "fr"

  return (
    <div className="space-y-5">
      {!receipt.snapshot_available ? <SnapshotMissingBanner /> : null}

      <header className="flex items-start justify-between gap-4">
        <div>
          <p className="font-mono text-[10px] font-bold uppercase tracking-[0.12em] text-primary">
            {t("detailEyebrow")}
          </p>
          <p className="mt-1 font-serif text-[22px] font-medium tracking-[-0.01em] text-foreground">
            {formatCurrency(receipt.amount_cents / 100)}
          </p>
          <p className="mt-0.5 text-[12.5px] text-muted-foreground">
            {t("detailIssuedOn", { date: formatDate(receipt.created_at) })}
          </p>
        </div>
        <a
          href={getReceiptPdfUrl(receipt.id, pdfLang)}
          target="_blank"
          rel="noopener noreferrer"
          className={cn(
            "inline-flex shrink-0 items-center gap-1.5 rounded-full border border-border-strong bg-card px-3 py-1.5",
            "text-[12px] font-semibold text-foreground transition-colors hover:border-primary hover:text-primary",
          )}
          aria-label={t("downloadPdfLabel")}
        >
          <Download className="h-3.5 w-3.5" aria-hidden="true" />
          {t("downloadPdf")}
        </a>
      </header>

      <PartyBlock
        title={t("partyClient")}
        party={receipt.client}
        fallbackKey="partyMissingClient"
      />
      <PartyBlock
        title={t("partyProvider")}
        party={receipt.provider}
        fallbackKey="partyMissingProvider"
      />
      {receipt.referrer ? (
        <PartyBlock
          title={t("partyReferrer")}
          party={receipt.referrer}
          extra={
            receipt.referrer_commission_amount_cents > 0
              ? `${t("referrerCommissionLabel")} : ${formatCurrency(
                  receipt.referrer_commission_amount_cents / 100,
                )}`
              : null
          }
        />
      ) : null}

      <ReferenceBlock receipt={receipt} />
    </div>
  )
}

function SnapshotMissingBanner() {
  const t = useTranslations("receipts")
  return (
    <div
      role="alert"
      className="flex items-start gap-3 rounded-2xl bg-amber-soft p-4 text-[13px] text-foreground"
    >
      <AlertTriangle
        className="mt-0.5 h-4 w-4 shrink-0 text-foreground"
        aria-hidden="true"
        strokeWidth={1.6}
      />
      <div>
        <p className="font-semibold">{t("snapshotMissingTitle")}</p>
        <p className="mt-0.5 text-muted-foreground">
          {t("snapshotMissingBody")}
        </p>
      </div>
    </div>
  )
}

type PartyBlockProps = {
  title: string
  party: ReceiptParty | null
  /** Translation key used when `party` is null. */
  fallbackKey?: "partyMissingClient" | "partyMissingProvider"
  /** Optional inline footer (used for the referrer's commission). */
  extra?: string | null
}

function PartyBlock({ title, party, fallbackKey, extra }: PartyBlockProps) {
  const t = useTranslations("receipts")
  return (
    <section className="rounded-2xl border border-border bg-background p-4">
      <p className="font-mono text-[10px] font-bold uppercase tracking-[0.12em] text-primary">
        {title}
      </p>
      {party ? (
        <div className="mt-2 space-y-0.5 text-[13.5px] text-foreground">
          <p className="font-semibold">{party.name}</p>
          {party.siret ? (
            <p className="font-mono text-[11.5px] tracking-tight text-subtle-foreground">
              SIRET {party.siret}
            </p>
          ) : null}
          {party.vat ? (
            <p className="font-mono text-[11.5px] tracking-tight text-subtle-foreground">
              {party.vat}
            </p>
          ) : null}
          {hasAddress(party) ? <AddressLines party={party} /> : null}
          {extra ? (
            <p className="mt-2 text-[12.5px] text-muted-foreground">{extra}</p>
          ) : null}
        </div>
      ) : (
        <p className="mt-2 text-[13px] italic text-muted-foreground">
          {fallbackKey ? t(fallbackKey) : t("partyMissingGeneric")}
        </p>
      )}
    </section>
  )
}

function AddressLines({ party }: { party: ReceiptParty }) {
  return (
    <div className="text-[12.5px] text-muted-foreground">
      {party.address_line1 ? <p>{party.address_line1}</p> : null}
      {party.address_line2 ? <p>{party.address_line2}</p> : null}
      {(party.postal_code || party.city) ? (
        <p>
          {[party.postal_code, party.city].filter(Boolean).join(" ")}
        </p>
      ) : null}
      {party.country ? <p>{party.country}</p> : null}
    </div>
  )
}

function hasAddress(party: ReceiptParty): boolean {
  return Boolean(
    party.address_line1 ||
      party.address_line2 ||
      party.city ||
      party.postal_code ||
      party.country,
  )
}

function ReferenceBlock({ receipt }: { receipt: Receipt }) {
  const t = useTranslations("receipts")
  return (
    <section className="rounded-2xl border border-border bg-background p-4 text-[12.5px] text-muted-foreground">
      <p className="font-mono text-[10px] font-bold uppercase tracking-[0.12em] text-primary">
        {t("referencesTitle")}
      </p>
      <dl className="mt-2 grid grid-cols-1 gap-1 sm:grid-cols-2">
        <ReferenceLine
          label={t("referencePaymentRecord")}
          value={receipt.payment_record_id}
        />
        {receipt.proposal_id ? (
          <ReferenceLine
            label={t("referenceProposal")}
            value={receipt.proposal_id}
          />
        ) : null}
        {receipt.milestone_id ? (
          <ReferenceLine
            label={t("referenceMilestone")}
            value={receipt.milestone_id}
          />
        ) : null}
        <ReferenceLine label={t("referenceReceipt")} value={receipt.id} />
      </dl>
    </section>
  )
}

function ReferenceLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <dt className="text-[11px] uppercase tracking-wide text-subtle-foreground">
        {label}
      </dt>
      <dd className="font-mono text-[11.5px] text-foreground break-all">
        {value}
      </dd>
    </div>
  )
}
