import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/fr.json"
import DashboardPage from "../[locale]/(app)/dashboard/page"

// Regression coverage for FIX-DASH:
//   * Bug 1: the "messages non lus" row must reflect the messaging
//     unread-count query (`/api/v1/messaging/unread-count`), NOT the
//     notification unread-count. When messaging unread drops to 0
//     (after read), the action row disappears on the next query
//     refresh.
//   * Bug 3: the "billing_info_missing" action row must NEVER be
//     rendered — billing info is collected at withdrawal time, only
//     KYC blocks payouts.

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
  useRouter: () => ({
    replace: vi.fn(),
    push: vi.fn(),
    refresh: vi.fn(),
  }),
  usePathname: () => "/dashboard",
}))

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    replace: vi.fn(),
    push: vi.fn(),
    refresh: vi.fn(),
  }),
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

const userMock = {
  id: "u1",
  role: "provider" as const,
  first_name: "Sam",
  display_name: "Sam Provider",
  kyc_status: "none",
}

vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({ data: userMock, isLoading: false }),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "u1",
}))

vi.mock("@/features/profile-completion/hooks/use-profile-completion", () => ({
  useProfileCompletion: () => ({
    data: { percent: 95 },
    isLoading: false,
  }),
}))

// Capture the URL of every fetch so we can assert the dashboard does
// NOT call /api/v1/notifications/unread-count for the "messages"
// action row anymore — only /api/v1/messaging/unread-count.
const fetchedUrls: string[] = []

function buildFetch(messagingUnreadCount: number) {
  return vi.fn(async (input: RequestInfo | URL) => {
    const url = input.toString()
    fetchedUrls.push(url)
    if (url.includes("/api/v1/messaging/unread-count")) {
      return new Response(JSON.stringify({ count: messagingUnreadCount }), {
        status: 200,
      })
    }
    // Visibility / applications stats — return empty payload.
    if (url.includes("/me/stats/visibility")) {
      return new Response(
        JSON.stringify({
          data: {
            organization_id: "o1",
            period_days: 7,
            total_views: 0,
            unique_viewers: 0,
            search_appearances: 0,
            avg_search_position: null,
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
  fetchedUrls.length = 0
})

afterEach(() => {
  vi.unstubAllGlobals()
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

describe("DashboardPage / actions-todo aggregator", () => {
  it("never renders a billing_info row (Bug 3 regression)", async () => {
    vi.stubGlobal("fetch", buildFetch(0))
    render(wrap(<DashboardPage />))
    // Wait for the page to settle.
    await waitFor(() =>
      expect(screen.getByRole("heading", { level: 2, name: /Actions/i })).toBeInTheDocument(),
    )
    // The billing copy ("informations de facturation") must not appear.
    expect(screen.queryByText(/informations de facturation/i)).toBeNull()
    expect(screen.queryByText(/billing/i)).toBeNull()
  })

  it("renders the unread messages row from the messaging unread-count endpoint (Bug 1)", async () => {
    vi.stubGlobal("fetch", buildFetch(3))
    render(wrap(<DashboardPage />))
    await waitFor(() =>
      expect(
        screen.getByText(/3 messages non lus/i),
      ).toBeInTheDocument(),
    )
    // Critical assertion: dashboard now reads the MESSAGING endpoint,
    // never the notifications endpoint. (notifications-unread is a
    // different concept — was wired by mistake before this fix.)
    expect(
      fetchedUrls.some((u) => u.includes("/api/v1/messaging/unread-count")),
    ).toBe(true)
    expect(
      fetchedUrls.some((u) =>
        u.includes("/api/v1/notifications/unread-count"),
      ),
    ).toBe(false)
  })

  it("hides the unread messages row when count is 0 (Bug 1 — read flow drops the action)", async () => {
    vi.stubGlobal("fetch", buildFetch(0))
    render(wrap(<DashboardPage />))
    await waitFor(() =>
      expect(screen.getByRole("heading", { level: 2, name: /Actions/i })).toBeInTheDocument(),
    )
    expect(screen.queryByText(/messages non lus/i)).toBeNull()
    // Empty state (no actions) message shows the "Tout est à jour" copy.
    expect(screen.getAllByText(/Tout est à jour/i).length).toBeGreaterThan(0)
  })
})
