import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { FilterSectionLocation } from "../filter-section-location"

function renderSection(opts: {
  city?: string
  countryCode?: string
  radiusKm?: number | null
} = {}) {
  const onCityChange = vi.fn()
  const onCountryChange = vi.fn()
  const onRadiusChange = vi.fn()
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <FilterSectionLocation
        city={opts.city ?? ""}
        countryCode={opts.countryCode ?? ""}
        radiusKm={opts.radiusKm ?? null}
        onCityChange={onCityChange}
        onCountryChange={onCountryChange}
        onRadiusChange={onRadiusChange}
      />
    </NextIntlClientProvider>,
  )
  return {
    onCityChange,
    onCountryChange,
    onRadiusChange,
  }
}

describe("FilterSectionLocation", () => {
  it("renders the 'Location' section heading", () => {
    renderSection()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.location }),
    ).toBeInTheDocument()
  })

  it("does NOT render any work-mode pills (work mode now lives in its own section)", () => {
    renderSection()
    expect(
      screen.queryByRole("button", { name: messages.search.filters.remote }),
    ).toBeNull()
    expect(
      screen.queryByRole("button", { name: messages.search.filters.onSite }),
    ).toBeNull()
    expect(
      screen.queryByRole("button", { name: messages.search.filters.hybrid }),
    ).toBeNull()
  })

  it("renders a country select dropdown with the placeholder", () => {
    renderSection()
    expect(
      screen.getByLabelText(messages.search.filters.countryPlaceholder),
    ).toHaveProperty("tagName", "SELECT")
  })

  it("emits onCountryChange when the country select changes", () => {
    const { onCountryChange } = renderSection()
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.countryPlaceholder),
      { target: { value: "FR" } },
    )
    expect(onCountryChange).toHaveBeenCalledWith("FR")
  })

  it("emits null when the radius input is cleared", () => {
    const { onRadiusChange } = renderSection({ radiusKm: 50 })
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.radiusPlaceholder),
      { target: { value: "" } },
    )
    expect(onRadiusChange).toHaveBeenCalledWith(null)
  })
})
