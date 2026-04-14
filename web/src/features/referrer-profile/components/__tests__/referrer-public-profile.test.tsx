import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import messages from "@/../messages/en.json"
import { ReferrerPublicProfile } from "../referrer-public-profile"
import type { ReferrerProfile } from "../../api/referrer-profile-api"

vi.mock("@i18n/navigation", () => ({
  Link: ({ children, ...rest }: { children: ReactNode }) => (
    <a {...rest}>{children}</a>
  ),
  useRouter: () => ({ back: () => {}, push: () => {} }),
}))

vi.mock("@/shared/hooks/profile/use-project-history", () => ({
  useProjectHistory: () => ({ data: undefined, isLoading: false, isError: false }),
}))
vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "user-1",
}))
vi.mock("../../hooks/use-referrer-pricing", () => ({
  useReferrerPricing: () => ({ data: null }),
}))

function buildProfile(
  overrides: Partial<ReferrerProfile> = {},
): ReferrerProfile {
  return {
    id: "profile-1",
    organization_id: "org-1",
    title: "Connector",
    about: "I connect enterprises with the right providers.",
    video_url: "",
    availability_status: "available_soon",
    expertise_domains: ["consulting_strategy"],
    photo_url: "",
    city: "Lyon",
    country_code: "FR",
    latitude: null,
    longitude: null,
    work_mode: ["remote"],
    travel_radius_km: null,
    languages_professional: ["fr"],
    languages_conversational: ["en"],
    pricing: null,
    created_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

function renderProfile(profile: ReferrerProfile) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <QueryClientProvider client={client}>
        <ReferrerPublicProfile profile={profile} displayName="Grace" />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

describe("ReferrerPublicProfile", () => {
  it("renders the identity header with the business referrer badge", () => {
    renderProfile(buildProfile())
    expect(
      screen.getByRole("heading", { level: 1, name: "Grace" }),
    ).toBeInTheDocument()
    expect(screen.getByText("Business Referrer")).toBeInTheDocument()
    expect(
      screen.getByTestId("availability-pill-available_soon"),
    ).toBeInTheDocument()
  })

  it("does NOT render a skills section", () => {
    renderProfile(buildProfile())
    // The freelance persona owns skills — the referrer view must
    // not carry a list element tagged with the freelance testid.
    expect(
      screen.queryByTestId("freelance-skills-list"),
    ).not.toBeInTheDocument()
  })

  it("renders the referrer about card with its persona label", () => {
    renderProfile(buildProfile())
    expect(
      screen.getByText("I connect enterprises with the right providers."),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("heading", { level: 2, name: /about the business referrer/i }),
    ).toBeInTheDocument()
  })

  it("surfaces the referrer-specific empty-history state", () => {
    renderProfile(buildProfile())
    expect(
      screen.getByText(/No referred deals yet/i),
    ).toBeInTheDocument()
  })
})
