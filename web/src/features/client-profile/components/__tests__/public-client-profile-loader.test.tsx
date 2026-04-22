import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import type { PublicClientProfile } from "../../api/client-profile-api"
import type { ProjectHistoryResponse } from "@/shared/hooks/profile/use-project-history"

// Public loader exercises the unified "project history" card — one
// entry per completed mission with the provider→client review
// embedded inline, or an "awaiting review" placeholder otherwise.
// We mock the shared project-history hook and the public client
// profile hook so the section renders from in-memory fixtures.

const publicClientProfile: PublicClientProfile = {
  organization_id: "org-1",
  type: "enterprise",
  company_name: "Acme Corp",
  avatar_url: null,
  client_description: "We run ads at scale.",
  total_spent: 0,
  review_count: 1,
  average_rating: 5,
  projects_completed_as_client: 2,
}

const projectHistory: ProjectHistoryResponse = {
  data: [
    {
      proposal_id: "prop-1",
      title: "Landing page redesign",
      amount: 500000,
      currency: "EUR",
      completed_at: "2026-03-01T10:00:00Z",
      review: {
        id: "rev-1",
        proposal_id: "prop-1",
        reviewer_id: "user-provider-1",
        reviewed_id: "user-client-1",
        global_rating: 5,
        timeliness: null,
        communication: null,
        quality: null,
        comment: "Great client, clear brief and fast feedback.",
        video_url: null,
        title_visible: true,
        side: "provider_to_client",
        published_at: "2026-03-05T10:00:00Z",
        created_at: "2026-03-05T10:00:00Z",
      },
    },
    {
      proposal_id: "prop-2",
      title: "Growth audit",
      amount: 200000,
      currency: "EUR",
      completed_at: "2026-02-01T10:00:00Z",
      review: null,
    },
  ],
  next_cursor: "",
  has_more: false,
}

vi.mock("../../hooks/use-public-client-profile", () => ({
  usePublicClientProfile: () => ({
    data: publicClientProfile,
    isLoading: false,
    isError: false,
  }),
}))

vi.mock("@/shared/hooks/profile/use-project-history", async () => {
  const actual = await vi.importActual<
    typeof import("@/shared/hooks/profile/use-project-history")
  >("@/shared/hooks/profile/use-project-history")
  return {
    ...actual,
    useProjectHistory: () => ({
      data: projectHistory,
      isLoading: false,
      isError: false,
    }),
  }
})

import { PublicClientProfileLoader } from "../public-client-profile-loader"

function renderLoader() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <QueryClientProvider client={client}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <PublicClientProfileLoader orgId="org-1" />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

describe("PublicClientProfileLoader — unified project history", () => {
  it("renders one card per completed mission", () => {
    renderLoader()
    expect(screen.getByText("Landing page redesign")).toBeInTheDocument()
    expect(screen.getByText("Growth audit")).toBeInTheDocument()
  })

  it("embeds the provider→client review when present", () => {
    renderLoader()
    expect(
      screen.getByText("Great client, clear brief and fast feedback."),
    ).toBeInTheDocument()
  })

  it("falls back to the awaiting-review placeholder when missing", () => {
    renderLoader()
    // Profile namespace key shared with the provider surface.
    expect(screen.getByText(messages.profile.awaitingReview)).toBeInTheDocument()
  })

  it("does not render a standalone reviews section anymore", () => {
    renderLoader()
    // Old "Reviews from providers" block is gone; the review is now
    // embedded inside the corresponding project card.
    expect(
      screen.queryByRole("heading", { name: messages.clientProfile.reviewsTitle }),
    ).not.toBeInTheDocument()
  })
})
