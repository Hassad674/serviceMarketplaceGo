import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { CountrySelect } from "../country-select"

function renderSelect(opts: {
  value?: string
  placeholder?: string
  ariaLabel?: string
  locale?: "en" | "fr"
} = {}) {
  const onChange = vi.fn()
  render(
    <NextIntlClientProvider locale={opts.locale ?? "en"} messages={messages}>
      <CountrySelect
        value={opts.value ?? ""}
        onChange={onChange}
        placeholder={opts.placeholder ?? "Pick a country"}
        ariaLabel={opts.ariaLabel ?? "country"}
      />
    </NextIntlClientProvider>,
  )
  return { onChange }
}

describe("CountrySelect", () => {
  it("renders as a native <select>", () => {
    renderSelect()
    const select = screen.getByLabelText("country")
    expect(select.tagName).toBe("SELECT")
  })

  it("includes the placeholder as a disabled empty option", () => {
    renderSelect({ placeholder: "Pick a country" })
    expect(screen.getByText("Pick a country")).toBeInTheDocument()
  })

  it("emits the selected ISO code through onChange", () => {
    const { onChange } = renderSelect()
    fireEvent.change(screen.getByLabelText("country"), {
      target: { value: "ES" },
    })
    expect(onChange).toHaveBeenCalledWith("ES")
  })

  it("renders FR option with the French flag emoji prefix in EN locale", () => {
    renderSelect({ locale: "en" })
    expect(screen.getByText(/🇫🇷 France/)).toBeInTheDocument()
  })

  it("renders FR option with the French label in FR locale", () => {
    renderSelect({ locale: "fr" })
    // The label uses the FR localised string ("France" stays the same for
    // FR but Spain becomes "Espagne", which we cross-check below).
    expect(screen.getByText(/🇪🇸 Espagne/)).toBeInTheDocument()
  })

  it("preselects the value prop", () => {
    renderSelect({ value: "DE" })
    const select = screen.getByLabelText("country") as HTMLSelectElement
    expect(select.value).toBe("DE")
  })

  it("respects disabled prop", () => {
    const onChange = vi.fn()
    render(
      <NextIntlClientProvider locale="en" messages={messages}>
        <CountrySelect
          value=""
          onChange={onChange}
          ariaLabel="country"
          disabled
        />
      </NextIntlClientProvider>,
    )
    const select = screen.getByLabelText("country") as HTMLSelectElement
    expect(select.disabled).toBe(true)
  })
})
