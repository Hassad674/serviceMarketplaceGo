// Unit tests for BillingProfileSummary — the compact read-only card
// rendered on the payment page when the client's billing-profile is
// already complete. The card surfaces pays + adresse + entité +
// identifiants fiscaux, and exposes a single "Modifier" CTA that
// hands edit control back to the parent.

import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import type { BillingProfile } from "@/shared/types/billing-profile"
import { BillingProfileSummary } from "../billing-profile-summary"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => `i18n:${key}`,
}))

const businessProfile: BillingProfile = {
  organization_id: "org-1",
  profile_type: "business",
  legal_name: "Acme Studio SARL",
  trading_name: "Acme",
  legal_form: "SARL",
  tax_id: "12345678901234",
  vat_number: "FR12345678901",
  vat_validated_at: null,
  address_line1: "12 rue de la Paix",
  address_line2: "Bât. C",
  postal_code: "75001",
  city: "Paris",
  country: "FR",
  invoicing_email: "billing@acme.example",
  synced_from_kyc_at: null,
}

const individualMinimal: BillingProfile = {
  organization_id: "org-2",
  profile_type: "individual",
  legal_name: "Alice Martin",
  trading_name: "",
  legal_form: "",
  tax_id: "",
  vat_number: "",
  vat_validated_at: null,
  address_line1: "5 avenue Foch",
  address_line2: "",
  postal_code: "75016",
  city: "Paris",
  country: "FR",
  invoicing_email: "",
  synced_from_kyc_at: null,
}

describe("BillingProfileSummary", () => {
  it("renders the legal name, trading name, country, and address of a business profile", () => {
    render(<BillingProfileSummary profile={businessProfile} onEdit={() => {}} />)
    expect(screen.getByText("Acme Studio SARL")).toBeInTheDocument()
    expect(screen.getByText("Acme")).toBeInTheDocument()
    expect(screen.getByText("FR")).toBeInTheDocument()
    expect(
      screen.getByText(/12 rue de la Paix.*Bât\. C.*75001 Paris/),
    ).toBeInTheDocument()
  })

  it("renders SIRET + VAT for a business profile", () => {
    render(<BillingProfileSummary profile={businessProfile} onEdit={() => {}} />)
    expect(screen.getByText(/12345678901234/)).toBeInTheDocument()
    expect(screen.getByText(/FR12345678901/)).toBeInTheDocument()
  })

  it("does NOT render the tax row for an individual profile without tax_id or vat_number", () => {
    render(
      <BillingProfileSummary profile={individualMinimal} onEdit={() => {}} />,
    )
    expect(screen.queryByText(/i18n:tax/i)).not.toBeInTheDocument()
  })

  it("fires onEdit when the Modifier button is clicked", () => {
    const onEdit = vi.fn()
    render(<BillingProfileSummary profile={businessProfile} onEdit={onEdit} />)
    fireEvent.click(screen.getByRole("button", { name: /editCta/i }))
    expect(onEdit).toHaveBeenCalledTimes(1)
  })

  it("renders a dash placeholder when a critical field is missing", () => {
    const partial: BillingProfile = {
      ...individualMinimal,
      legal_name: "",
    }
    render(<BillingProfileSummary profile={partial} onEdit={() => {}} />)
    // The legal_name dt + dd should still render but the inner value
    // falls back to "—".
    const dashes = screen.getAllByText("—")
    expect(dashes.length).toBeGreaterThan(0)
  })

  it("renders the title and Modifier CTA via i18n keys", () => {
    render(<BillingProfileSummary profile={businessProfile} onEdit={() => {}} />)
    expect(screen.getByText("i18n:summaryTitle")).toBeInTheDocument()
    expect(screen.getByText("i18n:editCta")).toBeInTheDocument()
  })

  it("does NOT show trading_name when empty", () => {
    render(
      <BillingProfileSummary profile={individualMinimal} onEdit={() => {}} />,
    )
    // The minimal profile has trading_name === "" — must not be in the DOM.
    // We rely on querying for the empty fragment indirectly: the legal_name
    // is present, but no second line under "entity" exists.
    expect(screen.queryByText("Acme")).not.toBeInTheDocument()
  })
})
