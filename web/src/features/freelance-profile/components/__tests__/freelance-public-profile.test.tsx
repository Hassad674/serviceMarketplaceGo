import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import messages from "@/../messages/en.json"
import { FreelancePublicProfile } from "../freelance-public-profile"
import type { FreelanceProfile } from "../../api/freelance-profile-api"

// Route the app's i18n navigation wrapper to a plain <a> so the
// listing links inside the profile card don't require a real
// Next.js router in the test environment.
vi.mock("@i18n/navigation", () => ({
  Link: ({ children, ...rest }: { children: ReactNode }) => (
    <a {...rest}>{children}</a>
  ),
  useRouter: () => ({ back: () => {}, push: () => {} }),
}))

// Short-circuit hooks that hit the network in sub-components so the
// profile shell itself can be asserted in isolation.
vi.mock("@/shared/hooks/profile/use-project-history", () => ({
  useProjectHistory: () => ({ data: undefined, isLoading: false, isError: false }),
}))
vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "user-1",
}))
vi.mock("../../hooks/use-freelance-pricing", () => ({
  useFreelancePricing: () => ({ data: null }),
}))

function buildProfile(overrides: Partial<FreelanceProfile> = {}): FreelanceProfile {
  return {
    id: "profile-1",
    organization_id: "org-1",
    title: "Senior Go engineer",
    about: "I build distributed systems for a living.",
    video_url: "",
    availability_status: "available_now",
    expertise_domains: ["development", "consulting_strategy"],
    photo_url: "",
    city: "Paris",
    country_code: "FR",
    latitude: null,
    longitude: null,
    work_mode: ["remote"],
    travel_radius_km: null,
    languages_professional: ["fr", "en"],
    languages_conversational: [],
    skills: [{ skill_text: "go", display_text: "Go" }],
    pricing: null,
    created_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

function renderProfile(profile: FreelanceProfile) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <QueryClientProvider client={client}>
        <FreelancePublicProfile profile={profile} displayName="Ada" />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

describe("FreelancePublicProfile", () => {
  it("renders the identity header with display name, title and availability", () => {
    renderProfile(buildProfile())
    expect(
      screen.getByRole("heading", { level: 1, name: "Ada" }),
    ).toBeInTheDocument()
    expect(screen.getByText("Senior Go engineer")).toBeInTheDocument()
    expect(
      screen.getByTestId("availability-pill-available_now"),
    ).toBeInTheDocument()
  })

  it("renders the about section with the profile content", () => {
    renderProfile(buildProfile())
    expect(
      screen.getByText("I build distributed systems for a living."),
    ).toBeInTheDocument()
  })

  it("renders the skills strip when skills are present", () => {
    renderProfile(buildProfile())
    expect(screen.getByTestId("freelance-skills-list")).toBeInTheDocument()
    expect(screen.getByText("Go")).toBeInTheDocument()
  })

  it("hides the skills strip when skills are empty", () => {
    renderProfile(buildProfile({ skills: [] }))
    expect(
      screen.queryByTestId("freelance-skills-list"),
    ).not.toBeInTheDocument()
  })

  it("renders expertise pills for valid domain keys", () => {
    renderProfile(buildProfile())
    expect(screen.getByText("Development")).toBeInTheDocument()
  })
})
