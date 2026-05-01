import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ProfileAboutCard } from "../profile-about-card"

function renderCard(props: Parameters<typeof ProfileAboutCard>[0]) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ProfileAboutCard {...props} />
    </NextIntlClientProvider>,
  )
}

describe("ProfileAboutCard", () => {
  it("renders the heading and content when provided", () => {
    renderCard({
      content: "Hello world",
      label: "About",
      placeholder: "Tell us more",
      onSave: vi.fn(),
    })

    expect(screen.getByRole("heading", { name: "About" })).toBeInTheDocument()
    expect(screen.getByText("Hello world")).toBeInTheDocument()
  })

  // Regression: hooks must be called in the same order across renders.
  // readOnly + empty content returns null AFTER the hooks, so toggling
  // these props between renders must not throw a hook order violation.
  it("returns null when readOnly + no content, then renders cleanly when content arrives", () => {
    const props = {
      content: "",
      label: "About",
      placeholder: "Tell us more",
      readOnly: true,
    }
    const { container, rerender } = render(
      <NextIntlClientProvider locale="en" messages={messages}>
        <ProfileAboutCard {...props} />
      </NextIntlClientProvider>,
    )
    expect(container.firstChild).toBeNull()

    rerender(
      <NextIntlClientProvider locale="en" messages={messages}>
        <ProfileAboutCard {...props} content="Now visible" />
      </NextIntlClientProvider>,
    )
    expect(screen.getByText("Now visible")).toBeInTheDocument()
  })
})
