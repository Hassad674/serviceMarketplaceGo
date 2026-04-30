import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { CityAutocomplete, type CitySelection } from "../city-autocomplete"

// The component fans out to the city-search lib. Mocking it lets us
// test the component glue (debounce, keyboard nav, ARIA wiring,
// selection lifecycle) instead of re-testing the network layer that
// already has its own dedicated test file.
vi.mock("@/shared/lib/location/city-search", async () => {
  const actual = await vi.importActual<
    typeof import("@/shared/lib/location/city-search")
  >("@/shared/lib/location/city-search")
  return {
    ...actual,
    searchCities: vi.fn(),
  }
})

import { searchCities } from "@/shared/lib/location/city-search"

const mockedSearchCities = vi.mocked(searchCities)

const PARIS_RESULT = {
  city: "Paris",
  postcode: "75001",
  countryCode: "FR",
  latitude: 48.8566,
  longitude: 2.3522,
  context: "75 · Île-de-France",
}

const LYON_RESULT = {
  city: "Lyon",
  postcode: "69001",
  countryCode: "FR",
  latitude: 45.764,
  longitude: 4.8357,
  context: "69 · Auvergne-Rhône-Alpes",
}

// The component debounces the search by 250ms. We wait slightly longer
// than that in tests rather than mocking timers — fake timers play
// poorly with React 19's async transitions and the @testing-library
// async helpers.
const DEBOUNCE_MS = 300

function renderAutocomplete(props: {
  value?: CitySelection | null
  countryCode?: string
  disabled?: boolean
  onChange?: (next: CitySelection | null) => void
}) {
  const onChange = props.onChange ?? vi.fn()
  const utils = render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <CityAutocomplete
        value={props.value ?? null}
        countryCode={props.countryCode ?? "FR"}
        onChange={onChange}
        disabled={props.disabled}
      />
    </NextIntlClientProvider>,
  )
  return { ...utils, onChange }
}

describe("CityAutocomplete", () => {
  beforeEach(() => {
    mockedSearchCities.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("renders an empty input with the placeholder when no value is set", () => {
    renderAutocomplete({ value: null })
    const input = screen.getByRole("combobox")
    expect(input).toHaveValue("")
    expect(input).toHaveAttribute("placeholder", "Search for a city…")
  })

  it("hydrates the visible query from the persisted value", () => {
    renderAutocomplete({
      value: { city: "Paris", countryCode: "FR", latitude: 1, longitude: 2 },
    })
    expect(screen.getByRole("combobox")).toHaveValue("Paris")
  })

  it("shows the hint message when the query is below the minimum length", () => {
    renderAutocomplete({})
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "P" } })
    expect(
      screen.getByText("Type at least 2 characters to search"),
    ).toBeInTheDocument()
    expect(mockedSearchCities).not.toHaveBeenCalled()
  })

  it("debounces and fires the city search after the user types enough characters", async () => {
    mockedSearchCities.mockResolvedValue([PARIS_RESULT])

    renderAutocomplete({})
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Par" } })

    await waitFor(
      () => {
        expect(mockedSearchCities).toHaveBeenCalled()
      },
      { timeout: 1000 },
    )
    expect(await screen.findByText("Paris")).toBeInTheDocument()
  })

  it("commits the typed selection when the user presses Enter", async () => {
    mockedSearchCities.mockResolvedValue([PARIS_RESULT])
    const onChange = vi.fn()

    renderAutocomplete({ onChange })
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Par" } })

    await screen.findByText("Paris", undefined, { timeout: 1000 })

    fireEvent.keyDown(input, { key: "Enter" })

    expect(onChange).toHaveBeenLastCalledWith({
      city: "Paris",
      countryCode: "FR",
      latitude: 48.8566,
      longitude: 2.3522,
    })
  })

  it("navigates results with arrow keys and commits the highlighted row on Enter", async () => {
    mockedSearchCities.mockResolvedValue([PARIS_RESULT, LYON_RESULT])
    const onChange = vi.fn()

    renderAutocomplete({ onChange })
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Pa" } })

    await screen.findByText("Lyon", undefined, { timeout: 1000 })

    fireEvent.keyDown(input, { key: "ArrowDown" })
    fireEvent.keyDown(input, { key: "Enter" })

    expect(onChange).toHaveBeenLastCalledWith({
      city: "Lyon",
      countryCode: "FR",
      latitude: 45.764,
      longitude: 4.8357,
    })
  })

  it("ArrowUp wraps back to the last item from the first", async () => {
    mockedSearchCities.mockResolvedValue([PARIS_RESULT, LYON_RESULT])
    const onChange = vi.fn()

    renderAutocomplete({ onChange })
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Pa" } })

    await screen.findByText("Lyon", undefined, { timeout: 1000 })

    // index starts at 0 → ArrowUp wraps to last (Lyon) → Enter.
    fireEvent.keyDown(input, { key: "ArrowUp" })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(onChange).toHaveBeenLastCalledWith({
      city: "Lyon",
      countryCode: "FR",
      latitude: 45.764,
      longitude: 4.8357,
    })
  })

  it("invalidates the persisted selection (calls onChange(null)) when the user retypes", () => {
    const onChange = vi.fn()
    renderAutocomplete({
      value: {
        city: "Paris",
        countryCode: "FR",
        latitude: 1,
        longitude: 2,
      },
      onChange,
    })

    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Par" } })

    expect(onChange).toHaveBeenCalledWith(null)
  })

  it("commits the selection when the user clicks (mousedown) a row", async () => {
    mockedSearchCities.mockResolvedValue([PARIS_RESULT])
    const onChange = vi.fn()

    renderAutocomplete({ onChange })
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Par" } })

    const option = await screen.findByText("Paris", undefined, {
      timeout: 1000,
    })
    fireEvent.mouseDown(option)

    expect(onChange).toHaveBeenLastCalledWith({
      city: "Paris",
      countryCode: "FR",
      latitude: 48.8566,
      longitude: 2.3522,
    })
  })

  it("Escape closes the dropdown and restores the canonical city in the input", async () => {
    mockedSearchCities.mockResolvedValue([PARIS_RESULT])

    renderAutocomplete({
      value: {
        city: "Paris",
        countryCode: "FR",
        latitude: 1,
        longitude: 2,
      },
    })

    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Pari" } })
    await screen.findByText("Paris", undefined, { timeout: 1000 })

    fireEvent.keyDown(input, { key: "Escape" })
    expect(input).toHaveValue("Paris")
  })

  it("reports the empty-state message when the search returns no city", async () => {
    mockedSearchCities.mockResolvedValue([])

    renderAutocomplete({})
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Zzz" } })

    await waitFor(
      () => {
        expect(screen.getByText("No city found")).toBeInTheDocument()
      },
      { timeout: 1000 },
    )
  })

  it("respects the `disabled` prop by disabling the input", () => {
    renderAutocomplete({ disabled: true })
    expect(screen.getByRole("combobox")).toBeDisabled()
  })

  it("does not query when the typed text equals the already-selected city", async () => {
    renderAutocomplete({
      value: {
        city: "Paris",
        countryCode: "FR",
        latitude: 1,
        longitude: 2,
      },
    })
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    // Simulate a synthetic re-render where the query equals value.city.
    fireEvent.change(input, { target: { value: "Paris" } })
    // Wait past the debounce window — verify nothing was fetched.
    await new Promise((resolve) => setTimeout(resolve, DEBOUNCE_MS))
    expect(mockedSearchCities).not.toHaveBeenCalled()
  })

  it("ignores AbortError thrown by the underlying search", async () => {
    const abortError = new DOMException("aborted", "AbortError")
    mockedSearchCities.mockRejectedValueOnce(abortError)

    renderAutocomplete({})
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Par" } })

    await waitFor(
      () => {
        expect(mockedSearchCities).toHaveBeenCalled()
      },
      { timeout: 1000 },
    )
    // Component must not crash; query value preserved.
    expect(input).toHaveValue("Par")
  })

  it("recovers from a non-abort error by clearing results without crashing", async () => {
    mockedSearchCities.mockRejectedValueOnce(new Error("boom"))

    renderAutocomplete({})
    const input = screen.getByRole("combobox")
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: "Par" } })

    await waitFor(
      () => {
        expect(mockedSearchCities).toHaveBeenCalled()
      },
      { timeout: 1000 },
    )
    // No "Paris" row would be rendered.
    expect(screen.queryByText("Paris")).toBeNull()
  })
})
