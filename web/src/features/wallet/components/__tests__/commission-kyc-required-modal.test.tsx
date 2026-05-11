import { describe, expect, it, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"

import { CommissionKYCRequiredModal } from "../commission-kyc-required-modal"

describe("CommissionKYCRequiredModal", () => {
  it("renders nothing when open=false", () => {
    const { container } = render(
      <CommissionKYCRequiredModal open={false} onClose={() => {}} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it("renders the title and explainer when open=true", () => {
    render(<CommissionKYCRequiredModal open={true} onClose={() => {}} />)
    expect(
      screen.getByRole("heading", {
        name: /Termine ton KYC pour recevoir ta commission/i,
      }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/Ta commission est prête à être versée/i),
    ).toBeInTheDocument()
  })

  it("falls back to /payment-info when onboardingURL is missing", () => {
    render(<CommissionKYCRequiredModal open={true} onClose={() => {}} />)
    const cta = screen.getByRole("link", { name: /Compléter mes infos paiement/i })
    expect(cta).toHaveAttribute("href", "/payment-info")
    expect(cta).not.toHaveAttribute("target", "_blank")
  })

  it("deep-links to the Stripe onboarding URL when provided", () => {
    render(
      <CommissionKYCRequiredModal
        open={true}
        onClose={() => {}}
        onboardingURL="https://stripe.com/connect/onboarding/abc"
      />,
    )
    const cta = screen.getByRole("link", { name: /Terminer mon KYC/i })
    expect(cta).toHaveAttribute("href", "https://stripe.com/connect/onboarding/abc")
    expect(cta).toHaveAttribute("target", "_blank")
    expect(cta).toHaveAttribute("rel", "noopener noreferrer")
  })

  it("calls onClose when the X button is clicked", () => {
    const onClose = vi.fn()
    render(<CommissionKYCRequiredModal open={true} onClose={onClose} />)
    const closeButton = screen.getByRole("button", { name: /Fermer/i })
    fireEvent.click(closeButton)
    expect(onClose).toHaveBeenCalledOnce()
  })

  it("calls onClose when Plus tard is clicked", () => {
    const onClose = vi.fn()
    render(<CommissionKYCRequiredModal open={true} onClose={onClose} />)
    const laterButton = screen.getByRole("button", { name: /Plus tard/i })
    fireEvent.click(laterButton)
    expect(onClose).toHaveBeenCalledOnce()
  })

  it("calls onClose on Escape keypress", () => {
    const onClose = vi.fn()
    render(<CommissionKYCRequiredModal open={true} onClose={onClose} />)
    fireEvent.keyDown(window, { key: "Escape" })
    expect(onClose).toHaveBeenCalledOnce()
  })

  it("does NOT call onClose on a non-Escape key", () => {
    const onClose = vi.fn()
    render(<CommissionKYCRequiredModal open={true} onClose={onClose} />)
    fireEvent.keyDown(window, { key: "Enter" })
    expect(onClose).not.toHaveBeenCalled()
  })
})
