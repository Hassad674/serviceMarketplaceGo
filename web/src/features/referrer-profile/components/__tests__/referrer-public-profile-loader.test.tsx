import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import messages from "@/../messages/en.json"
import type { ReferrerProfile } from "../../api/referrer-profile-api"

// The loader fans the public referrer profile out into the header,
// reputation, and social-links sub-surfaces. These tests focus on the
// regression that surfaced in production where an empty `title` field
// caused the page header to render the raw organization UUID — both
// ugly and a privacy concern. Behaviour expected after the fix:
//
//   - title set    -> header reads exactly that title
//   - title empty  -> header falls back to the localized "Apporteur
//     d'affaires" / "Business referrer" label
//   - the raw UUID is NEVER rendered as the header text.

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

vi.mock("../../hooks/use-referrer-reputation", () => ({
  useReferrerReputation: () => ({
    data: {
      pages: [
        {
          rating_avg: 0,
          review_count: 0,
          history: [],
          next_cursor: "",
          has_more: false,
        },
      ],
    },
    isLoading: false,
    isError: false,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: () => {},
  }),
}))

const profileMock = vi.hoisted(() => ({
  current: null as ReferrerProfile | null,
  loading: false,
  error: null as Error | null,
}))

vi.mock("../../hooks/use-referrer-profile", () => ({
  usePublicReferrerProfile: () => ({
    data: profileMock.current,
    isLoading: profileMock.loading,
    error: profileMock.error,
  }),
}))

import { ReferrerPublicProfileLoader } from "../referrer-public-profile-loader"

function buildProfile(overrides: Partial<ReferrerProfile> = {}): ReferrerProfile {
  return {
    id: "profile-1",
    organization_id: "2d454cba-6949-4c08-95a1-e105c51ff368",
    title: "",
    about: "",
    video_url: "",
    availability_status: "available_now",
    expertise_domains: [],
    photo_url: "",
    city: "",
    country_code: "",
    latitude: null,
    longitude: null,
    work_mode: [],
    travel_radius_km: null,
    languages_professional: [],
    languages_conversational: [],
    pricing: null,
    created_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

function renderLoader() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <QueryClientProvider client={client}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <ReferrerPublicProfileLoader orgId="2d454cba-6949-4c08-95a1-e105c51ff368" />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

describe("ReferrerPublicProfileLoader — display name fallback", () => {
  it("renders the title verbatim when set", () => {
    profileMock.current = buildProfile({ title: "Top Connector" })
    profileMock.loading = false
    profileMock.error = null
    renderLoader()
    expect(
      screen.getByRole("heading", { level: 1, name: "Top Connector" }),
    ).toBeInTheDocument()
  })

  it("falls back to the localized referrer label when title is empty", () => {
    profileMock.current = buildProfile({ title: "" })
    profileMock.loading = false
    profileMock.error = null
    renderLoader()
    // The English fallback is "Business referrer". The exact wording
    // comes from messages.profile.referrer.displayNameFallback.
    expect(
      screen.getByRole("heading", {
        level: 1,
        name: messages.profile.referrer.displayNameFallback,
      }),
    ).toBeInTheDocument()
    // Hard guarantee: the raw UUID must NEVER reach the rendered DOM
    // as the header. The previous bug surfaced exactly this string on
    // the production /fr/referrers/{uuid} surface.
    expect(
      screen.queryByText("2d454cba-6949-4c08-95a1-e105c51ff368"),
    ).not.toBeInTheDocument()
  })
})
