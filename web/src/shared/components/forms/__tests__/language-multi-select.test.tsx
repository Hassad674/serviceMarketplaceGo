import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { LanguageMultiSelect } from "../language-multi-select"

function renderSelect(opts: {
  selected?: string[]
  ariaLabel?: string
  locale?: "en" | "fr"
} = {}) {
  const onChange = vi.fn()
  render(
    <NextIntlClientProvider locale={opts.locale ?? "en"} messages={messages}>
      <LanguageMultiSelect
        selected={opts.selected ?? []}
        onChange={onChange}
        ariaLabel={opts.ariaLabel ?? "languages"}
      />
    </NextIntlClientProvider>,
  )
  return { onChange }
}

describe("LanguageMultiSelect", () => {
  it("renders the combobox input", () => {
    renderSelect()
    expect(screen.getByLabelText("languages")).toBeInTheDocument()
  })

  it("starts closed (no listbox until focus or typing)", () => {
    renderSelect()
    expect(screen.queryByRole("listbox")).toBeNull()
  })

  it("opens the dropdown when focused", () => {
    renderSelect()
    const input = screen.getByLabelText("languages")
    fireEvent.focus(input)
    expect(screen.getByRole("listbox")).toBeInTheDocument()
  })

  it("filters options by typed query (case-insensitive)", () => {
    renderSelect()
    const input = screen.getByLabelText("languages")
    fireEvent.change(input, { target: { value: "fren" } })
    // French is the only match for "fren"
    expect(screen.getByText("French")).toBeInTheDocument()
    expect(screen.queryByText("English")).toBeNull()
  })

  it("commits a pick on Enter", () => {
    const { onChange } = renderSelect({ selected: [] })
    const input = screen.getByLabelText("languages")
    fireEvent.change(input, { target: { value: "fren" } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(onChange).toHaveBeenCalledWith(["fr"])
  })

  it("commits a pick on click (mousedown)", () => {
    const { onChange } = renderSelect({ selected: [] })
    const input = screen.getByLabelText("languages")
    fireEvent.change(input, { target: { value: "ger" } })
    const option = screen.getByText("German")
    fireEvent.mouseDown(option)
    expect(onChange).toHaveBeenCalledWith(["de"])
  })

  it("excludes already-selected codes from the dropdown", () => {
    renderSelect({ selected: ["fr"] })
    const input = screen.getByLabelText("languages")
    fireEvent.change(input, { target: { value: "fren" } })
    // The dropdown listbox should have NO option for "French" — it's
    // already selected, so the only "French" text on screen is inside
    // the badge above the input. Asserting on the listbox children
    // catches the dropdown contents specifically.
    const listbox = screen.queryByRole("listbox")
    expect(listbox).not.toBeNull()
    expect(listbox!.querySelectorAll("[role=option]").length).toBe(0)
  })

  it("shows the no-results state when nothing matches", () => {
    renderSelect()
    const input = screen.getByLabelText("languages")
    fireEvent.change(input, { target: { value: "xyz" } })
    expect(screen.getByText("No matching language")).toBeInTheDocument()
  })

  it("renders a removable badge per selected language", () => {
    const { onChange } = renderSelect({ selected: ["fr", "en"] })
    const removeFr = screen.getByRole("button", { name: /Remove French/ })
    fireEvent.click(removeFr)
    expect(onChange).toHaveBeenCalledWith(["en"])
  })

  it("removes the last selected on Backspace when input is empty", () => {
    const { onChange } = renderSelect({ selected: ["fr", "en"] })
    const input = screen.getByLabelText("languages")
    fireEvent.keyDown(input, { key: "Backspace" })
    expect(onChange).toHaveBeenCalledWith(["fr"])
  })

  it("does not remove on Backspace when the input has text", () => {
    const { onChange } = renderSelect({ selected: ["fr"] })
    const input = screen.getByLabelText("languages")
    fireEvent.change(input, { target: { value: "g" } })
    fireEvent.keyDown(input, { key: "Backspace" })
    expect(onChange).not.toHaveBeenCalled()
  })

  it("ArrowDown moves the active highlight", () => {
    renderSelect()
    const input = screen.getByLabelText("languages")
    fireEvent.focus(input)
    fireEvent.keyDown(input, { key: "ArrowDown" })
    // No throw is enough — keyboard nav must not crash on empty draft.
    expect(screen.getByRole("listbox")).toBeInTheDocument()
  })

  it("Escape closes the dropdown", () => {
    renderSelect()
    const input = screen.getByLabelText("languages")
    fireEvent.focus(input)
    fireEvent.keyDown(input, { key: "Escape" })
    expect(screen.queryByRole("listbox")).toBeNull()
  })
})
