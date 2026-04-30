import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { FilterSectionRating } from "../filter-section-rating"

function renderSection(value = 0) {
  const onChange = vi.fn()
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <FilterSectionRating value={value} onChange={onChange} />
    </NextIntlClientProvider>,
  )
  return { onChange }
}

describe("FilterSectionRating", () => {
  it("renders the section heading + 5 star radios", () => {
    renderSection()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.rating }),
    ).toBeInTheDocument()
    const radios = screen.getAllByRole("radio")
    expect(radios).toHaveLength(5)
  })

  it("marks the radio as checked up to the current value", () => {
    renderSection(3)
    const radios = screen.getAllByRole("radio")
    expect(radios[0]).toHaveAttribute("aria-checked", "true")
    expect(radios[1]).toHaveAttribute("aria-checked", "true")
    expect(radios[2]).toHaveAttribute("aria-checked", "true")
    expect(radios[3]).toHaveAttribute("aria-checked", "false")
    expect(radios[4]).toHaveAttribute("aria-checked", "false")
  })

  it("emits the new floor when clicking a different star", () => {
    const { onChange } = renderSection(0)
    fireEvent.click(screen.getByRole("radio", { name: "4" }))
    expect(onChange).toHaveBeenCalledWith(4)
  })

  it("clears (returns 0) when clicking the same star already selected", () => {
    const { onChange } = renderSection(3)
    fireEvent.click(screen.getByRole("radio", { name: "3" }))
    expect(onChange).toHaveBeenCalledWith(0)
  })
})
