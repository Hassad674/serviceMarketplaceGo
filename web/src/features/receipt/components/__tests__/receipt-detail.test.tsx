import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { ReceiptDetail } from "../receipt-detail"
import type { Receipt } from "../../types"

const mockGetReceipt = vi.fn()

vi.mock("../../api/receipt-api", async () => {
  const actual =
    await vi.importActual<typeof import("../../api/receipt-api")>(
      "../../api/receipt-api",
    )
  return {
    ...actual,
    getReceipt: (...args: unknown[]) => mockGetReceipt(...args),
  }
})

function renderDetail({ receiptId }: { receiptId: string | null }) {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return render(
    <NextIntlClientProvider locale="fr" messages={messages}>
      <QueryClientProvider client={client}>
        <ReceiptDetail receiptId={receiptId} onClose={vi.fn()} />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

const fullReceipt: Receipt = {
  id: "rec-1",
  payment_record_id: "pay-1",
  proposal_id: "prop-1",
  milestone_id: "mile-1",
  amount_cents: 50000,
  currency: "EUR",
  created_at: "2026-04-15T10:00:00Z",
  client: {
    organization_id: "org-client",
    name: "Acme Corp",
    siret: "123456789",
    vat: "FR12345678901",
    address_line1: "1 rue de la Paix",
    address_line2: "",
    city: "Paris",
    postal_code: "75001",
    country: "France",
  },
  provider: {
    organization_id: "org-provider",
    name: "Studio Indie",
    siret: "",
    vat: "",
    address_line1: "",
    address_line2: "",
    city: "",
    postal_code: "",
    country: "",
  },
  referrer: {
    organization_id: "org-ref",
    name: "Apporteur SARL",
    siret: "",
    vat: "",
    address_line1: "",
    address_line2: "",
    city: "",
    postal_code: "",
    country: "",
  },
  referrer_commission_amount_cents: 5000,
  snapshot_available: true,
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("ReceiptDetail", () => {
  it("renders nothing when receiptId is null (modal closed)", () => {
    renderDetail({ receiptId: null })
    expect(screen.queryByText(/Détail du reçu/i)).not.toBeInTheDocument()
  })

  it("renders the full snapshot when data is available", async () => {
    mockGetReceipt.mockResolvedValue(fullReceipt)
    renderDetail({ receiptId: "rec-1" })

    await waitFor(() =>
      expect(screen.getByText("Acme Corp")).toBeInTheDocument(),
    )
    expect(screen.getByText("Studio Indie")).toBeInTheDocument()
    expect(screen.getByText("Apporteur SARL")).toBeInTheDocument()
    expect(screen.getByText(/SIRET 123456789/i)).toBeInTheDocument()
    expect(screen.getByText("FR12345678901")).toBeInTheDocument()
    // Commission line for the referrer block
    expect(screen.getByText(/Commission/)).toBeInTheDocument()
  })

  it("shows the snapshot-missing banner when snapshot_available is false", async () => {
    mockGetReceipt.mockResolvedValue({
      ...fullReceipt,
      snapshot_available: false,
    })
    renderDetail({ receiptId: "rec-1" })

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument()
    })
  })

  it("renders the error placeholder when the query fails", async () => {
    mockGetReceipt.mockRejectedValue(new Error("boom"))
    renderDetail({ receiptId: "rec-1" })
    await waitFor(
      () =>
        expect(
          screen.getByText(/Impossible de charger le détail du reçu/i),
        ).toBeInTheDocument(),
      { timeout: 5_000 },
    )
  })
})
