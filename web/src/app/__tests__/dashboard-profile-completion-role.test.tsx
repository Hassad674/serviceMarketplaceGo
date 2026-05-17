import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/fr.json"
import DashboardPage from "../[locale]/(app)/dashboard/page"

// Regression coverage for UI-POLISH item 1:
//   The "Actions à faire" profile-completion nudge ("Ton profil est
//   complété à X% …") must NEVER render for an Enterprise (client)
//   account — a client has no public profile to complete. It must
//   still render for freelance / agency. When the Enterprise card has
//   no other action, the whole card is hidden rather than showing an
//   empty "Tout est à jour" surface.

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
  useRouter: () => ({ replace: vi.fn(), push: vi.fn(), refresh: vi.fn() }),
  usePathname: () => "/dashboard",
}))

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: vi.fn(), push: vi.fn(), refresh: vi.fn() }),
  useSearchParams: () => new URLSearchParams(),
  usePathname: () => "/dashboard",
}))

vi.mock("@/shared/hooks/use-workspace", () => ({
  useWorkspace: () => ({
    isReferrerMode: false,
    switchToReferrer: vi.fn(),
    switchToFreelance: vi.fn(),
  }),
}))

// Role is swapped per-test through this mutable holder.
const userState: { role: "provider" | "enterprise" | "agency" } = {
  role: "provider",
}

vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({
    data: {
      id: "u1",
      role: userState.role,
      first_name: "Sam",
      display_name: "Sam Org",
      kyc_status: "none",
    },
    isLoading: false,
  }),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "u1",
}))

// Profile is far from complete (12%) → the nudge WOULD show for a
// non-enterprise role. This is the exact condition item 1 must gate.
vi.mock("@/features/profile-completion/hooks/use-profile-completion", () => ({
  useProfileCompletion: () => ({ data: { percent: 12 }, isLoading: false }),
}))

function buildFetch() {
  return vi.fn(async (input: RequestInfo | URL) => {
    const url = input.toString()
    if (url.includes("/api/v1/messaging/unread-count")) {
      return new Response(JSON.stringify({ count: 0 }), { status: 200 })
    }
    if (url.includes("/me/stats/")) {
      return new Response(
        JSON.stringify({
          data: {
            organization_id: "o1",
            period_days: 7,
            total_views: 0,
            unique_viewers: 0,
            search_appearances: 0,
            avg_search_position: null,
            total_count: 0,
            series: [],
          },
        }),
        { status: 200 },
      )
    }
    return new Response("{}", { status: 200 })
  })
}

beforeEach(() => {
  vi.stubGlobal("fetch", buildFetch())
})

afterEach(() => {
  vi.unstubAllGlobals()
  userState.role = "provider"
})

function wrap(node: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 0 } },
  })
  return (
    <QueryClientProvider client={client}>
      <NextIntlClientProvider locale="fr" messages={messages}>
        {node}
      </NextIntlClientProvider>
    </QueryClientProvider>
  )
}

describe("Dashboard profile-completion nudge — role gating (item 1)", () => {
  it("freelance (provider): shows the profile-completion nudge", async () => {
    userState.role = "provider"
    render(wrap(<DashboardPage />))
    await waitFor(() =>
      expect(
        screen.getByText(/Ton profil est complété à 12\s*%/i),
      ).toBeInTheDocument(),
    )
  })

  it("agency: shows the profile-completion nudge", async () => {
    userState.role = "agency"
    render(wrap(<DashboardPage />))
    await waitFor(() =>
      expect(
        screen.getByText(/Ton profil est complété à 12\s*%/i),
      ).toBeInTheDocument(),
    )
  })

  it("enterprise: never shows the profile-completion nudge", async () => {
    userState.role = "enterprise"
    render(wrap(<DashboardPage />))
    // Page settles — the recruitments section card always renders for
    // enterprise, so we can wait on a stable enterprise-only surface.
    await waitFor(() =>
      expect(
        screen.getByRole("heading", { level: 2, name: /recrutements/i }),
      ).toBeInTheDocument(),
    )
    expect(screen.queryByText(/complété à/i)).toBeNull()
  })

  it("enterprise: hides the whole Actions card when it has no items", async () => {
    userState.role = "enterprise"
    render(wrap(<DashboardPage />))
    // Wait until every action query settles — the loading skeleton
    // keeps the card mounted, so we assert on the post-settle state.
    await waitFor(() =>
      expect(
        screen.queryByRole("heading", { name: /Actions à faire/i }),
      ).toBeNull(),
    )
    // Sanity: the rest of the enterprise dashboard still rendered.
    expect(
      screen.getByRole("heading", { level: 2, name: /recrutements/i }),
    ).toBeInTheDocument()
  })
})
