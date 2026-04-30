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
    // Submit via the form's submit event so RHF runs validation +
    // delegates to handleSubmit (button click via fireEvent does not
    // bubble to a real submit in JSDOM-stripped <form>).
    const saveBtn = screen.getByRole("button", { name: /Enregistrer/i })
    fireEvent.click(saveBtn)
    await waitFor(() => {
      expect(mockUpdate).toHaveBeenCalledTimes(1)
    }, { timeout: 5000 })
    const payload = mockUpdate.mock.calls[0][0] as { legal_name: string }
    expect(payload.legal_name).toBe("Acme New")
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

  it("triggers the sync mutation when 'Pré-remplir depuis Stripe' is clicked", async () => {
    mockFetch.mockResolvedValue(SNAPSHOT)
    mockSync.mockResolvedValue(SNAPSHOT)
    render(withQueryClient(<BillingProfileForm />))
    await waitFor(() =>
      expect(
        (screen.getByLabelText(/Raison sociale ou nom légal/i) as HTMLInputElement).value,
      ).toBe("Acme SAS"),
    )
    fireEvent.click(screen.getByRole("button", { name: /Pré-remplir depuis Stripe/i }))
    await waitFor(() => expect(mockSync).toHaveBeenCalledTimes(1))
  })

  it("shows the sync error banner when sync fails", async () => {
    mockFetch.mockResolvedValue(SNAPSHOT)
    mockSync.mockRejectedValue(new Error("boom"))
    render(withQueryClient(<BillingProfileForm />))
    await waitFor(() =>
      expect(
        (screen.getByLabelText(/Raison sociale ou nom légal/i) as HTMLInputElement).value,
      ).toBe("Acme SAS"),
    )
    fireEvent.click(screen.getByRole("button", { name: /Pré-remplir depuis Stripe/i }))
    await waitFor(() =>
      expect(
        screen.getByText(/La synchronisation Stripe a échoué/),
      ).toBeInTheDocument(),
    )
  })

  it("shows 'Profil enregistré' after a successful save", async () => {
    mockFetch.mockResolvedValue(SNAPSHOT)
    mockUpdate.mockResolvedValue({ ...SNAPSHOT, is_complete: true })
    render(withQueryClient(<BillingProfileForm />))
    await waitFor(() =>
      expect(
        (screen.getByLabelText(/Raison sociale ou nom légal/i) as HTMLInputElement).value,
      ).toBe("Acme SAS"),
    )
    fireEvent.click(screen.getByRole("button", { name: /Enregistrer/i }))
    await waitFor(() =>
      expect(screen.getByText(/Profil enregistré/)).toBeInTheDocument(),
    )
  })

  it("shows the update error message when save fails", async () => {
    mockFetch.mockResolvedValue(SNAPSHOT)
    mockUpdate.mockRejectedValue(new Error("server-error"))
    render(withQueryClient(<BillingProfileForm />))
    await waitFor(() =>
      expect(
        (screen.getByLabelText(/Raison sociale ou nom légal/i) as HTMLInputElement).value,
      ).toBe("Acme SAS"),
    )
    fireEvent.click(screen.getByRole("button", { name: /Enregistrer/i }))
    await waitFor(() =>
      expect(
        screen.getByText(/L'enregistrement a échoué/),
      ).toBeInTheDocument(),
    )
  })

  it("triggers onSaved when the save is successful AND profile is complete", async () => {
    mockFetch.mockResolvedValue({ ...SNAPSHOT, is_complete: true })
    mockUpdate.mockResolvedValue({ ...SNAPSHOT, is_complete: true })
    const onSaved = vi.fn()
    render(withQueryClient(<BillingProfileForm onSaved={onSaved} />))
    await waitFor(() =>
      expect(
        (screen.getByLabelText(/Raison sociale ou nom légal/i) as HTMLInputElement).value,
      ).toBe("Acme SAS"),
    )
    fireEvent.click(screen.getByRole("button", { name: /Enregistrer/i }))
    await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1))
  })

  it("does NOT trigger onSaved when the saved profile is still incomplete", async () => {
    mockFetch.mockResolvedValue({
      ...SNAPSHOT,
      is_complete: false,
      missing_fields: [{ field: "tax_id", reason: "required" }],
    })
    mockUpdate.mockResolvedValue({ ...SNAPSHOT, is_complete: false })
    const onSaved = vi.fn()
    render(withQueryClient(<BillingProfileForm onSaved={onSaved} />))
    await waitFor(() =>
      expect(
        (screen.getByLabelText(/Raison sociale ou nom légal/i) as HTMLInputElement).value,
      ).toBe("Acme SAS"),
    )
    fireEvent.click(screen.getByRole("button", { name: /Enregistrer/i }))
    // Wait briefly to give the effect a chance to fire (it will not).
    await new Promise((resolve) => setTimeout(resolve, 100))
    expect(onSaved).not.toHaveBeenCalled()
  })

  it("renders the FormSkeleton when the query is loading", async () => {
    mockFetch.mockReturnValue(new Promise(() => {})) // never resolves
    const { container } = render(withQueryClient(<BillingProfileForm />))
    expect(container.innerHTML).toContain("animate-shimmer")
  })

  it("renders the error fallback when the query errors out", async () => {
    // The hook configures retry: 1 — let it run twice before the
    // error state lands.
    mockFetch.mockRejectedValue(new Error("nope"))
    render(withQueryClient(<BillingProfileForm />))
    await waitFor(
      () =>
        expect(
          screen.getByText(/Impossible de charger le profil de facturation/),
        ).toBeInTheDocument(),
      { timeout: 5_000 },
    )
  })

  it("compact variant hides the synced indicator", async () => {
    mockFetch.mockResolvedValue(SNAPSHOT)
    render(withQueryClient(<BillingProfileForm variant="compact" />))
    await waitFor(() =>
      expect(
        (screen.getByLabelText(/Raison sociale ou nom légal/i) as HTMLInputElement).value,
      ).toBe("Acme SAS"),
    )
    expect(screen.queryByText(/Synchronisé depuis Stripe/)).not.toBeInTheDocument()
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
