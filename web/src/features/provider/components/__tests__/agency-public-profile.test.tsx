import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import messages from "@/../messages/en.json"
import { AgencyPublicProfile } from "../agency-public-profile"
import type { Profile } from "../../api/profile-api"

// Route the app's i18n navigation wrapper to a plain <a> so the
// listing links inside the profile card don't require a real
// Next.js router in the test environment.
vi.mock("@i18n/navigation", () => ({
  Link: ({ children, ...rest }: { children: ReactNode }) => (
    <a {...rest}>{children}</a>
  ),
  useRouter: () => ({ back: () => {}, push: () => {} }),
}))

// Stub out network-touching nested sections so the surface itself is
// asserted in isolation.
vi.mock("@/shared/hooks/profile/use-project-history", () => ({
  useProjectHistory: () => ({
    data: undefined,
    isLoading: false,
    isError: false,
  }),
}))
vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "user-1",
}))
vi.mock("@/features/provider/hooks/use-portfolio", () => ({
  usePortfolio: () => ({ data: [], isLoading: false }),
  usePublicPortfolio: () => ({ data: [], isLoading: false }),
  usePortfolioByOrganization: () => ({ data: [], isLoading: false }),
}))

function buildProfile(overrides: Partial<Profile> = {}): Profile {
  return {
    organization_id: "org-agency-1",
    title: "Boutique creative agency",
    photo_url: "",
    presentation_video_url: "",
    referrer_video_url: "",
    about: "We craft brand systems for ambitious teams.",
    referrer_about: "",
    expertise_domains: ["development", "design_ui_ux"],
    skills: [{ skill_text: "design", display_text: "Design" }],
    city: "Lyon",
    country_code: "FR",
    work_mode: ["remote"],
    travel_radius_km: null,
    languages_professional: ["fr", "en"],
    languages_conversational: [],
    availability_status: "available_now",
    pricing: [
      {
        kind: "direct",
        type: "project_from",
        min_amount: 1000000,
        max_amount: null,
        currency: "EUR",
        note: "",
        negotiable: false,
      },
    ],
    created_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

function renderProfile(profile: Profile) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <QueryClientProvider client={client}>
        <AgencyPublicProfile
          profile={profile}
          orgId={profile.organization_id}
          displayName="Studio Forge"
        />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

describe("AgencyPublicProfile — unified Soleil v2 shell", () => {
  it("renders the prestataire hero with display name and italic title", () => {
    renderProfile(buildProfile())
    expect(
      screen.getByRole("heading", { level: 1, name: "Studio Forge" }),
    ).toBeInTheDocument()
    expect(screen.getByText("Boutique creative agency")).toBeInTheDocument()
    expect(
      screen.getByTestId("availability-pill-available_now"),
    ).toBeInTheDocument()
  })

  it("renders the pricing rail headline derived from the direct pricing row", () => {
    renderProfile(buildProfile())
    // 1_000_000 cents = €10,000 — surfaced both on the hero rail and
    // on the dedicated PricingDisplayCard further down.
    expect(screen.getByText("Starting at")).toBeInTheDocument()
    expect(screen.getAllByText(/From €10,000/).length).toBeGreaterThan(0)
  })

  it("uses 'logo' wording on the agency portrait alt text", () => {
    renderProfile(buildProfile())
    expect(
      screen.getByRole("img", { name: /Logo of Studio Forge/i }),
    ).toBeInTheDocument()
  })

  it("renders the about section with agency-flavored copy", () => {
    renderProfile(buildProfile())
    expect(
      screen.getByRole("heading", { name: "About the agency" }),
    ).toBeInTheDocument()
    expect(
      screen.getByText("We craft brand systems for ambitious teams."),
    ).toBeInTheDocument()
  })

  it("renders expertise pills and skills card when populated", () => {
    renderProfile(buildProfile())
    expect(screen.getByText("Development")).toBeInTheDocument()
    expect(
      screen.getByRole("heading", { name: "Skills" }),
    ).toBeInTheDocument()
  })

  it("hides the pricing rail when the agency has no direct pricing", () => {
    renderProfile(buildProfile({ pricing: [] }))
    expect(screen.queryByText("Starting at")).not.toBeInTheDocument()
  })
})
