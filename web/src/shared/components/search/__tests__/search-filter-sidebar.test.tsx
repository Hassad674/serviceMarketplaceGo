import { describe, expect, it, vi } from "vitest"
import { fireEvent, render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { SearchFilterSidebar } from "../search-filter-sidebar"
import {
  EMPTY_SEARCH_FILTERS,
  type SearchFilters,
} from "../search-filters"

function renderSidebar(overrides: Partial<SearchFilters> = {}) {
  const filters: SearchFilters = { ...EMPTY_SEARCH_FILTERS, ...overrides }
  const onChange = vi.fn()
  const onApply = vi.fn()
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SearchFilterSidebar
        filters={filters}
        onChange={onChange}
        onApply={onApply}
        resultsCount={42}
      />
    </NextIntlClientProvider>,
  )
  return { onChange, onApply }
}

describe("SearchFilterSidebar", () => {
  it("renders the title and results count", () => {
    renderSidebar()
    expect(
      screen.getByRole("complementary", {
        name: messages.search.filters.title,
      }),
    ).toBeInTheDocument()
    expect(screen.getByText(/42/)).toBeInTheDocument()
  })

  it("renders every section header", () => {
    renderSidebar()
    for (const label of [
      messages.search.filters.availability,
      messages.search.filters.price,
      messages.search.filters.location,
      messages.search.filters.languages,
      messages.search.filters.expertise,
      messages.search.filters.skills,
      messages.search.filters.rating,
      messages.search.filters.workMode,
    ]) {
      expect(screen.getByRole("heading", { name: label })).toBeInTheDocument()
    }
  })

  it("fires onChange when an availability pill is clicked", () => {
    const { onChange } = renderSidebar()
    fireEvent.click(
      screen.getByRole("button", {
        name: messages.search.filters.availableNow,
      }),
    )
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ availability: "now" }),
    )
  })

  it("fires onChange when a work mode pill is clicked", () => {
    const { onChange } = renderSidebar()
    fireEvent.click(
      screen.getByRole("button", { name: messages.search.filters.remote }),
    )
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ workModes: ["remote"] }),
    )
  })

  it("fires onChange with toggled language selection", () => {
    const { onChange } = renderSidebar()
    fireEvent.click(screen.getByRole("button", { name: "FR" }))
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ languages: ["fr"] }),
    )
  })

  it("hides the reset button when filters are empty", () => {
    renderSidebar()
    expect(
      screen.queryByRole("button", { name: messages.search.filters.reset }),
    ).not.toBeInTheDocument()
  })

  it("shows the reset button when any filter is set", () => {
    renderSidebar({ availability: "now" })
    expect(
      screen.getByRole("button", { name: messages.search.filters.reset }),
    ).toBeInTheDocument()
  })

  it("resets every filter when the reset button is clicked", () => {
    const { onChange } = renderSidebar({
      availability: "now",
      languages: ["fr", "en"],
      minRating: 4,
    })
    fireEvent.click(
      screen.getByRole("button", { name: messages.search.filters.reset }),
    )
    expect(onChange).toHaveBeenCalledWith(EMPTY_SEARCH_FILTERS)
  })

  it("calls onApply when apply is clicked", () => {
    const { onApply } = renderSidebar()
    fireEvent.click(
      screen.getByRole("button", { name: messages.search.filters.apply }),
    )
    expect(onApply).toHaveBeenCalledTimes(1)
  })
})
