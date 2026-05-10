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
  it("renders the three allowed period buttons with the active one pressed", () => {
    render(wrap(<PeriodSelector value={30} onChange={() => {}} />))
    const buttons = screen.getAllByRole("button")
    expect(buttons).toHaveLength(3)
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
})
