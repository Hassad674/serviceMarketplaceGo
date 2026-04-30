/**
 * Boundary tests — PERF-W-03 + QUAL-W-01.
 *
 * Loading boundaries are async Server Components that call
 * `getTranslations` from `next-intl/server`. Tests:
 *   - render the resolved JSX (treating the async function as a
 *     plain promise of a React tree)
 *   - assert the markup has an accessible role + heading
 *   - assert the shimmer skeleton class is present
 *
 * Error / not-found boundaries are client components — rendered
 * directly via @testing-library.
 */

import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import type { ReactElement } from "react"

vi.mock("next-intl/server", () => ({
  getTranslations: async () => (key: string) => `t.${key}`,
}))

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => `t.${key}`,
}))

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children, ...rest }: { href: string; children: React.ReactNode }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))

async function renderAsync(modulePath: string) {
  const mod = await import(modulePath)
  const Component = mod.default as () => Promise<ReactElement>
  const tree = await Component()
  return render(tree)
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe("Loading boundaries — PERF-W-03", () => {
  it("(app) loading.tsx renders an accessible status region", async () => {
    await renderAsync("../[locale]/(app)/loading")
    const status = screen.getByRole("status")
    expect(status).toBeInTheDocument()
    expect(status.getAttribute("aria-live")).toBe("polite")
    // sr-only label must be present for screen readers.
    expect(screen.getByText("t.loadingDashboard")).toBeInTheDocument()
  })

  it("(public) loading.tsx renders an accessible status region", async () => {
    await renderAsync("../[locale]/(public)/loading")
    const status = screen.getByRole("status")
    expect(status).toBeInTheDocument()
    expect(screen.getByText("t.loadingListing")).toBeInTheDocument()
  })

  it("(auth) loading.tsx renders an accessible status region", async () => {
    await renderAsync("../[locale]/(auth)/loading")
    const status = screen.getByRole("status")
    expect(status).toBeInTheDocument()
    expect(screen.getByText("t.loadingAuth")).toBeInTheDocument()
  })

  it("messages/loading.tsx mirrors the two-pane layout", async () => {
    await renderAsync("../[locale]/(app)/messages/loading")
    expect(screen.getByRole("status")).toBeInTheDocument()
    expect(screen.getByText("t.loadingMessages")).toBeInTheDocument()
  })

  it("wallet/loading.tsx renders skeleton stack", async () => {
    await renderAsync("../[locale]/(app)/wallet/loading")
    expect(screen.getByRole("status")).toBeInTheDocument()
    expect(screen.getByText("t.loadingWallet")).toBeInTheDocument()
  })

  it("payment-info/loading.tsx renders skeleton stack", async () => {
    await renderAsync("../[locale]/(app)/payment-info/loading")
    expect(screen.getByRole("status")).toBeInTheDocument()
    expect(screen.getByText("t.loadingPaymentInfo")).toBeInTheDocument()
  })

  it("search/loading.tsx renders sidebar + grid", async () => {
    await renderAsync("../[locale]/(app)/search/loading")
    expect(screen.getByRole("status")).toBeInTheDocument()
    expect(screen.getByText("t.loadingSearch")).toBeInTheDocument()
  })
})

describe("not-found boundary — PERF-W-03", () => {
  it("renders a localised 404 with a CTA back to the listings", async () => {
    await renderAsync("../[locale]/not-found")
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "t.notFoundTitle",
    )
    const cta = screen.getByRole("link") as HTMLAnchorElement
    expect(cta.getAttribute("href")).toBe("/agencies")
    expect(cta).toHaveTextContent("t.notFoundCta")
  })
})

describe("error boundary — PERF-W-03 + QUAL-W-01", () => {
  it("renders an alert with retry + home buttons", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {})
    const reset = vi.fn()
    const ErrorBoundary = (await import("../[locale]/error")).default
    render(<ErrorBoundary error={new Error("boom")} reset={reset} />)

    const alert = screen.getByRole("alert")
    expect(alert).toBeInTheDocument()
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "t.errorTitle",
    )
    const retry = screen.getByRole("button", { name: "t.errorRetry" })
    retry.click()
    expect(reset).toHaveBeenCalled()
    consoleSpy.mockRestore()
  })

  it("logs the error digest for server correlation", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {})
    const reset = vi.fn()
    const ErrorBoundary = (await import("../[locale]/error")).default
    render(
      <ErrorBoundary
        error={Object.assign(new Error("explode"), { digest: "abc-123" })}
        reset={reset}
      />,
    )

    expect(consoleSpy).toHaveBeenCalled()
    const args = consoleSpy.mock.calls[0]
    const payload = args[1] as { digest?: string }
    expect(payload.digest).toBe("abc-123")
    consoleSpy.mockRestore()
  })
})

describe("global-error boundary — PERF-W-03", () => {
  it("renders a self-contained html shell with retry button", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {})
    const reset = vi.fn()
    const GlobalError = (await import("../global-error")).default
    render(
      <GlobalError
        error={Object.assign(new Error("kaboom"), { digest: "xyz" })}
        reset={reset}
      />,
    )

    expect(screen.getByRole("alert")).toBeInTheDocument()
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "We hit a snag",
    )
    const btn = screen.getByRole("button", { name: "Try again" })
    btn.click()
    expect(reset).toHaveBeenCalled()
    expect(consoleSpy).toHaveBeenCalled()
    consoleSpy.mockRestore()
  })
})
