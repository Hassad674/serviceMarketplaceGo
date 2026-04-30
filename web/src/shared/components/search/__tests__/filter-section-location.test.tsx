import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { FilterSectionLocation } from "../filter-section-location"
import type { SearchWorkMode } from "../search-filters"

function renderSection(opts: {
  city?: string
  countryCode?: string
  radiusKm?: number | null
  workModes?: SearchWorkMode[]
} = {}) {
  const onCityChange = vi.fn()
  const onCountryChange = vi.fn()
  const onRadiusChange = vi.fn()
  const onWorkModesChange = vi.fn()
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <FilterSectionLocation
        city={opts.city ?? ""}
        countryCode={opts.countryCode ?? ""}
        radiusKm={opts.radiusKm ?? null}
        workModes={opts.workModes ?? []}
        onCityChange={onCityChange}
        onCountryChange={onCountryChange}
        onRadiusChange={onRadiusChange}
        onWorkModesChange={onWorkModesChange}
      />
    </NextIntlClientProvider>,
  )
  return {
    onCityChange,
    onCountryChange,
    onRadiusChange,
    onWorkModesChange,
  }
}

describe("FilterSectionLocation", () => {
  it("renders both 'Location' and 'Work mode' section headings", () => {
    renderSection()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.location }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.workMode }),
    ).toBeInTheDocument()
  })

  it("renders the 3 work-mode pills", () => {
    renderSection()
    expect(
      screen.getByRole("button", { name: messages.search.filters.remote }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: messages.search.filters.onSite }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: messages.search.filters.hybrid }),
    ).toBeInTheDocument()
  })

  it("emits onCityChange when the city input is typed in", () => {
    const { onCityChange } = renderSection()
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.cityPlaceholder),
      { target: { value: "Paris" } },
    )
    expect(onCityChange).toHaveBeenCalledWith("Paris")
  })

  it("uppercases and clamps the country code to 2 chars", () => {
    const { onCountryChange } = renderSection()
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.countryPlaceholder),
      { target: { value: "fra" } },
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

  it("toggles a work mode on/off (additive)", () => {
    const { onWorkModesChange } = renderSection({ workModes: [] })
    fireEvent.click(screen.getByRole("button", { name: messages.search.filters.remote }))
    expect(onWorkModesChange).toHaveBeenCalledWith(["remote"])
  })

  it("toggles a work mode off when already selected", () => {
    const { onWorkModesChange } = renderSection({ workModes: ["remote", "hybrid"] })
    fireEvent.click(screen.getByRole("button", { name: messages.search.filters.remote }))
    expect(onWorkModesChange).toHaveBeenCalledWith(["hybrid"])
  })
})
