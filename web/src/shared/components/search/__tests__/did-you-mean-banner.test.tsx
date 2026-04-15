import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { DidYouMeanBanner } from "../did-you-mean-banner"

function renderWithIntl(ui: React.ReactNode) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      {ui}
    </NextIntlClientProvider>,
  )
}

describe("DidYouMeanBanner", () => {
  it("renders the corrected query string", () => {
    renderWithIntl(<DidYouMeanBanner correctedQuery="react" onApply={() => {}} />)
    expect(screen.getByText(/Did you mean/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /run the corrected/i })).toHaveTextContent(
      "react",
    )
  })

  it("returns null for empty corrected query", () => {
    const { container } = renderWithIntl(
      <DidYouMeanBanner correctedQuery="" onApply={() => {}} />,
    )
    expect(container.firstChild).toBeNull()
  })

  it("returns null for whitespace-only corrected query", () => {
    const { container } = renderWithIntl(
      <DidYouMeanBanner correctedQuery="   " onApply={() => {}} />,
    )
    expect(container.firstChild).toBeNull()
  })

  it("calls onApply with the trimmed corrected query when clicked", async () => {
    const onApply = vi.fn()
    renderWithIntl(
      <DidYouMeanBanner correctedQuery="  react  " onApply={onApply} />,
    )
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: /run the corrected/i }))
    expect(onApply).toHaveBeenCalledWith("react")
  })

  it("uses an aria-live region for accessibility", () => {
    renderWithIntl(<DidYouMeanBanner correctedQuery="react" onApply={() => {}} />)
    const status = screen.getByRole("status")
    expect(status).toHaveAttribute("aria-live", "polite")
  })
})
