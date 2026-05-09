import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { SharedLocationSection } from "../shared-location-section"

/**
 * shared-location-section.test.tsx — verifies the editable profile
 * location card uses the same shared form primitives as the search
 * filter sidebar (CountrySelect + Photon CityAutocomplete).
 *
 * The previous implementation referenced the BAN-only catalogue
 * directly; this test pins the new contract so a regression that
 * swaps the shared select back to a French-only dropdown fails fast.
 */

const mockUseShared = vi.fn()
const mockMutate = vi.fn()

vi.mock("../../hooks/use-organization-shared", () => ({
  useOrganizationShared: () => mockUseShared(),
}))

vi.mock("../../hooks/use-update-organization-location", () => ({
  useUpdateOrganizationLocation: () => ({
    mutate: mockMutate,
    isPending: false,
  }),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockUseShared.mockReturnValue({
    data: {
      city: "",
      country_code: "",
      latitude: null,
      longitude: null,
      work_mode: [],
      travel_radius_km: null,
    },
  })
})

function renderSection() {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SharedLocationSection />
    </NextIntlClientProvider>,
  )
}

describe("SharedLocationSection (shared picker integration)", () => {
  it("renders a country select dropdown wired to the new shared CountrySelect", () => {
    renderSection()
    const countryLabel = messages.profile.location.countryLabel
    const select = screen.getByLabelText(countryLabel)
    expect(select.tagName).toBe("SELECT")
  })

  it("offers ISO country options with flag emojis (Photon-friendly catalog)", () => {
    renderSection()
    // The catalogue lives in shared/lib/profile and is the same one
    // the search filter uses — flag emoji + locale-aware label.
    expect(screen.getByText(/🇫🇷 France/)).toBeInTheDocument()
    expect(screen.getByText(/🇪🇸 Spain/)).toBeInTheDocument()
  })

  it("commits the country change through the form select", () => {
    renderSection()
    const countryLabel = messages.profile.location.countryLabel
    const select = screen.getByLabelText(countryLabel) as HTMLSelectElement
    fireEvent.change(select, { target: { value: "DE" } })
    // The select reflects the user's choice.
    expect(select.value).toBe("DE")
  })

  it("renders the city autocomplete combobox (not a plain text input)", () => {
    renderSection()
    // CityAutocomplete uses role=combobox with aria-autocomplete=list.
    // We pin the autocomplete attribute to differentiate it from the
    // native <select> (which getByRole also surfaces as a combobox).
    const comboboxes = screen.getAllByRole("combobox")
    const cityCombobox = comboboxes.find(
      (el) => el.getAttribute("aria-autocomplete") === "list",
    )
    expect(cityCombobox).toBeDefined()
  })
})
