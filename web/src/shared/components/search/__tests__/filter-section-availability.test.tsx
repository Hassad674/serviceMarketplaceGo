import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { FilterSectionAvailability } from "../filter-section-availability"
import type { SearchAvailabilityFilter } from "../search-filters"

function renderSection(value: SearchAvailabilityFilter = "all") {
  const onChange = vi.fn()
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <FilterSectionAvailability value={value} onChange={onChange} />
    </NextIntlClientProvider>,
  )
  return { onChange }
}

describe("FilterSectionAvailability", () => {
  it("renders the section heading", () => {
    renderSection()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.availability }),
    ).toBeInTheDocument()
  })

  it("renders the 3 availability pills", () => {
    renderSection()
    expect(
      screen.getByRole("button", { name: messages.search.filters.availableNow }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: messages.search.filters.availableSoon }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: messages.search.filters.availabilityAll }),
    ).toBeInTheDocument()
  })

  it("marks the selected pill via aria-pressed=true", () => {
    renderSection("now")
    const pill = screen.getByRole("button", {
      name: messages.search.filters.availableNow,
    })
    expect(pill).toHaveAttribute("aria-pressed", "true")
  })

  it("emits the new value when a different pill is clicked", () => {
    const { onChange } = renderSection("all")
    fireEvent.click(
      screen.getByRole("button", { name: messages.search.filters.availableSoon }),
    )
    expect(onChange).toHaveBeenCalledWith("soon")
  })

  it("emits the same value when the already-selected pill is clicked (no toggle)", () => {
    const { onChange } = renderSection("now")
    fireEvent.click(
      screen.getByRole("button", { name: messages.search.filters.availableNow }),
    )
    expect(onChange).toHaveBeenCalledWith("now")
  })
})
