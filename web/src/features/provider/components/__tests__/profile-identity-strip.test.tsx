import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ProfileIdentityStrip } from "../profile-identity-strip"
import type { Profile } from "../../api/profile-api"

function buildProfile(overrides: Partial<Profile> = {}): Profile {
  return {
    organization_id: "org-1",
    title: "",
    photo_url: "",
    presentation_video_url: "",
    referrer_video_url: "",
    about: "",
    referrer_about: "",
    created_at: "2026-04-01",
    updated_at: "2026-04-01",
    ...overrides,
  }
}

function renderStrip(profile: Profile) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ProfileIdentityStrip profile={profile} />
    </NextIntlClientProvider>,
  )
}

describe("ProfileIdentityStrip", () => {
  it("renders nothing when no tier-1 fields are populated", () => {
    const { container } = renderStrip(buildProfile())
    expect(container).toBeEmptyDOMElement()
  })

  it("renders the availability badge when availability_status is set", () => {
    renderStrip(
      buildProfile({ availability_status: "available_now" }),
    )
    expect(
      screen.getByText(messages.profile.availability.statusAvailableNow),
    ).toBeInTheDocument()
  })

  it("renders both direct and referrer availability badges when both are set", () => {
    renderStrip(
      buildProfile({
        availability_status: "available_now",
        referrer_availability_status: "available_soon",
      }),
    )
    expect(
      screen.getByText(messages.profile.availability.statusAvailableNow),
    ).toBeInTheDocument()
    expect(
      screen.getByText(messages.profile.availability.statusAvailableSoon),
    ).toBeInTheDocument()
  })

  it("renders the location block with city and country label", () => {
    renderStrip(buildProfile({ city: "Paris", country_code: "FR" }))
    expect(screen.getByText(/Paris, France/)).toBeInTheDocument()
  })

  it("renders work mode badges", () => {
    renderStrip(
      buildProfile({
        city: "Paris",
        country_code: "FR",
        work_mode: ["remote", "hybrid"],
      }),
    )
    expect(
      screen.getByText(messages.profile.location.workModeRemote),
    ).toBeInTheDocument()
    expect(
      screen.getByText(messages.profile.location.workModeHybrid),
    ).toBeInTheDocument()
  })

  it("renders the pricing block with the formatted amount", () => {
    renderStrip(
      buildProfile({
        pricing: [
          {
            kind: "direct",
            type: "daily",
            min_amount: 50000,
            max_amount: null,
            currency: "EUR",
            note: "",
          },
        ],
      }),
    )
    expect(
      screen.getByText(messages.profile.pricing.kindDirect),
    ).toBeInTheDocument()
    expect(screen.getByText(/500/)).toBeInTheDocument()
  })

  it("renders up to 5 professional language chips with overflow", () => {
    renderStrip(
      buildProfile({
        languages_professional: ["fr", "en", "es", "de", "it", "pt", "nl"],
      }),
    )
    expect(screen.getByText("FR")).toBeInTheDocument()
    expect(screen.getByText("EN")).toBeInTheDocument()
    expect(screen.getByText("ES")).toBeInTheDocument()
    expect(screen.getByText("DE")).toBeInTheDocument()
    expect(screen.getByText("IT")).toBeInTheDocument()
    expect(screen.queryByText("PT")).not.toBeInTheDocument()
    expect(screen.getByText("+2")).toBeInTheDocument()
  })
})
