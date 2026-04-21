import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ClientProfileHeader } from "../client-profile-header"

function renderHeader(
  props: Partial<Parameters<typeof ClientProfileHeader>[0]> = {},
) {
  const defaults = {
    companyName: "Acme Corp",
    avatarUrl: null,
    stats: {
      totalSpent: 0,
      reviewCount: 0,
      averageRating: 0,
      projectsCompleted: 0,
    },
    ...props,
  }

  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ClientProfileHeader {...defaults} />
    </NextIntlClientProvider>,
  )
}

describe("ClientProfileHeader", () => {
  it("renders the company name as the h1", () => {
    renderHeader({ companyName: "Acme Corp" })
    expect(
      screen.getByRole("heading", { level: 1, name: "Acme Corp" }),
    ).toBeInTheDocument()
  })

  it("falls back to initials when no avatar is provided", () => {
    renderHeader({ companyName: "Acme Corp", avatarUrl: null })
    // "AC" is the uppercase 2-char initial fallback.
    expect(screen.getByText("AC")).toBeInTheDocument()
  })

  it("formats total spent from cents to a euro string", () => {
    renderHeader({
      stats: {
        totalSpent: 1234500, // 12 345,00 €
        reviewCount: 0,
        averageRating: 0,
        projectsCompleted: 0,
      },
    })
    // fr-FR currency formatting uses non-breaking space + € suffix;
    // match the leading digit grouping rather than the exact glyph
    // sequence so this does not break when ICU normalizes thinspaces.
    expect(screen.getByText(/12\s?345/)).toBeInTheDocument()
  })

  it("shows the em-dash placeholder when there are no reviews", () => {
    renderHeader({
      stats: {
        totalSpent: 0,
        reviewCount: 0,
        averageRating: 0,
        projectsCompleted: 0,
      },
    })
    expect(screen.getByText("—")).toBeInTheDocument()
  })

  it("renders the averaged rating when at least one review exists", () => {
    renderHeader({
      stats: {
        totalSpent: 0,
        reviewCount: 3,
        averageRating: 4.5,
        projectsCompleted: 2,
      },
    })
    expect(screen.getByText("4.5 / 5")).toBeInTheDocument()
    expect(screen.getByText("3")).toBeInTheDocument()
    expect(screen.getByText("2")).toBeInTheDocument()
  })
})
