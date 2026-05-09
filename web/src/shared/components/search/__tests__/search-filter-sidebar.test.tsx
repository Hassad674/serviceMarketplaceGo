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

  it("renders every section header (freelance default — full visibility)", () => {
    // Default sidebar (no persona) renders every section. Persona-
    // specific visibility is covered by the dedicated test cases.
    renderSidebar({ persona: "freelance" })
    for (const label of [
      messages.search.filters.availability,
      messages.search.filters.freelancePrice,
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

  it("hides the work-mode section for the agency persona", () => {
    renderSidebar({ persona: "agency" })
    expect(
      screen.queryByRole("heading", { name: messages.search.filters.workMode }),
    ).toBeNull()
    // Skills + pricing stay visible for agencies.
    expect(
      screen.getByRole("heading", { name: messages.search.filters.skills }),
    ).toBeInTheDocument()
  })

  it("hides work-mode + skills + pricing for the referrer persona", () => {
    renderSidebar({ persona: "referrer" })
    expect(
      screen.queryByRole("heading", { name: messages.search.filters.workMode }),
    ).toBeNull()
    expect(
      screen.queryByRole("heading", { name: messages.search.filters.skills }),
    ).toBeNull()
    // Referrer pricing is the commission section title — it should not
    // be visible because the whole pricing block is hidden.
    expect(
      screen.queryByRole("heading", {
        name: messages.search.filters.referrerPrice,
      }),
    ).toBeNull()
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

  it("fires onChange when a language is committed from the combobox", () => {
    const { onChange } = renderSidebar()
    const input = screen.getByLabelText(messages.search.filters.languages)
    fireEvent.change(input, { target: { value: "Fren" } })
    fireEvent.keyDown(input, { key: "Enter" })
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

  // ---------------------------------------------------------------------
  // Orchestrator → child wiring — make sure each arrow callback in
  // the SearchFilterSidebar template actually delivers the expected
  // partial update through `onChange`.
  // ---------------------------------------------------------------------
  it("forwards country select changes through onChange", () => {
    const { onChange } = renderSidebar()
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.countryPlaceholder),
      { target: { value: "ES" } },
    )
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ countryCode: "ES" }),
    )
  })

  it("forwards radius changes through onChange", () => {
    const { onChange } = renderSidebar()
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.radiusPlaceholder),
      { target: { value: "50" } },
    )
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ radiusKm: 50 }),
    )
  })

  it("forwards skill additions through onChange", () => {
    const { onChange } = renderSidebar()
    const input = screen.getByLabelText(
      messages.search.filters.skillsSearchPlaceholder,
    )
    fireEvent.change(input, { target: { value: "Rust" } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ skills: ["Rust"] }),
    )
  })

  it("forwards minRating star clicks through onChange", () => {
    const { onChange } = renderSidebar()
    fireEvent.click(screen.getByRole("radio", { name: "5" }))
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ minRating: 5 }),
    )
  })

  it("forwards expertise checkbox clicks through onChange", () => {
    const { onChange } = renderSidebar()
    const checkboxes = screen.getAllByRole("checkbox")
    fireEvent.click(checkboxes[0])
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ expertise: expect.any(Array) }),
    )
  })

  it("forwards price min changes through onChange", () => {
    const { onChange } = renderSidebar({ persona: "freelance" })
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.freelancePriceMin),
      { target: { value: "300" } },
    )
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ priceMin: 300 }),
    )
  })

  it("forwards price max changes through onChange", () => {
    const { onChange } = renderSidebar({ persona: "agency" })
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.agencyPriceMax),
      { target: { value: "10000" } },
    )
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ priceMax: 10000 }),
    )
  })

  // V1 pricing simplification: the price section must relabel itself
  // per persona so the filter matches the primary pricing shape shown
  // on the cards (TJM / Budget / Commission). Referrer hides pricing
  // entirely (per per-persona visibility config), so it is excluded
  // from this table — covered by the dedicated visibility test above.
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
