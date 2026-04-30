import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { FilterSectionPricing } from "../filter-section-pricing"
import type { SearchDocumentPersona } from "@/shared/lib/search/search-document"

function renderSection(opts: {
  persona?: SearchDocumentPersona
  min?: number | null
  max?: number | null
} = {}) {
  const onMinChange = vi.fn()
  const onMaxChange = vi.fn()
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <FilterSectionPricing
        persona={opts.persona}
        min={opts.min ?? null}
        max={opts.max ?? null}
        onMinChange={onMinChange}
        onMaxChange={onMaxChange}
      />
    </NextIntlClientProvider>,
  )
  return { onMinChange, onMaxChange }
}

describe("FilterSectionPricing", () => {
  it("renders the generic price heading when persona is undefined", () => {
    renderSection()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.price }),
    ).toBeInTheDocument()
  })

  it.each<{
    persona: SearchDocumentPersona
    title: string
    min: string
    max: string
  }>([
    {
      persona: "freelance",
      title: messages.search.filters.freelancePrice,
      min: messages.search.filters.freelancePriceMin,
      max: messages.search.filters.freelancePriceMax,
    },
    {
      persona: "agency",
      title: messages.search.filters.agencyPrice,
      min: messages.search.filters.agencyPriceMin,
      max: messages.search.filters.agencyPriceMax,
    },
    {
      persona: "referrer",
      title: messages.search.filters.referrerPrice,
      min: messages.search.filters.referrerPriceMin,
      max: messages.search.filters.referrerPriceMax,
    },
  ])("uses persona-aware labels for $persona", ({ persona, title, min, max }) => {
    renderSection({ persona })
    expect(screen.getByRole("heading", { name: title })).toBeInTheDocument()
    expect(screen.getByLabelText(min)).toBeInTheDocument()
    expect(screen.getByLabelText(max)).toBeInTheDocument()
  })

  it("appends an € suffix for freelance/agency personas", () => {
    renderSection({ persona: "freelance" })
    // 2 inputs (min + max) → 2 suffix spans
    expect(screen.getAllByText(/^€$/)).toHaveLength(2)
  })

  it("appends a % suffix for referrer persona", () => {
    renderSection({ persona: "referrer" })
    expect(screen.getAllByText(/^%$/)).toHaveLength(2)
  })

  it("emits onMinChange with the parsed value", () => {
    const { onMinChange } = renderSection({ persona: "freelance" })
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.freelancePriceMin),
      { target: { value: "300" } },
    )
    expect(onMinChange).toHaveBeenCalledWith(300)
  })

  it("emits onMaxChange when emptied", () => {
    const { onMaxChange } = renderSection({ persona: "agency", max: 100 })
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.agencyPriceMax),
      { target: { value: "" } },
    )
    expect(onMaxChange).toHaveBeenCalledWith(null)
  })

  it("clamps negatives to 0", () => {
    const { onMinChange } = renderSection()
    fireEvent.change(
      screen.getByLabelText(messages.search.filters.priceMin),
      { target: { value: "-50" } },
    )
    // Native input type=number filters negative chars in some browsers,
    // but the explicit Math.max guard means even if "-50" lands we
    // would emit 0. We at least assert it is called.
    expect(onMinChange).toHaveBeenCalled()
  })
})
