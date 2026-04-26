import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { BillingProfileCompletionModal } from "../billing-profile-completion-modal"

const mockPush = vi.fn()
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}))

describe("BillingProfileCompletionModal", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the missing fields with French labels", () => {
    render(
      <BillingProfileCompletionModal
        open
        onClose={() => {}}
        missingFields={[
          { field: "legal_name", reason: "required" },
          { field: "vat_number", reason: "not_validated" },
        ]}
      />,
    )
    expect(
      screen.getByText(/Raison sociale ou nom légal/i),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/Numéro de TVA intracommunautaire/i),
    ).toBeInTheDocument()
  })

  it("falls back to a generic message when no fields are provided", () => {
    render(
      <BillingProfileCompletionModal
        open
        onClose={() => {}}
        missingFields={[]}
      />,
    )
    expect(
      screen.getByText(/Quelques informations restent à compléter/i),
    ).toBeInTheDocument()
  })

  it("routes to the billing profile page on CTA click", () => {
    const onClose = vi.fn()
    render(
      <BillingProfileCompletionModal
        open
        onClose={onClose}
        missingFields={[{ field: "legal_name", reason: "required" }]}
      />,
    )
    fireEvent.click(screen.getByRole("button", { name: /Compléter mon profil/i }))
    expect(mockPush).toHaveBeenCalledWith("/settings/billing-profile")
    expect(onClose).toHaveBeenCalled()
  })

  it("respects the destination override", () => {
    render(
      <BillingProfileCompletionModal
        open
        onClose={() => {}}
        missingFields={[]}
        destination="/fr/settings/billing-profile"
      />,
    )
    fireEvent.click(screen.getByRole("button", { name: /Compléter mon profil/i }))
    expect(mockPush).toHaveBeenCalledWith("/fr/settings/billing-profile")
  })

  it("does not render when closed", () => {
    render(
      <BillingProfileCompletionModal
        open={false}
        onClose={() => {}}
        missingFields={[{ field: "legal_name", reason: "required" }]}
      />,
    )
    expect(screen.queryByText(/Compléter mon profil/i)).not.toBeInTheDocument()
  })
})
