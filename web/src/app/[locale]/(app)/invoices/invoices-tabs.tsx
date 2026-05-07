"use client"

import { useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { CurrentMonthAggregate } from "@/features/invoicing/components/current-month-aggregate"
import { InvoiceList } from "@/features/invoicing/components/invoice-list"
import { ReceiptList } from "@/features/receipt/components/receipt-list"
import { Button } from "@/shared/components/ui/button"
import { cn } from "@/shared/lib/utils"

/**
 * URL-bookmarkable tab switcher between platform invoices and
 * transaction receipts. Mirrors the `?section=` pattern used by the
 * Account settings surface so the two areas feel consistent.
 *
 * Lives in the route directory rather than inside any feature folder
 * because composition is the page's job (per `web/CLAUDE.md` —
 * features never import other features). Importing both
 * `features/invoicing` and `features/receipt` here is the legitimate
 * composition point.
 */
const VALID_TABS = ["invoices", "receipts"] as const
type InvoicesTab = (typeof VALID_TABS)[number]
const DEFAULT_TAB: InvoicesTab = "invoices"

export function InvoicesTabs() {
  const t = useTranslations("invoicesTabs")
  const searchParams = useSearchParams()
  const router = useRouter()

  const rawTab = searchParams.get("tab") || DEFAULT_TAB
  const tab: InvoicesTab = (VALID_TABS as readonly string[]).includes(rawTab)
    ? (rawTab as InvoicesTab)
    : DEFAULT_TAB

  function handleTabChange(next: InvoicesTab) {
    if (next === DEFAULT_TAB) {
      router.replace("/invoices")
      return
    }
    router.replace(`/invoices?tab=${next}`)
  }

  return (
    <div className="space-y-6">
      <nav
        role="tablist"
        aria-label={t("navLabel")}
        className="inline-flex items-center gap-1 rounded-full border border-border bg-card p-1 shadow-[var(--shadow-card)]"
      >
        <TabButton
          isActive={tab === "invoices"}
          onClick={() => handleTabChange("invoices")}
          controls="panel-invoices"
          id="tab-invoices"
          label={t("invoicesTab")}
        />
        <TabButton
          isActive={tab === "receipts"}
          onClick={() => handleTabChange("receipts")}
          controls="panel-receipts"
          id="tab-receipts"
          label={t("receiptsTab")}
        />
      </nav>

      <div
        role="tabpanel"
        id="panel-invoices"
        aria-labelledby="tab-invoices"
        hidden={tab !== "invoices"}
        className={tab === "invoices" ? "space-y-8" : ""}
      >
        {tab === "invoices" ? (
          <>
            <CurrentMonthAggregate />
            <InvoiceList />
          </>
        ) : null}
      </div>
      <div
        role="tabpanel"
        id="panel-receipts"
        aria-labelledby="tab-receipts"
        hidden={tab !== "receipts"}
      >
        {tab === "receipts" ? <ReceiptList /> : null}
      </div>
    </div>
  )
}

type TabButtonProps = {
  isActive: boolean
  onClick: () => void
  controls: string
  id: string
  label: string
}

function TabButton({ isActive, onClick, controls, id, label }: TabButtonProps) {
  return (
    <Button
      variant="ghost"
      size="auto"
      type="button"
      role="tab"
      id={id}
      aria-controls={controls}
      aria-selected={isActive}
      onClick={onClick}
      className={cn(
        "inline-flex items-center rounded-full px-4 py-2 text-[13px] font-semibold transition-colors",
        isActive
          ? "bg-primary-soft text-[var(--primary-deep)] hover:bg-primary-soft"
          : "text-muted-foreground hover:text-foreground",
      )}
    >
      {label}
    </Button>
  )
}
