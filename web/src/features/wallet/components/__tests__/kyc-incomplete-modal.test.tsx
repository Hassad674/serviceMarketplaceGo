import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { KYCIncompleteModal } from "../kyc-incomplete-modal"

const mockPush = vi.fn()
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}))

describe("KYCIncompleteModal", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the default Stripe onboarding copy", () => {
    render(<KYCIncompleteModal open onClose={() => {}} />)
    expect(
      screen.getByText(/Termine ton onboarding Stripe pour pouvoir retirer/i),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/finalise ton onboarding Stripe/i),
    ).toBeInTheDocument()
  })

  it("uses the server-provided message when supplied", () => {
    const serverMsg =
      "Termine d'abord ton onboarding Stripe sur la page Infos paiement avant de pouvoir retirer."
    render(<KYCIncompleteModal open onClose={() => {}} message={serverMsg} />)
    expect(screen.getByText(serverMsg)).toBeInTheDocument()
    // The default copy must NOT also be rendered
    expect(
      screen.queryByText(/finalise ton onboarding Stripe/i),
    ).not.toBeInTheDocument()
  })

  it("routes to /payment-info on CTA click and closes the modal", () => {
    const onClose = vi.fn()
    render(<KYCIncompleteModal open onClose={onClose} />)
    fireEvent.click(
      screen.getByRole("button", { name: /Aller à Infos paiement/i }),
    )
    expect(mockPush).toHaveBeenCalledWith("/payment-info")
    expect(onClose).toHaveBeenCalled()
  })

  it("respects a destination override", () => {
    render(
      <KYCIncompleteModal
        open
        onClose={() => {}}
        destination="/fr/payment-info"
      />,
    )
    fireEvent.click(
      screen.getByRole("button", { name: /Aller à Infos paiement/i }),
    )
    expect(mockPush).toHaveBeenCalledWith("/fr/payment-info")
  })

  it("does not render when closed", () => {
    render(<KYCIncompleteModal open={false} onClose={() => {}} />)
    expect(
      screen.queryByText(/Termine ton onboarding Stripe/i),
    ).not.toBeInTheDocument()
  })

  it("dismisses via the secondary 'Plus tard' button without redirecting", () => {
    const onClose = vi.fn()
    render(<KYCIncompleteModal open onClose={onClose} />)
    fireEvent.click(screen.getByRole("button", { name: /Plus tard/i }))
    expect(onClose).toHaveBeenCalled()
    expect(mockPush).not.toHaveBeenCalled()
  })
})
