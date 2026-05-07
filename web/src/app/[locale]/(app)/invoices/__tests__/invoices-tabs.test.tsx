import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/fr.json"
import { InvoicesTabs } from "../invoices-tabs"

// Mock the search-params + router so we can drive the tab without
// the Next router runtime. Pattern matches
// `billing-profile-completion-modal.test.tsx`.
const mockReplace = vi.fn()
const tabValueRef = { current: "" as string | null }

vi.mock("next/navigation", () => ({
  useSearchParams: () => ({
    get: (key: string) => (key === "tab" ? tabValueRef.current : null),
  }),
}))

vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ replace: mockReplace }),
}))

// Avoid hitting the network for the embedded child components — they
// each make a TanStack Query call on mount, which would otherwise
// fail the test deterministically. We mock both feature APIs at the
// module level.
vi.mock("@/features/invoicing/api/invoicing-api", async () => {
  const actual = await vi.importActual<
    typeof import("@/features/invoicing/api/invoicing-api")
  >("@/features/invoicing/api/invoicing-api")
  return {
    ...actual,
    fetchInvoices: vi.fn(async () => ({ data: [] })),
  }
})

// CurrentMonthAggregate's hook reads from the shared billing-profile
// module — mock that one too so the embedded card doesn't render an
// error state during tab switches.
vi.mock("@/shared/lib/billing-profile/billing-profile-api", async () => {
  const actual = await vi.importActual<
    typeof import("@/shared/lib/billing-profile/billing-profile-api")
  >("@/shared/lib/billing-profile/billing-profile-api")
  return {
    ...actual,
    fetchCurrentMonthAggregate: vi.fn(async () => ({
      period_start: "2026-04-01",
      period_end: "2026-04-30",
      total_commission_cents: 0,
      lines: [],
    })),
  }
})

vi.mock("@/features/receipt/api/receipt-api", async () => {
  const actual = await vi.importActual<
    typeof import("@/features/receipt/api/receipt-api")
  >("@/features/receipt/api/receipt-api")
  return {
    ...actual,
    listReceipts: vi.fn(async () => ({ data: [] })),
    getReceipt: vi.fn(),
  }
})

function renderTabs(initialTab: string | null = null) {
  tabValueRef.current = initialTab
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return render(
    <NextIntlClientProvider locale="fr" messages={messages}>
      <QueryClientProvider client={client}>
        <InvoicesTabs />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("InvoicesTabs", () => {
  it("defaults to the invoices tab when no `?tab=` is set", () => {
    renderTabs(null)
    const invoicesTab = screen.getByRole("tab", {
      name: /Mes factures plateforme/i,
    })
    const receiptsTab = screen.getByRole("tab", { name: /Reçus/i })
    expect(invoicesTab).toHaveAttribute("aria-selected", "true")
    expect(receiptsTab).toHaveAttribute("aria-selected", "false")
  })

  it("activates the receipts tab when `?tab=receipts`", () => {
    renderTabs("receipts")
    const receiptsTab = screen.getByRole("tab", { name: /Reçus/i })
    expect(receiptsTab).toHaveAttribute("aria-selected", "true")
  })

  it("falls back to the default tab when the param value is unknown", () => {
    renderTabs("garbage")
    const invoicesTab = screen.getByRole("tab", {
      name: /Mes factures plateforme/i,
    })
    expect(invoicesTab).toHaveAttribute("aria-selected", "true")
  })

  it("calls router.replace with the receipts query on click", async () => {
    const user = userEvent.setup()
    renderTabs(null)
    await user.click(screen.getByRole("tab", { name: /Reçus/i }))
    expect(mockReplace).toHaveBeenCalledWith("/invoices?tab=receipts")
  })

  it("strips the query string when navigating back to the default tab", async () => {
    const user = userEvent.setup()
    renderTabs("receipts")
    await user.click(
      screen.getByRole("tab", { name: /Mes factures plateforme/i }),
    )
    expect(mockReplace).toHaveBeenCalledWith("/invoices")
  })
})
