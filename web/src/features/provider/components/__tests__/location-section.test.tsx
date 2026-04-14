import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
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

  it("renders city and country inputs with empty defaults", () => {
    renderSection()
    expect(screen.getByLabelText(messages.profile.location.cityLabel)).toHaveValue("")
    expect(screen.getByLabelText(messages.profile.location.countryLabel)).toHaveValue("")
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

  it("submits the form with sanitized values on save", async () => {
    const user = userEvent.setup()
    renderSection()

    const cityInput = screen.getByLabelText(messages.profile.location.cityLabel)
    await user.type(cityInput, "Paris")
    const countrySelect = screen.getByLabelText(
      messages.profile.location.countryLabel,
    )
    await user.selectOptions(countrySelect, "FR")
    await user.click(
      screen.getByRole("button", { name: messages.profile.location.workModeRemote }),
    )

    const saveBtn = screen.getByRole("button", {
      name: new RegExp(messages.profile.location.save, "i"),
    })
    await user.click(saveBtn)

    expect(mockMutate).toHaveBeenCalledWith({
      city: "Paris",
      country_code: "FR",
      work_mode: ["remote"],
      travel_radius_km: null,
    })
  })
})
