import { describe, expect, it } from "vitest"
import { render } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { LineChart } from "../line-chart"

// LineChart tests (D3) — cover the empty state, single-line render,
// and the new two-line mode with the secondary (total) overlay.

function wrap(node: React.ReactNode) {
  return (
    <NextIntlClientProvider locale="fr" messages={messages}>
      {node}
    </NextIntlClientProvider>
  )
}

const SERIES_PRIMARY = [
  { date: "2026-04-01T00:00:00Z", count: 2 },
  { date: "2026-04-02T00:00:00Z", count: 5 },
  { date: "2026-04-03T00:00:00Z", count: 3 },
]
const SERIES_SECONDARY = [
  { date: "2026-04-01T00:00:00Z", count: 4 },
  { date: "2026-04-02T00:00:00Z", count: 9 },
  { date: "2026-04-03T00:00:00Z", count: 7 },
]

describe("LineChart", () => {
  it("renders the empty card when series is empty", () => {
    const { getByTestId } = render(
      wrap(<LineChart series={[]} title="Vues" emptyMessage="rien" />),
    )
    expect(getByTestId("line-chart-empty")).toBeTruthy()
  })

  it("renders the empty card when every point is zero (no activity)", () => {
    const { getByTestId } = render(
      wrap(
        <LineChart
          series={[
            { date: "2026-04-01T00:00:00Z", count: 0 },
            { date: "2026-04-02T00:00:00Z", count: 0 },
          ]}
          title="Vues"
          emptyMessage="rien"
        />,
      ),
    )
    expect(getByTestId("line-chart-empty")).toBeTruthy()
  })

  it("renders a single primary polyline when no secondary series given", () => {
    const { getByTestId, queryByTestId } = render(
      wrap(<LineChart series={SERIES_PRIMARY} title="Vues" />),
    )
    expect(getByTestId("line-chart-primary")).toBeTruthy()
    expect(queryByTestId("line-chart-secondary")).toBeNull()
  })

  it("renders both primary AND secondary polylines when secondarySeries provided", () => {
    const { getByTestId } = render(
      wrap(
        <LineChart
          series={SERIES_PRIMARY}
          secondarySeries={SERIES_SECONDARY}
          title="Vues"
          primaryLabel="Uniques"
          secondaryLabel="Total"
        />,
      ),
    )
    expect(getByTestId("line-chart-primary")).toBeTruthy()
    const secondary = getByTestId("line-chart-secondary")
    expect(secondary).toBeTruthy()
    expect(secondary.getAttribute("stroke-dasharray")).toBe("4 4")
  })

  it("renders the legend when both labels are provided", () => {
    const { getByText } = render(
      wrap(
        <LineChart
          series={SERIES_PRIMARY}
          secondarySeries={SERIES_SECONDARY}
          title="Vues"
          primaryLabel="Uniques"
          secondaryLabel="Total"
        />,
      ),
    )
    expect(getByText("Uniques")).toBeTruthy()
    expect(getByText("Total")).toBeTruthy()
  })

  it("scales Y axis to the combined max of both series", () => {
    // The total series peaks at 9 while uniques peak at 5; the primary
    // polyline must therefore be drawn LOWER (higher y values) than it
    // would be with span=5 alone. Compare the primary path's minY
    // (the highest visible point) with and without the secondary
    // series — adding the secondary must push the primary downward.
    const minY = (path: string): number => {
      const ys: number[] = []
      const re = /[ML]\s*([\-\d.]+),([\-\d.]+)/g
      let match: RegExpExecArray | null
      while ((match = re.exec(path)) !== null) {
        ys.push(Number(match[2]))
      }
      return Math.min(...ys)
    }
    const withSecRender = render(
      wrap(
        <LineChart
          series={SERIES_PRIMARY}
          secondarySeries={SERIES_SECONDARY}
          title="Vues"
        />,
      ),
    )
    const withSec = minY(
      withSecRender.getByTestId("line-chart-primary").getAttribute("d") ?? "",
    )
    withSecRender.unmount()

    const withoutSecRender = render(
      wrap(<LineChart series={SERIES_PRIMARY} title="Vues" />),
    )
    const withoutSec = minY(
      withoutSecRender.getByTestId("line-chart-primary").getAttribute("d") ?? "",
    )
    withoutSecRender.unmount()

    // Adding a higher-magnitude secondary series shrinks the relative
    // height of the primary line — its highest point drops (= higher
    // y coordinate in SVG space).
    expect(withSec).toBeGreaterThan(withoutSec)
  })
})
