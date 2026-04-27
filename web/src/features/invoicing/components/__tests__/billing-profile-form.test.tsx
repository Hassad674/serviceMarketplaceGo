import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { BillingProfileForm } from "../billing-profile-form"

const mockFetch = vi.fn()
const mockUpdate = vi.fn()
const mockSync = vi.fn()
const mockValidate = vi.fn()

vi.mock("../../api/invoicing-api", () => ({
  fetchBillingProfile: () => mockFetch(),
  updateBillingProfile: (...args: unknown[]) => mockUpdate(...args),
  syncBillingProfileFromStripe: () => mockSync(),
  validateBillingProfileVAT: () => mockValidate(),
}))

function withQueryClient(ui: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return createElement(QueryClientProvider, { client }, ui)
}

const SNAPSHOT = {
  profile: {
    organization_id: "org-1",
    profile_type: "business" as const,
    legal_name: "Acme SAS",
    trading_name: "Acme",
    legal_form: "SAS",
    tax_id: "12345678901234",
    vat_number: "FR12345678901",
    vat_validated_at: null,
    address_line1: "1 rue de la Paix",
    address_line2: "",
    postal_code: "75001",
    city: "Paris",
    country: "FR",
    invoicing_email: "billing@acme.com",
    synced_from_kyc_at: "2026-04-01T10:00:00Z",
  },
  missing_fields: [],
  is_complete: true,
}

describe("BillingProfileForm", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the hydrated profile fields", async () => {
    mockFetch.mockResolvedValue(SNAPSHOT)
    render(withQueryClient(<BillingProfileForm />))
    await waitFor(() => {
      expect(
        (screen.getByLabelText(/Raison sociale ou nom légal/i) as HTMLInputElement).value,
      ).toBe("Acme SAS")
    })
    expect(
      (screen.getByLabelText(/Numéro SIRET/i) as HTMLInputElement).value,
    ).toBe("12345678901234")
    expect(screen.getByText(/Synchronisé depuis Stripe/i)).toBeInTheDocument()
  })

  it("submits the update mutation with the patched payload", async () => {
    mockFetch.mockResolvedValue(SNAPSHOT)
    mockUpdate.mockResolvedValue({ ...SNAPSHOT, is_complete: true })
    render(withQueryClient(<BillingProfileForm />))
    await waitFor(() =>
      expect(
        (screen.getByLabelText(/Raison sociale ou nom légal/i) as HTMLInputElement).value,
      ).toBe("Acme SAS"),
    )
    fireEvent.change(screen.getByLabelText(/Raison sociale ou nom légal/i), {
      target: { value: "Acme New" },
    })
    fireEvent.click(screen.getByRole("button", { name: /Enregistrer/i }))
    await waitFor(() => {
      expect(mockUpdate).toHaveBeenCalledTimes(1)
      const payload = mockUpdate.mock.calls[0][0] as { legal_name: string }
      expect(payload.legal_name).toBe("Acme New")
    })
  })

  it("renders the missing-fields banner when the snapshot is incomplete", async () => {
    mockFetch.mockResolvedValue({
      ...SNAPSHOT,
      missing_fields: [{ field: "legal_name", reason: "required" }],
      is_complete: false,
    })
    render(withQueryClient(<BillingProfileForm />))
    await waitFor(() =>
      expect(
        screen.getByText(/Quelques informations restent à compléter/i),
      ).toBeInTheDocument(),
    )
    // The "Identité légale" section heading is still rendered — the
    // banner echoes the field via describeMissing in the list item.
    expect(
      screen.getByRole("heading", { name: /Identité légale/i }),
    ).toBeInTheDocument()
  })

  it("disables the VAT validation button until a number is filled", async () => {
    mockFetch.mockResolvedValue({
      ...SNAPSHOT,
      profile: { ...SNAPSHOT.profile, vat_number: "" },
    })
    render(withQueryClient(<BillingProfileForm />))
    await waitFor(() => {
      const btn = screen.getByRole("button", { name: /Valider mon n° TVA/i })
      expect(btn).toBeDisabled()
    })
  })
})
