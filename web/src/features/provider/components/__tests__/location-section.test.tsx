import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { LocationSection } from "../location-section"
import type { Profile } from "../../api/profile-api"

const baseProfile: Profile = {
  organization_id: "org-1",
  title: "",
  photo_url: "",
  presentation_video_url: "",
  referrer_video_url: "",
  about: "",
  referrer_about: "",
  city: "",
  country_code: "",
  latitude: null,
  longitude: null,
  work_mode: [],
  travel_radius_km: null,
  created_at: "2026-04-01",
  updated_at: "2026-04-01",
}

let profileOverride: Partial<Profile> = {}
const mockMutate = vi.fn()

vi.mock("../../hooks/use-profile", () => ({
  useProfile: () => ({ data: { ...baseProfile, ...profileOverride } }),
  profileQueryKey: () => ["user", "x", "profile"],
}))

vi.mock("../../hooks/use-update-location", () => ({
  useUpdateLocation: () => ({
    mutate: (value: unknown) => mockMutate(value),
    isPending: false,
  }),
}))

// Stub the CityAutocomplete so the section test stays focused on
// the section's own responsibilities (work-mode gating, mutation
// payload shape). The autocomplete itself is covered by its own
// unit test — mixing the two would turn this file into an
// integration test.
vi.mock("../city-autocomplete", () => ({
  CityAutocomplete: ({
    value,
    countryCode,
    onChange,
  }: {
    value: { city: string; countryCode: string; latitude: number; longitude: number } | null
    countryCode: string
    onChange: (next: { city: string; countryCode: string; latitude: number; longitude: number } | null) => void
  }) => (
    <div>
      <span data-testid="autocomplete-value">{value?.city ?? ""}</span>
      <span data-testid="autocomplete-country">{countryCode}</span>
      <button
        type="button"
        data-testid="autocomplete-pick-paris"
        onClick={() =>
          onChange({
            city: "Paris",
            countryCode: "FR",
            latitude: 48.8566,
            longitude: 2.3522,
          })
        }
      >
        pick paris
      </button>
      <button
        type="button"
        data-testid="autocomplete-clear"
        onClick={() => onChange(null)}
      >
        clear
      </button>
    </div>
  ),
}))

function renderSection(
  props: Partial<Parameters<typeof LocationSection>[0]> = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const defaults = {
    orgType: "provider_personal",
    readOnly: false,
  }
  return render(
    <QueryClientProvider client={queryClient}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <LocationSection {...defaults} {...props} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  profileOverride = {}
})

describe("LocationSection", () => {
  it("renders nothing for enterprise", () => {
    const { container } = renderSection({ orgType: "enterprise" })
    expect(container).toBeEmptyDOMElement()
  })

  it("hides the work-mode row for agencies", () => {
    renderSection({ orgType: "agency" })
    expect(
      screen.queryByText(messages.profile.location.workModeLabel),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole("button", {
        name: messages.profile.location.workModeRemote,
      }),
    ).not.toBeInTheDocument()
  })

  it("shows the work-mode row for freelance providers", () => {
    renderSection({ orgType: "provider_personal" })
    expect(
      screen.getByRole("button", { name: messages.profile.location.workModeRemote }),
    ).toBeInTheDocument()
  })

  it("shows the radius input only when on_site or hybrid is selected", async () => {
    const user = userEvent.setup()
    renderSection()
    expect(
      screen.queryByLabelText(messages.profile.location.travelRadiusLabel),
    ).not.toBeInTheDocument()
    await user.click(
      screen.getByRole("button", { name: messages.profile.location.workModeOnSite }),
    )
    expect(
      screen.getByLabelText(messages.profile.location.travelRadiusLabel),
    ).toBeInTheDocument()
  })

  it("submits the canonical city + coords when the user picks a city and saves", async () => {
    const user = userEvent.setup()
    renderSection()

    const countrySelect = screen.getByLabelText(
      messages.profile.location.countryLabel,
    )
    await user.selectOptions(countrySelect, "FR")
    await user.click(screen.getByTestId("autocomplete-pick-paris"))
    await user.click(
      screen.getByRole("button", { name: messages.profile.location.workModeRemote }),
    )

    const saveBtn = screen.getByRole("button", {
      name: new RegExp(messages.profile.location.save, "i"),
    })
    await waitFor(() => expect(saveBtn).toBeEnabled())
    await user.click(saveBtn)

    expect(mockMutate).toHaveBeenCalledWith({
      city: "Paris",
      country_code: "FR",
      latitude: 48.8566,
      longitude: 2.3522,
      work_mode: ["remote"],
      travel_radius_km: null,
    })
  })

  it("clears the city selection when the country changes", async () => {
    const user = userEvent.setup()
    profileOverride = {
      city: "Paris",
      country_code: "FR",
      latitude: 48.8566,
      longitude: 2.3522,
    }
    renderSection()

    expect(screen.getByTestId("autocomplete-value").textContent).toBe("Paris")

    const countrySelect = screen.getByLabelText(
      messages.profile.location.countryLabel,
    )
    await user.selectOptions(countrySelect, "BE")
    expect(screen.getByTestId("autocomplete-value").textContent).toBe("")
  })
})
