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
    const pressed = screen
      .getAllByRole("button")
      .find((b) => b.getAttribute("aria-pressed") === "true")
    expect(pressed?.textContent).toContain("30")
  })

  it("changing period synchronises the URL via router.replace", async () => {
    replaceMock.mockClear()
    render(wrap(<StatsOverview />))
    const ninetyDays = screen
      .getAllByRole("button")
      .find((b) => b.textContent?.includes("90"))
    expect(ninetyDays).toBeTruthy()
    await userEvent.click(ninetyDays as HTMLElement)
    await waitFor(() => expect(replaceMock).toHaveBeenCalled())
    const lastCall = replaceMock.mock.calls.at(-1)?.[0] as string
    expect(lastCall).toContain("period=90")
  })

  it("supports the 1-year period filter (D3)", async () => {
    // The 365-day window must be wired end-to-end: clicking "1 an"
    // updates the URL state, which the visibility hook uses as a
    // query key — the next fetch carries `days=365`.
    replaceMock.mockClear()
    render(wrap(<StatsOverview />))
    const oneYear = screen
      .getAllByRole("button")
      .find((b) => b.textContent?.includes("1 an"))
    expect(oneYear).toBeTruthy()
    await userEvent.click(oneYear as HTMLElement)
    await waitFor(() => expect(replaceMock).toHaveBeenCalled())
    const lastCall = replaceMock.mock.calls.at(-1)?.[0] as string
    expect(lastCall).toContain("period=365")
  })
})

// Bug 2 regression: the unit counts (total_views, search_appearances)
// must render as numbers even at zero. The "patience ~7 days" copy is
// reserved for the avg_search_position card.

function buildStatsFetch(payload: {
  total_views: number
  search_appearances: number
  avg_search_position: number | null
}) {
  return vi.fn(async (input: RequestInfo | URL) => {
    const url = input.toString()
    if (url.includes("/me/stats/visibility")) {
      return new Response(
        JSON.stringify({
          data: {
            organization_id: "org",
            period_days: 30,
            total_views: payload.total_views,
            unique_viewers: 0,
            search_appearances: payload.search_appearances,
            avg_search_position: payload.avg_search_position,
            series: [],
          },
        }),
        { status: 200 },
      )
    }
    if (url.includes("/me/stats/keywords")) {
      return new Response(JSON.stringify({ data: [] }), { status: 200 })
    }
    return new Response("{}", { status: 200 })
  })
}

describe("StatsOverview — unit counts (Bug 2 regression)", () => {
  it("renders '0' for total_views and search_appearances when both are 0", async () => {
    vi.stubGlobal(
      "fetch",
      buildStatsFetch({
        total_views: 0,
        search_appearances: 0,
        avg_search_position: null,
      }),
    )
    render(wrap(<StatsOverview />))
    const strip = await screen.findByTestId("stats-metric-strip")
    await waitFor(() => {
      // Unit-count cards render a literal "0" (not the patience copy).
      const zeros = Array.from(strip.querySelectorAll("p")).filter(
        (el) => el.textContent?.trim() === "0",
      )
      expect(zeros.length).toBeGreaterThanOrEqual(3)
    })
    // The legacy patience copy must NOT appear at the page level — only
    // the chart-level neutral empty copy is acceptable. Assert the old
    // banner string is gone.
    expect(
      screen.queryByText(
        /Données insuffisantes — patiente pendant que ton profil/i,
      ),
    ).toBeNull()
  })

  it("renders all metric cards with proper values when data is plentiful", async () => {
    vi.stubGlobal(
      "fetch",
      buildStatsFetch({
        total_views: 150,
        search_appearances: 75,
        avg_search_position: 3.5,
      }),
    )
    render(wrap(<StatsOverview />))
    const strip = await screen.findByTestId("stats-metric-strip")
    await waitFor(() => {
      // The integer formatter uses fr-FR ("150" without thousand
      // separator at this magnitude).
      expect(strip.textContent).toContain("150")
      expect(strip.textContent).toContain("75")
      // Rounded position rendered via plural unit ("4 places").
      expect(strip.textContent).toMatch(/4\s+places/i)
    })
  })

  it("renders the patience copy for avg_position when below significance threshold", async () => {
    vi.stubGlobal(
      "fetch",
      buildStatsFetch({
        total_views: 150,
        search_appearances: 3, // below 10
        avg_search_position: 2.0,
      }),
    )
    render(wrap(<StatsOverview />))
    const strip = await screen.findByTestId("stats-metric-strip")
    await waitFor(() => {
      // The avg_position card shows the patience caption.
      expect(strip.textContent).toMatch(/~7 jours pour des résultats stables/i)
    })
  })
})

// D3: empty state UX. When the org has zero recorded views, the chart
// pair is replaced by a friendly accentSoft card prompting a LinkedIn
// share. The strip remains visible so the user still sees the zeros.
describe("StatsOverview — D3 empty card", () => {
  it("renders the empty card when total_views is 0", async () => {
    vi.stubGlobal(
      "fetch",
      buildStatsFetch({
        total_views: 0,
        search_appearances: 0,
        avg_search_position: null,
      }),
    )
    render(wrap(<StatsOverview />))
    const empty = await screen.findByTestId("stats-empty-card")
    expect(empty).toBeInTheDocument()
    expect(empty.textContent).toMatch(/Personne n'a encore consulté/i)
    expect(empty.textContent).toMatch(/LinkedIn/i)
  })

  it("does NOT render the empty card when there are views", async () => {
    vi.stubGlobal(
      "fetch",
      buildStatsFetch({
        total_views: 12,
        search_appearances: 3,
        avg_search_position: null,
      }),
    )
    render(wrap(<StatsOverview />))
    await screen.findByTestId("stats-metric-strip")
    expect(screen.queryByTestId("stats-empty-card")).toBeNull()
  })
})

// D3: visibility series carries `unique` alongside `count`. Both
// counts must flow through to their respective metric cards.
describe("StatsOverview — D3 unique / total split", () => {
  it("shows unique viewers and total views as separate metric cards", async () => {
    const split = vi.fn(async (input: RequestInfo | URL) => {
      const url = input.toString()
      if (url.includes("/me/stats/visibility")) {
        return new Response(
          JSON.stringify({
            data: {
              organization_id: "org",
              period_days: 30,
              total_views: 200,
              unique_viewers: 50,
              search_appearances: 12,
              avg_search_position: 3.5,
              series: [
                { date: "2026-04-10T00:00:00Z", count: 6, unique: 4 },
                { date: "2026-04-11T00:00:00Z", count: 9, unique: 6 },
              ],
            },
          }),
          { status: 200 },
        )
      }
      if (url.includes("/me/stats/keywords")) {
        return new Response(JSON.stringify({ data: [] }), { status: 200 })
      }
      return new Response("{}", { status: 200 })
    })
    vi.stubGlobal("fetch", split)
    render(wrap(<StatsOverview />))
    const strip = await screen.findByTestId("stats-metric-strip")
    await waitFor(() => {
      expect(strip.textContent).toContain("50")
      expect(strip.textContent).toContain("200")
      expect(strip.textContent).toMatch(/Visiteurs uniques/i)
      expect(strip.textContent).toMatch(/visiteurs distincts/i)
    })
  })
})
