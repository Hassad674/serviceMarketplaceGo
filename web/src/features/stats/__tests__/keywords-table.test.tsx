import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { KeywordsTable } from "../components/keywords-table"

function wrap(node: React.ReactNode) {
  return (
    <NextIntlClientProvider locale="fr" messages={messages}>
      {node}
    </NextIntlClientProvider>
  )
}

describe("KeywordsTable", () => {
  it("renders the empty state for an empty rows array", () => {
    render(wrap(<KeywordsTable rows={[]} />))
    expect(screen.getByText(/Aucun mot-clé/i)).toBeInTheDocument()
  })

  it("renders one row per keyword", () => {
    render(
      wrap(
        <KeywordsTable
          rows={[
            { keyword: "go developer", count: 5, avg_position: 2.5 },
            { keyword: "react", count: 3, avg_position: null },
          ]}
        />,
      ),
    )
    expect(screen.getByText("go developer")).toBeInTheDocument()
    expect(screen.getByText("react")).toBeInTheDocument()
    expect(screen.getByText("5")).toBeInTheDocument()
    expect(screen.getByText("2.5")).toBeInTheDocument()
    expect(screen.getByText("—")).toBeInTheDocument()
  })

  it("scales the volume bar relative to the table maximum", () => {
    render(
      wrap(
        <KeywordsTable
          rows={[
            { keyword: "a", count: 10, avg_position: 1 },
            { keyword: "b", count: 5, avg_position: 2 },
          ]}
        />,
      ),
    )
    const bars = screen.getAllByTestId("keyword-volume-bar") as HTMLElement[]
    expect(bars[0].style.width).toBe("100%")
    expect(bars[1].style.width).toBe("50%")
  })

  it("shows the loading skeleton when isLoading", () => {
    const { container } = render(wrap(<KeywordsTable rows={[]} isLoading />))
    expect(container.querySelectorAll(".animate-pulse").length).toBeGreaterThanOrEqual(2)
  })
})
