import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { AvailabilitySection } from "../availability-section"
import type { Profile } from "../../api/profile-api"

const baseProfile: Profile = {
  organization_id: "org-1",
  title: "",
  photo_url: "",
  presentation_video_url: "",
  referrer_video_url: "",
  about: "",
  referrer_about: "",
  availability_status: "available_now",
  referrer_availability_status: null,
  created_at: "2026-04-01",
  updated_at: "2026-04-01",
}

let profileOverride: Partial<Profile> = {}
const mockMutate = vi.fn()

vi.mock("../../hooks/use-profile", () => ({
  useProfile: () => ({ data: { ...baseProfile, ...profileOverride } }),
  profileQueryKey: () => ["user", "x", "profile"],
}))

vi.mock("../../hooks/use-update-availability", () => ({
  useUpdateAvailability: () => ({
    mutate: (value: unknown) => mockMutate(value),
    isPending: false,
    isSuccess: false,
  }),
}))

function renderSection(
  props: Partial<Parameters<typeof AvailabilitySection>[0]> = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const defaults: Parameters<typeof AvailabilitySection>[0] = {
    orgType: "provider_personal",
    referrerEnabled: false,
    variant: "direct",
    readOnly: false,
  }
  return render(
    <QueryClientProvider client={queryClient}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <AvailabilitySection {...defaults} {...props} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  profileOverride = {}
})

describe("AvailabilitySection", () => {
  it("renders nothing for enterprise orgs", () => {
    const { container } = renderSection({ orgType: "enterprise" })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders nothing in read-only mode", () => {
    const { container } = renderSection({ readOnly: true })
    expect(container).toBeEmptyDOMElement()
  })

  it("direct variant renders only the direct group", () => {
    renderSection({ variant: "direct", referrerEnabled: true })
    expect(
      screen.getByText(messages.profile.availability.directTitle),
    ).toBeInTheDocument()
    expect(
      screen.queryByText(messages.profile.availability.referrerTitle),
    ).not.toBeInTheDocument()
  })

  it("referrer variant renders only the referrer group", () => {
    renderSection({
      variant: "referrer",
      referrerEnabled: true,
      orgType: "provider_personal",
    })
    expect(
      screen.getByText(messages.profile.availability.referrerTitle),
    ).toBeInTheDocument()
    expect(
      screen.queryByText(messages.profile.availability.directTitle),
    ).not.toBeInTheDocument()
  })

  it("referrer variant self-hides when referrer is not enabled", () => {
    const { container } = renderSection({
      variant: "referrer",
      referrerEnabled: false,
    })
    expect(container).toBeEmptyDOMElement()
  })

  it("direct variant saves only the direct field", async () => {
    const user = userEvent.setup()
    renderSection({ variant: "direct" })

    const notAvailableBtn = screen.getByRole("radio", {
      name: messages.profile.availability.statusNotAvailable,
    })
    await user.click(notAvailableBtn)

    const saveBtn = screen.getByRole("button", {
      name: new RegExp(messages.profile.availability.save, "i"),
    })
    expect(saveBtn).not.toBeDisabled()
    await user.click(saveBtn)

    expect(mockMutate).toHaveBeenCalledWith({
      availability_status: "not_available",
    })
  })

  it("referrer variant saves only the referrer field", async () => {
    const user = userEvent.setup()
    renderSection({
      variant: "referrer",
      referrerEnabled: true,
      orgType: "provider_personal",
    })

    const notAvailableBtn = screen.getByRole("radio", {
      name: messages.profile.availability.statusNotAvailable,
    })
    await user.click(notAvailableBtn)

    const saveBtn = screen.getByRole("button", {
      name: new RegExp(messages.profile.availability.save, "i"),
    })
    expect(saveBtn).not.toBeDisabled()
    await user.click(saveBtn)

    expect(mockMutate).toHaveBeenCalledWith({
      referrer_availability_status: "not_available",
    })
  })
})
