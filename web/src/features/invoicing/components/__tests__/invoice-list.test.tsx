import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { InvoiceList } from "../invoice-list"

const mockFetchInvoices = vi.fn()

vi.mock("../../api/invoicing-api", async () => {
  const actual = await vi.importActual<typeof import("../../api/invoicing-api")>(
    "../../api/invoicing-api",
  )
  return {
    ...actual,
    fetchInvoices: (...args: unknown[]) => mockFetchInvoices(...args),
  }
})

function withQueryClient(ui: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return createElement(QueryClientProvider, { client }, ui)
}

describe("InvoiceList", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders an empty state when no invoices are returned", async () => {
    mockFetchInvoices.mockResolvedValue({ data: [] })
    render(withQueryClient(<InvoiceList />))
    await waitFor(() =>
      expect(
        screen.getByText(/Aucune facture pour le moment/i),
      ).toBeInTheDocument(),
    )
  })

  it("renders invoice rows with the source label and a download link", async () => {
    mockFetchInvoices.mockResolvedValue({
      data: [
        {
          id: "inv-1",
          number: "FA-2026-0001",
          issued_at: "2026-04-01T10:00:00Z",
          source_type: "subscription",
          amount_incl_tax_cents: 2280,
          currency: "EUR",
          pdf_url: "",
        },
        {
          id: "inv-2",
          number: "FA-2026-0002",
          issued_at: "2026-04-15T10:00:00Z",
          source_type: "monthly_commission",
          amount_incl_tax_cents: 5000,
          currency: "EUR",
          pdf_url: "",
        },
      ],
    })
    render(withQueryClient(<InvoiceList />))
    await waitFor(() =>
      expect(screen.getByText(/FA-2026-0001/)).toBeInTheDocument(),
    )
    expect(screen.getByText(/Abonnement Premium/i)).toBeInTheDocument()
    expect(screen.getByText(/Commission mensuelle/i)).toBeInTheDocument()

    const downloadLink = screen.getAllByRole("link", { name: /Télécharger/i })[0]
    expect(downloadLink).toHaveAttribute("href", expect.stringContaining("/api/v1/me/invoices/inv-1/pdf"))
  })

  it("shows a `Voir plus` button when next_cursor is present", async () => {
    mockFetchInvoices.mockResolvedValue({
      data: [
        {
          id: "inv-1",
          number: "FA-2026-0001",
          issued_at: "2026-04-01T10:00:00Z",
          source_type: "subscription",
          amount_incl_tax_cents: 1000,
          currency: "EUR",
          pdf_url: "",
        },
      ],
      next_cursor: "cursor-page-2",
    })
    render(withQueryClient(<InvoiceList />))
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /Voir plus/i })).toBeInTheDocument(),
    )
    fireEvent.click(screen.getByRole("button", { name: /Voir plus/i }))
    // After clicking, the hook swaps the cursor and triggers a refetch.
    await waitFor(() => {
      expect(mockFetchInvoices).toHaveBeenLastCalledWith("cursor-page-2")
    })
  })
})
