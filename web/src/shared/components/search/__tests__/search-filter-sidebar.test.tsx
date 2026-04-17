import { describe, expect, it, vi } from "vitest"
import { fireEvent, render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import type { SearchDocumentPersona } from "@/shared/lib/search/search-document"
import { SearchFilterSidebar } from "../search-filter-sidebar"
import {
  EMPTY_SEARCH_FILTERS,
  type SearchFilters,
} from "../search-filters"

interface RenderOptions {
  filters?: Partial<SearchFilters>
  persona?: SearchDocumentPersona
}

function renderSidebar(options: RenderOptions = {}) {
  const filters: SearchFilters = {
    ...EMPTY_SEARCH_FILTERS,
    ...(options.filters ?? {}),
  }
  const onChange = vi.fn()
  const onApply = vi.fn()
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SearchFilterSidebar
        filters={filters}
        onChange={onChange}
        onApply={onApply}
        resultsCount={42}
        persona={options.persona}
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
    renderSidebar({ filters: { availability: "now" } })
    expect(
      screen.getByRole("button", { name: messages.search.filters.reset }),
    ).toBeInTheDocument()
  })

  it("resets every filter when the reset button is clicked", () => {
    const { onChange } = renderSidebar({
      filters: {
        availability: "now",
        languages: ["fr", "en"],
        minRating: 4,
      },
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

  // V1 pricing simplification: the price section must relabel itself
  // per persona so the filter matches the primary pricing shape shown
  // on the cards (TJM / Budget / Commission). Table-driven to stay
  // green across all three personas and the undefined fallback.
  it.each<{
    name: string
    persona: SearchDocumentPersona | undefined
    expectedTitle: string
    expectedMin: string
    expectedMax: string
  }>([
    {
      name: "freelance",
      persona: "freelance",
      expectedTitle: messages.search.filters.freelancePrice,
      expectedMin: messages.search.filters.freelancePriceMin,
      expectedMax: messages.search.filters.freelancePriceMax,
    },
    {
      name: "agency",
      persona: "agency",
      expectedTitle: messages.search.filters.agencyPrice,
      expectedMin: messages.search.filters.agencyPriceMin,
      expectedMax: messages.search.filters.agencyPriceMax,
    },
    {
      name: "referrer",
      persona: "referrer",
      expectedTitle: messages.search.filters.referrerPrice,
      expectedMin: messages.search.filters.referrerPriceMin,
      expectedMax: messages.search.filters.referrerPriceMax,
    },
    {
      name: "undefined fallback",
      persona: undefined,
      expectedTitle: messages.search.filters.price,
      expectedMin: messages.search.filters.priceMin,
      expectedMax: messages.search.filters.priceMax,
    },
  ])(
    "uses persona-aware price labels: $name",
    ({ persona, expectedTitle, expectedMin, expectedMax }) => {
      renderSidebar({ persona })
      expect(
        screen.getByRole("heading", { name: expectedTitle }),
      ).toBeInTheDocument()
      expect(screen.getByLabelText(expectedMin)).toBeInTheDocument()
      expect(screen.getByLabelText(expectedMax)).toBeInTheDocument()
    },
  )
})
