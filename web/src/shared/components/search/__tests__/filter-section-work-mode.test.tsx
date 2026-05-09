import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { FilterSectionWorkMode } from "../filter-section-work-mode"
import type { SearchWorkMode } from "../search-filters"

function renderSection(opts: { workModes?: SearchWorkMode[] } = {}) {
  const onWorkModesChange = vi.fn()
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <FilterSectionWorkMode
        workModes={opts.workModes ?? []}
        onWorkModesChange={onWorkModesChange}
      />
    </NextIntlClientProvider>,
  )
  return { onWorkModesChange }
}

describe("FilterSectionWorkMode", () => {
  it("renders the section heading", () => {
    renderSection()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.workMode }),
    ).toBeInTheDocument()
  })

  it("renders 3 toggle pills (remote, on site, hybrid)", () => {
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

  it("toggles a mode on when none selected", () => {
    const { onWorkModesChange } = renderSection({ workModes: [] })
    fireEvent.click(
      screen.getByRole("button", { name: messages.search.filters.remote }),
    )
    expect(onWorkModesChange).toHaveBeenCalledWith(["remote"])
  })

  it("toggles a mode off when already selected", () => {
    const { onWorkModesChange } = renderSection({
      workModes: ["remote", "hybrid"],
    })
    fireEvent.click(
      screen.getByRole("button", { name: messages.search.filters.remote }),
    )
    expect(onWorkModesChange).toHaveBeenCalledWith(["hybrid"])
  })
})
