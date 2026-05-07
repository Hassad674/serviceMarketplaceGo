import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { ReceiptList } from "../receipt-list"
import type { Receipt } from "../../types"

const mockListReceipts = vi.fn()
const mockGetReceipt = vi.fn()

vi.mock("../../api/receipt-api", async () => {
  const actual =
    await vi.importActual<typeof import("../../api/receipt-api")>(
      "../../api/receipt-api",
    )
  return {
    ...actual,
    listReceipts: (...args: unknown[]) => mockListReceipts(...args),
    getReceipt: (...args: unknown[]) => mockGetReceipt(...args),
  }
})

function renderList() {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return render(
    <NextIntlClientProvider locale="fr" messages={messages}>
      <QueryClientProvider client={client}>
        <ReceiptList />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

const baseReceipt: Receipt = {
  id: "11111111-2222-3333-4444-555555555555",
  payment_record_id: "pay-1",
  proposal_id: "prop-1",
  milestone_id: "mile-1",
  amount_cents: 12000,
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
    city: "Lyon",
    postal_code: "69000",
    country: "France",
  },
  referrer: null,
  referrer_commission_amount_cents: 0,
  snapshot_available: true,
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("ReceiptList", () => {
  it("renders the empty state when the API returns no items", async () => {
    mockListReceipts.mockResolvedValue({ data: [] })
    renderList()
    await waitFor(() =>
      expect(screen.getByText(/Aucun reçu/i)).toBeInTheDocument(),
    )
  })

  it("renders one row per receipt with the counterparty name", async () => {
    mockListReceipts.mockResolvedValue({ data: [baseReceipt] })
    renderList()
    await waitFor(() =>
      expect(screen.getByText("Acme Corp")).toBeInTheDocument(),
    )
    expect(screen.getByText(/120,00/)).toBeInTheDocument()
  })

  it("shows the legacy badge when snapshot_available is false", async () => {
    mockListReceipts.mockResolvedValue({
      data: [{ ...baseReceipt, snapshot_available: false }],
    })
    renderList()
    await waitFor(() =>
      expect(
        screen.getByText(/Reçu antérieur — données indisponibles/i),
      ).toBeInTheDocument(),
    )
  })

  it("renders a PDF link pointing at the FR endpoint", async () => {
    mockListReceipts.mockResolvedValue({ data: [baseReceipt] })
    renderList()
    const pdfLink = await screen.findByRole("link", {
      name: /Télécharger le reçu au format PDF/i,
    })
    expect(pdfLink).toHaveAttribute(
      "href",
      expect.stringContaining(`/api/v1/receipts/${baseReceipt.id}/pdf?lang=fr`),
    )
    expect(pdfLink).toHaveAttribute("target", "_blank")
    expect(pdfLink).toHaveAttribute("rel", "noopener noreferrer")
  })

  it("opens the detail modal when 'Voir détails' is clicked", async () => {
    const user = userEvent.setup()
    mockListReceipts.mockResolvedValue({ data: [baseReceipt] })
    mockGetReceipt.mockResolvedValue(baseReceipt)
    renderList()
    await screen.findByText("Acme Corp")

    await user.click(screen.getByRole("button", { name: /Voir détails/i }))

    await waitFor(() => {
      // Modal title is `detailTitle`. Using a regex tolerant to FR copy.
      expect(screen.getByText(/Détail du reçu/i)).toBeInTheDocument()
    })
    expect(mockGetReceipt).toHaveBeenCalledWith(baseReceipt.id)
  })

  it("renders the load-more button when next_cursor is present", async () => {
    mockListReceipts.mockResolvedValue({
      data: [baseReceipt],
      next_cursor: "next-1",
    })
    renderList()
    await screen.findByText("Acme Corp")
    expect(
      screen.getByRole("button", { name: /Voir plus/i }),
    ).toBeInTheDocument()
  })

  it("renders an error message when the query fails", async () => {
    mockListReceipts.mockRejectedValue(new Error("boom"))
    renderList()
    await waitFor(
      () =>
        expect(
          screen.getByText(/Impossible de charger les reçus/i),
        ).toBeInTheDocument(),
      { timeout: 5_000 },
    )
  })
})
