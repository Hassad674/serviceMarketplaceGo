import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { PaymentTypeSelector } from "../payment-type-selector"
import type { PaymentType } from "../../types"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  Check: (props: Record<string, unknown>) => <span data-testid="check-icon" {...props} />,
  FileText: (props: Record<string, unknown>) => <span data-testid="filetext-icon" {...props} />,
  ShieldCheck: (props: Record<string, unknown>) => <span data-testid="shieldcheck-icon" {...props} />,
}))

describe("PaymentTypeSelector", () => {
  it("renders both payment cards", () => {
    const onChange = vi.fn()
    render(<PaymentTypeSelector value="escrow" onChange={onChange} />)

    expect(screen.getByText("invoiceBilling")).toBeDefined()
    expect(screen.getByText("escrowPayments")).toBeDefined()
  })

  it("escrow selected by default when value is escrow", () => {
    const onChange = vi.fn()
    const { container } = render(
      <PaymentTypeSelector value="escrow" onChange={onChange} />,
    )

    // The escrow card should have the selected border class
    const buttons = container.querySelectorAll("button[type='button']")
    // Invoice card (first) should NOT have the selected class
    expect(buttons[0].className).not.toContain("border-rose-500")
    // Escrow card (second) should have the selected class
    expect(buttons[1].className).toContain("border-rose-500")
  })

  it("click switches selection to invoice", () => {
    const onChange = vi.fn()
    render(<PaymentTypeSelector value="escrow" onChange={onChange} />)

    fireEvent.click(screen.getByText("invoiceBilling"))

    expect(onChange).toHaveBeenCalledWith("invoice")
  })

  it("click switches selection to escrow", () => {
    const onChange = vi.fn()
    render(<PaymentTypeSelector value="invoice" onChange={onChange} />)

    fireEvent.click(screen.getByText("escrowPayments"))

    expect(onChange).toHaveBeenCalledWith("escrow")
  })

  it("shows check icon on selected card only", () => {
    const onChange = vi.fn()
    const { container } = render(
      <PaymentTypeSelector value="escrow" onChange={onChange} />,
    )

    // The check icon should appear only once (on the selected card)
    const checkIcons = container.querySelectorAll("[data-testid='check-icon']")
    expect(checkIcons.length).toBe(1)
  })

  it("renders section title", () => {
    const onChange = vi.fn()
    render(<PaymentTypeSelector value="escrow" onChange={onChange} />)

    expect(screen.getByText("paymentType")).toBeDefined()
  })

  it("renders descriptions for both cards", () => {
    const onChange = vi.fn()
    render(<PaymentTypeSelector value="escrow" onChange={onChange} />)

    expect(screen.getByText("invoiceBillingDesc")).toBeDefined()
    expect(screen.getByText("escrowPaymentsDesc")).toBeDefined()
  })

  it("invoice card is highlighted when selected", () => {
    const onChange = vi.fn()
    const { container } = render(
      <PaymentTypeSelector value={"invoice" as PaymentType} onChange={onChange} />,
    )

    const buttons = container.querySelectorAll("button[type='button']")
    // Invoice card (first) should be selected
    expect(buttons[0].className).toContain("border-rose-500")
    // Escrow card (second) should NOT be selected
    expect(buttons[1].className).not.toContain("border-rose-500")
  })
})
