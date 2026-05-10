import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/fr.json"
import { StatsOverview } from "../components/stats-overview"

const replaceMock = vi.fn()
vi.mock("next/navigation", () => ({
  useRouter: () => ({
    replace: replaceMock,
    push: vi.fn(),
  }),
  useSearchParams: () => new URLSearchParams("period=30"),
}))

vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({ data: { id: "u1" }, isLoading: false }),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "u1",
}))

const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
  const url = input.toString()
  if (url.includes("/me/stats/visibility")) {
    return new Response(
      JSON.stringify({
        data: {
          organization_id: "org",
          period_days: 30,
          total_views: 25,
          unique_viewers: 18,
          search_appearances: 12,
          avg_search_position: 3.2,
          series: [
            { date: "2026-04-10T00:00:00Z", count: 1 },
            { date: "2026-04-11T00:00:00Z", count: 2 },
            { date: "2026-04-12T00:00:00Z", count: 3 },
          ],
        },
      }),
      { status: 200 },
    )
  }
  if (url.includes("/me/stats/keywords")) {
    return new Response(
      JSON.stringify({
        data: [
          { keyword: "react developer", count: 5, avg_position: 2.5 },
          { keyword: "go", count: 2, avg_position: null },
        ],
      }),
      { status: 200 },
    )
  }
  return new Response("{}", { status: 200 })
})

beforeEach(() => {
  vi.stubGlobal("fetch", fetchMock)
  fetchMock.mockClear()
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

describe("StatsOverview integration", () => {
  it("loads visibility + keywords on mount and renders each section", async () => {
    render(wrap(<StatsOverview />))
    await waitFor(() =>
      expect(screen.getByText("react developer")).toBeInTheDocument(),
    )
    expect(screen.getByText("go")).toBeInTheDocument()
    // Period selector defaults to 30d, picked up from URL
    const buttons = screen.getAllByRole("button")
    const pressed = buttons.find((b) => b.getAttribute("aria-pressed") === "true")
    expect(pressed?.textContent).toContain("30")
  })

  it("changing period synchronises the URL via router.replace", async () => {
    replaceMock.mockClear()
    render(wrap(<StatsOverview />))
    const buttons = screen.getAllByRole("button")
    const ninetyDays = buttons.find((b) => b.textContent?.includes("90"))
    expect(ninetyDays).toBeTruthy()
    await userEvent.click(ninetyDays as HTMLElement)
    await waitFor(() => expect(replaceMock).toHaveBeenCalled())
    const lastCall = replaceMock.mock.calls.at(-1)?.[0] as string
    expect(lastCall).toContain("period=90")
  })
})
