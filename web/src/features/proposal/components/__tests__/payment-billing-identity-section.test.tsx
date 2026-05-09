/**
 * Unit tests for PaymentBillingIdentitySection — the inline B2B legal-
 * identity capture rendered above the Stripe Payment Element on the
 * proposal payment page.
 *
 * Behaviour pinned by these tests:
 *   - Pre-fills the inputs from the existing billing profile (so the
 *     second payment never re-asks for already-saved data).
 *   - SIRET input strips non-digits and caps at 14 characters.
 *   - SIRET error message renders for FR profiles when partial.
 *   - VAT error message renders for any profile when format is wrong.
 *   - onChange callback fires with the current values on every edit
 *     so the parent submit handler can capture the latest snapshot
 *     without coupling to the form's internal state.
 *   - persistInlineBillingIdentity short-circuits when every field is
 *     empty (no PUT call, no network round-trip).
 */

import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import {
  PaymentBillingIdentitySection,
  persistInlineBillingIdentity,
  type PaymentBillingIdentityValues,
} from "../payment-billing-identity-section"
import type { BillingProfile, BillingProfileSnapshot } from "@/shared/types/billing-profile"

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

const useBillingProfileMock = vi.fn()
vi.mock("@/shared/hooks/billing-profile/use-billing-profile", () => ({
  useBillingProfile: () => useBillingProfileMock(),
}))

const updateBillingProfileMock = vi.fn()
vi.mock("@/shared/lib/billing-profile/billing-profile-api", () => ({
  updateBillingProfile: (...args: unknown[]) => updateBillingProfileMock(...args),
}))

// Lucide icons → inert spans
vi.mock("lucide-react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("lucide-react")>()
  return new Proxy(actual, {
    get(target, prop) {
      if (typeof prop === "string" && /^[A-Z]/.test(prop)) {
        const Icon = () => <span data-testid={`icon-${prop}`} />
        Icon.displayName = `Icon(${prop})`
        return Icon
      }
      return Reflect.get(target, prop)
    },
  })
})

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeSnapshot(overrides: Partial<BillingProfile> = {}): BillingProfileSnapshot {
  return {
    profile: {
      organization_id: "org-1",
      profile_type: "business",
      legal_name: "",
      trading_name: "",
      legal_form: "",
      tax_id: "",
      vat_number: "",
      vat_validated_at: null,
      address_line1: "",
      address_line2: "",
      postal_code: "",
      city: "",
      country: "FR",
      invoicing_email: "",
      synced_from_kyc_at: null,
      ...overrides,
    },
    missing_fields: [],
    is_complete: false,
  }
}

beforeEach(() => {
  useBillingProfileMock.mockReset()
  updateBillingProfileMock.mockReset()
})

// ---------------------------------------------------------------------------
// Tests — UI behaviour
// ---------------------------------------------------------------------------

describe("PaymentBillingIdentitySection", () => {
  it("renders a loading indicator while the profile is being fetched", () => {
    useBillingProfileMock.mockReturnValue({ data: undefined, isLoading: true })
    render(<PaymentBillingIdentitySection />)
    // The loader is the only content rendered while loading.
    expect(screen.getByTestId("icon-Loader2")).toBeInTheDocument()
    // None of the form labels should appear yet.
    expect(screen.queryByText("legalNameLabel")).not.toBeInTheDocument()
  })

  it("pre-fills inputs from the existing billing profile snapshot", () => {
    useBillingProfileMock.mockReturnValue({
      data: makeSnapshot({
        legal_name: "Acme SARL",
        tax_id: "12345678901234",
        vat_number: "FR12345678901",
      }),
      isLoading: false,
    })
    render(<PaymentBillingIdentitySection />)
    expect(screen.getByDisplayValue("Acme SARL")).toBeInTheDocument()
    expect(screen.getByDisplayValue("12345678901234")).toBeInTheDocument()
    expect(screen.getByDisplayValue("FR12345678901")).toBeInTheDocument()
  })

  it("shows the SIRET field for French organizations and strips non-digits", () => {
    useBillingProfileMock.mockReturnValue({ data: makeSnapshot({ country: "FR" }), isLoading: false })
    render(<PaymentBillingIdentitySection />)
    const input = screen.getByLabelText("siretLabel") as HTMLInputElement
    fireEvent.change(input, { target: { value: "123-456 789 01234extra" } })
    // Strips non-digits, caps at 14 characters.
    expect(input.value).toBe("12345678901234")
  })

  it("shows a SIRET format error when the user types a partial value", () => {
    useBillingProfileMock.mockReturnValue({ data: makeSnapshot({ country: "FR" }), isLoading: false })
    render(<PaymentBillingIdentitySection />)
    const input = screen.getByLabelText("siretLabel") as HTMLInputElement
    fireEvent.change(input, { target: { value: "123" } })
    expect(screen.getByText("siretError")).toBeInTheDocument()
  })

  it("shows a VAT format error when the user types an invalid intra-EU code", () => {
    useBillingProfileMock.mockReturnValue({ data: makeSnapshot(), isLoading: false })
    render(<PaymentBillingIdentitySection />)
    const input = screen.getByLabelText(/vatLabel/) as HTMLInputElement
    fireEvent.change(input, { target: { value: "12X" } })
    expect(screen.getByText("vatError")).toBeInTheDocument()
  })

  it("calls onChange with the current values on every edit", () => {
    useBillingProfileMock.mockReturnValue({ data: makeSnapshot(), isLoading: false })
    const onChange = vi.fn()
    render(<PaymentBillingIdentitySection onChange={onChange} />)

    const legalName = screen.getByLabelText("legalNameLabel") as HTMLInputElement
    fireEvent.change(legalName, { target: { value: "Acme" } })

    // The latest call carries the new legal_name value.
    const calls = onChange.mock.calls
    expect(calls.length).toBeGreaterThan(0)
    const lastCall = calls[calls.length - 1][0] as PaymentBillingIdentityValues
    expect(lastCall.legal_name).toBe("Acme")
  })

  it("hides the SIRET field for non-FR profiles", () => {
    useBillingProfileMock.mockReturnValue({ data: makeSnapshot({ country: "DE" }), isLoading: false })
    render(<PaymentBillingIdentitySection />)
    expect(screen.queryByLabelText("siretLabel")).not.toBeInTheDocument()
    // VAT field still shows.
    expect(screen.getByLabelText(/vatLabel/)).toBeInTheDocument()
  })
})

// ---------------------------------------------------------------------------
// Tests — persistInlineBillingIdentity helper
// ---------------------------------------------------------------------------

describe("persistInlineBillingIdentity", () => {
  it("short-circuits when every field is empty (no PUT call)", async () => {
    await persistInlineBillingIdentity(
      { legal_name: "", tax_id: "", vat_number: "" },
      undefined,
    )
    expect(updateBillingProfileMock).not.toHaveBeenCalled()
  })

  it("PUTs the merged payload when at least one field is filled", async () => {
    updateBillingProfileMock.mockResolvedValue({})
    await persistInlineBillingIdentity(
      { legal_name: "Acme", tax_id: "", vat_number: "" },
      undefined,
    )
    expect(updateBillingProfileMock).toHaveBeenCalledTimes(1)
    const payload = updateBillingProfileMock.mock.calls[0][0]
    expect(payload.legal_name).toBe("Acme")
    expect(payload.profile_type).toBe("business")
  })

  it("preserves existing fields not touched by the inline form", async () => {
    updateBillingProfileMock.mockResolvedValue({})
    await persistInlineBillingIdentity(
      { legal_name: "", tax_id: "12345678901234", vat_number: "" },
      makeSnapshot({
        legal_name: "Existing Name",
        address_line1: "1 rue de la Paix",
        city: "Paris",
        country: "FR",
      }).profile,
    )
    const payload = updateBillingProfileMock.mock.calls[0][0]
    expect(payload.tax_id).toBe("12345678901234")
    expect(payload.legal_name).toBe("Existing Name")
    expect(payload.address_line1).toBe("1 rue de la Paix")
    expect(payload.city).toBe("Paris")
  })
})
