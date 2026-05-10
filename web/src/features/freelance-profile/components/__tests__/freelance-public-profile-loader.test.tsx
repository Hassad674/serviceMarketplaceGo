import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import messages from "@/../messages/en.json"
import { FreelancePublicProfileLoader } from "../freelance-public-profile-loader"
import type { FreelanceProfile } from "../../api/freelance-profile-api"

// Plain <a> shim for the i18n navigation Link so the test runtime
// does not need the Next.js router context.
vi.mock("@i18n/navigation", () => ({
  Link: ({ children, ...rest }: { children: ReactNode }) => (
    <a {...rest}>{children}</a>
  ),
  useRouter: () => ({ back: () => {}, push: () => {} }),
}))

// Short-circuit nested data hooks so the loader can be asserted in
// isolation. Each mock factory returns the same shape per test
// because we drive the heading via the `usePublicFreelanceProfile`
// override below.
vi.mock("@/shared/hooks/profile/use-project-history", () => ({
  useProjectHistory: () => ({ data: undefined, isLoading: false, isError: false }),
}))
vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "user-1",
}))
vi.mock("../../hooks/use-freelance-pricing", () => ({
  useFreelancePricing: () => ({ data: null }),
}))
vi.mock("@/shared/hooks/profile/use-profile-rating", () => ({
  useProfileRating: () => ({ data: null }),
}))
vi.mock("@/features/freelance-profile/components/freelance-social-links-section", () => ({
  PublicFreelanceSocialLinks: () => null,
}))
vi.mock("@/shared/components/profile/project-history-section", () => ({
  ProjectHistorySection: () => null,
}))

const profileHookState = {
  data: undefined as FreelanceProfile | undefined,
  isLoading: false,
  error: null as unknown,
}

vi.mock("../../hooks/use-freelance-profile", () => ({
  usePublicFreelanceProfile: () => profileHookState,
}))

function buildProfile(overrides: Partial<FreelanceProfile> = {}): FreelanceProfile {
  return {
    id: "profile-1",
    organization_id: "org-1",
    title: "Senior Go engineer",
    about: "I build distributed systems for a living.",
    video_url: "",
    availability_status: "available_now",
    expertise_domains: [],
    photo_url: "",
    city: "Paris",
    country_code: "FR",
    latitude: null,
    longitude: null,
    work_mode: ["remote"],
    travel_radius_km: null,
    languages_professional: ["fr", "en"],
    languages_conversational: [],
    skills: [],
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
    <NextIntlClientProvider locale="en" messages={messages}>
      <QueryClientProvider client={client}>
        <FreelancePublicProfileLoader orgId="org-1" />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

describe("FreelancePublicProfileLoader — heading and subtitle wiring", () => {
  it("renders ${first_name} ${last_name} as the H1 when both are present", () => {
    profileHookState.data = buildProfile({
      first_name: "Ada",
      last_name: "Lovelace",
      title: "Senior Go engineer",
    })
    renderLoader()
    expect(
      screen.getByRole("heading", { level: 1, name: "Ada Lovelace" }),
    ).toBeInTheDocument()
  })

  it("renders the persona title as italic subtitle when name and title differ", () => {
    profileHookState.data = buildProfile({
      first_name: "Ada",
      last_name: "Lovelace",
      title: "Senior Go engineer",
    })
    renderLoader()
    expect(
      screen.getByText("Senior Go engineer", { selector: "p" }),
    ).toBeInTheDocument()
  })

  it("falls back to the title when first_name + last_name are empty", () => {
    profileHookState.data = buildProfile({
      first_name: "",
      last_name: "",
      title: "Senior Go engineer",
    })
    renderLoader()
    expect(
      screen.getByRole("heading", { level: 1, name: "Senior Go engineer" }),
    ).toBeInTheDocument()
    // No duplicated subtitle when displayName fell back to title.
    expect(
      screen.queryByText("Senior Go engineer", { selector: "p" }),
    ).not.toBeInTheDocument()
  })

  it("falls back to the localised 'Freelancer' label when both name and title are empty", () => {
    profileHookState.data = buildProfile({
      first_name: "",
      last_name: "",
      title: "",
    })
    renderLoader()
    expect(
      screen.getByRole("heading", { level: 1, name: "Freelancer" }),
    ).toBeInTheDocument()
  })

  it("trims a single missing name part (only first_name set)", () => {
    profileHookState.data = buildProfile({
      first_name: "Ada",
      last_name: "",
      title: "Senior Go engineer",
    })
    renderLoader()
    expect(
      screen.getByRole("heading", { level: 1, name: "Ada" }),
    ).toBeInTheDocument()
  })
})

describe("FreelancePublicProfileLoader — video section visibility", () => {
  it("omits the empty presentation video card when no video is set on the public view", () => {
    profileHookState.data = buildProfile({ video_url: "" })
    const { container } = renderLoader()
    // The empty-state placeholder ("Add a video to present your activity")
    // must not surface to public viewers when the org has no video.
    expect(
      screen.queryByText("Add a video to present your activity"),
    ).not.toBeInTheDocument()
    expect(container.querySelector("video")).toBeNull()
  })

  it("renders the embedded <video> tag when a video URL is set", () => {
    profileHookState.data = buildProfile({
      video_url: "https://media.example.test/intro.mp4",
    })
    const { container } = renderLoader()
    const video = container.querySelector("video")
    expect(video).not.toBeNull()
    expect(video?.getAttribute("src")).toBe(
      "https://media.example.test/intro.mp4",
    )
  })
})
