import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { PeriodSelector } from "../components/period-selector"

function wrap(node: React.ReactNode) {
  return (
    <NextIntlClientProvider locale="fr" messages={messages}>
      {node}
    </NextIntlClientProvider>
  )
}

describe("PeriodSelector", () => {
  it("renders the four allowed period buttons with the active one pressed", () => {
    // D3: 7 / 30 / 90 / 365 — long-tail option added so the user can
    // scan a 1-year window. Every option is keyboard accessible.
    render(wrap(<PeriodSelector value={30} onChange={() => {}} />))
    const buttons = screen.getAllByRole("button")
    expect(buttons).toHaveLength(4)
    const pressed = buttons.find((b) => b.getAttribute("aria-pressed") === "true")
    expect(pressed?.textContent).toContain("30")
  })

  it("invokes onChange with the selected days", async () => {
    const onChange = vi.fn()
    render(wrap(<PeriodSelector value={30} onChange={onChange} />))
    const buttons = screen.getAllByRole("button")
    await userEvent.click(buttons[2]) // 90d
    expect(onChange).toHaveBeenCalledWith(90)
  })

  it("renders the 1-year chip and forwards 365 onChange", async () => {
    const onChange = vi.fn()
    render(wrap(<PeriodSelector value={30} onChange={onChange} />))
    const oneYear = screen
      .getAllByRole("button")
      .find((b) => b.textContent?.includes("1 an"))
    expect(oneYear).toBeTruthy()
    await userEvent.click(oneYear as HTMLElement)
    expect(onChange).toHaveBeenCalledWith(365)
  })

  it("marks the 1-year chip as pressed when value=365", () => {
    render(wrap(<PeriodSelector value={365} onChange={() => {}} />))
    const pressed = screen
      .getAllByRole("button")
      .find((b) => b.getAttribute("aria-pressed") === "true")
    expect(pressed?.textContent).toContain("1 an")
  })
})
