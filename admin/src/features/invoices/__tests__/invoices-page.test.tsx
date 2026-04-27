import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, waitFor, fireEvent } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter } from "react-router-dom"
import { InvoicesPage } from "../components/invoices-page"
import * as api from "../api/invoicing-api"
import type { AdminInvoiceListResponse } from "../types"

function renderPage() {
  const qc = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <InvoicesPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const sampleResponse: AdminInvoiceListResponse = {
  data: [
    {
      id: "11111111-1111-1111-1111-111111111111",
      number: "FAC-000123",
      is_credit_note: false,
      recipient_org_id: "22222222-2222-2222-2222-222222222222",
      recipient_legal_name: "Acme Studio SARL",
      issued_at: "2026-04-15T10:00:00Z",
      amount_incl_tax_cents: 4900,
      currency: "EUR",
      tax_regime: "fr_franchise_base",
      status: "issued",
      source_type: "subscription",
    },
    {
      id: "33333333-3333-3333-3333-333333333333",
      number: "AV-000007",
      is_credit_note: true,
      recipient_org_id: "22222222-2222-2222-2222-222222222222",
      recipient_legal_name: "Acme Studio SARL",
      issued_at: "2026-04-20T10:00:00Z",
      amount_incl_tax_cents: 4900,
      currency: "EUR",
      tax_regime: "fr_franchise_base",
      status: "credit_note",
    },
  ],
  has_more: false,
}

describe("InvoicesPage", () => {
  let fetchSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    fetchSpy = vi.spyOn(api, "fetchAdminInvoices")
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("renders rows from the API", async () => {
    fetchSpy.mockResolvedValueOnce(sampleResponse)
    renderPage()

    await waitFor(() => {
      expect(screen.getByText("FAC-000123")).toBeInTheDocument()
    })
    expect(screen.getByText("AV-000007")).toBeInTheDocument()
    // The recipient legal name renders for both rows.
    expect(screen.getAllByText("Acme Studio SARL").length).toBe(2)
    // Page header
    expect(screen.getByText(/Toutes les factures emises/i)).toBeInTheDocument()
  })

  it("renders empty state when API returns no rows", async () => {
    fetchSpy.mockResolvedValueOnce({ data: [], has_more: false })
    renderPage()

    await waitFor(() => {
      // Use the precise empty-state title (a sibling description also
      // includes "Aucune facture" so the regex matches twice).
      expect(screen.getByText("Aucune facture")).toBeInTheDocument()
    })
  })

  it("changing the type filter triggers a refetch with the new status", async () => {
    fetchSpy.mockResolvedValueOnce(sampleResponse)
    renderPage()

    await waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledTimes(1)
    })

    fetchSpy.mockResolvedValueOnce({ data: [], has_more: false })

    const typeSelect = screen.getByRole("combobox") as HTMLSelectElement
    fireEvent.change(typeSelect, { target: { value: "credit_note" } })

    await waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledTimes(2)
    })
    const lastCallFilters = fetchSpy.mock.calls[1][0]
    expect(lastCallFilters.status).toBe("credit_note")
    // Cursor must be reset on filter change.
    expect(lastCallFilters.cursor).toBe("")
  })

  it("clicking a row calls openInvoicePDF with the right type flag", async () => {
    fetchSpy.mockResolvedValueOnce(sampleResponse)
    const openSpy = vi
      .spyOn(api, "openInvoicePDF")
      .mockResolvedValue("https://r2.test/x.pdf")
    // window.open is not implemented in jsdom — stub it.
    const openWindowSpy = vi
      .spyOn(window, "open")
      .mockReturnValue(null as unknown as Window)
    renderPage()

    await waitFor(() => {
      expect(screen.getByText("FAC-000123")).toBeInTheDocument()
    })

    fireEvent.click(screen.getByTestId("invoice-row-11111111-1111-1111-1111-111111111111"))

    await waitFor(() => {
      expect(openSpy).toHaveBeenCalledWith(
        "11111111-1111-1111-1111-111111111111",
        false,
      )
    })

    // Click the credit-note row → isCreditNote=true.
    fireEvent.click(screen.getByTestId("invoice-row-33333333-3333-3333-3333-333333333333"))
    await waitFor(() => {
      expect(openSpy).toHaveBeenCalledWith(
        "33333333-3333-3333-3333-333333333333",
        true,
      )
    })

    expect(openWindowSpy).toHaveBeenCalled()
  })
})
