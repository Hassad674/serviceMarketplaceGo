// Tests for (app)/layout.tsx — verifies the chromeless-path detection
// that lets /projects/pay opt out of the DashboardShell so its
// nested layout (PaymentCheckoutShell) is the sole chrome on screen.
//
// Regression target: the original layout wrapped every /(app) route
// in DashboardShell unconditionally, which rendered sidebar + top
// header behind the payment page's own editorial banner — the
// "deux navbar" UX bug.

import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"

// usePathname / useSearchParams are the only Next.js APIs the layout
// reads. Reassign per test so each scenario can pin the URL.
const pathnameMock = vi.fn<() => string>()
const searchParamsGetMock = vi.fn<(key: string) => string | null>(() => null)
vi.mock("next/navigation", () => ({
  usePathname: () => pathnameMock(),
  useSearchParams: () => ({ get: (key: string) => searchParamsGetMock(key) }),
}))

// DashboardShell is heavy (sidebar, header, LiveKit, KYC banner, etc.).
// Stub to a sentinel div so we can detect when the parent layout
// chose to wrap children in the dashboard chrome vs render them bare.
vi.mock("@/shared/components/layouts/dashboard-shell", () => ({
  DashboardShell: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dashboard-shell">{children}</div>
  ),
}))

async function renderLayout(): Promise<HTMLElement> {
  const mod = await import("../layout")
  // Default export is the layout component (next.js convention).
  const AppLayout = mod.default
  render(
    <AppLayout>
      <p data-testid="page-body">child</p>
    </AppLayout>,
  )
  return screen.getByTestId("page-body")
}

beforeEach(() => {
  pathnameMock.mockReset()
  searchParamsGetMock.mockReset()
  searchParamsGetMock.mockImplementation(() => null)
})

describe("(app)/layout — chromeless path detection", () => {
  it.each([
    ["/fr/dashboard", "FR dashboard"],
    ["/en/projects", "EN projects index"],
    ["/fr/profile", "FR profile"],
    ["/fr/projects/abc-123", "FR proposal detail (NOT /pay)"],
  ])("wraps %s in DashboardShell (%s)", async (path) => {
    pathnameMock.mockReturnValue(path)
    await renderLayout()
    expect(screen.getByTestId("dashboard-shell")).toBeInTheDocument()
    expect(screen.getByTestId("page-body")).toBeInTheDocument()
  })

  it.each([
    ["/fr/projects/pay", "FR checkout root"],
    ["/en/projects/pay", "EN checkout root"],
    ["/fr/projects/pay/", "trailing slash"],
    ["/fr/projects/pay?proposal=abc", "with query string is irrelevant (pathname only)"],
    ["/en/projects/pay/anything-after", "deeper segment under /pay"],
  ])("skips DashboardShell on %s (%s)", async (path) => {
    // usePathname returns the path WITHOUT the query string in Next.js.
    // We strip ?... here so the test reflects real runtime behavior.
    const cleanPath = path.split("?")[0]
    pathnameMock.mockReturnValue(cleanPath)
    await renderLayout()
    expect(
      screen.queryByTestId("dashboard-shell"),
    ).not.toBeInTheDocument()
    // The children must still render — we only skip the wrapper.
    expect(screen.getByTestId("page-body")).toBeInTheDocument()
  })

  it("still honours the legacy ?embedded=true short-circuit (kept for upgrade-modal flows)", async () => {
    pathnameMock.mockReturnValue("/fr/dashboard")
    searchParamsGetMock.mockImplementation((key: string) =>
      key === "embedded" ? "true" : null,
    )
    await renderLayout()
    expect(
      screen.queryByTestId("dashboard-shell"),
    ).not.toBeInTheDocument()
    expect(screen.getByTestId("page-body")).toBeInTheDocument()
  })

  it("does NOT match a path that contains 'projects/pay' as a substring of another segment", async () => {
    // Defensive: /fr/projects/payments-history (hypothetical future
    // route) must NOT be matched as the chromeless checkout flow.
    pathnameMock.mockReturnValue("/fr/projects/payments-history")
    await renderLayout()
    // Real dashboard chrome still applied.
    expect(screen.getByTestId("dashboard-shell")).toBeInTheDocument()
  })
})
