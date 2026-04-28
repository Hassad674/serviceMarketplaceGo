import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactElement } from "react"

import PaymentInfoV2Page from "../page"

// next/navigation
const mockSearchParams = new Map<string, string>()
vi.mock("next/navigation", () => ({
  useParams: () => ({ locale: "fr" }),
  useSearchParams: () => ({
    get: (key: string) => mockSearchParams.get(key) ?? null,
  }),
}))

// next-intl — return the translation key so tests can assert against it
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Stripe SDKs — fully stubbed; we never enter the wizard/dashboard flows
// in these tests, but the page imports must resolve without booting any
// real network requests in jsdom.
vi.mock("@stripe/connect-js", () => ({
  loadConnectAndInitialize: vi.fn(() => ({})),
}))

vi.mock("@stripe/react-connect-js", () => ({
  ConnectAccountManagement: () => null,
  ConnectAccountOnboarding: () => null,
  ConnectComponentsProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  ConnectNotificationBanner: () => null,
}))

// Permission hook — controlled per-test
const permissionStatusMock = vi.fn()
vi.mock("@/shared/hooks/use-permissions", () => ({
  usePermissionStatus: (perm: string) => permissionStatusMock(perm),
}))

function renderPage(): ReactElement {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return (
    <QueryClientProvider client={client}>
      <PaymentInfoV2Page />
    </QueryClientProvider>
  )
}

describe("PaymentInfoV2Page — permission gating", () => {
  beforeEach(() => {
    permissionStatusMock.mockReset()
    mockSearchParams.clear()
  })

  it("renders a loading skeleton while the permission check is pending and does NOT show the access-denied card", () => {
    permissionStatusMock.mockReturnValue({
      status: "loading",
      granted: false,
      isLoading: true,
      isError: false,
    })

    render(renderPage())

    // Loader sentinel is present
    expect(screen.getByTestId("permission-loading")).toBeDefined()

    // The "Accès restreint" card must NOT be rendered yet
    expect(screen.queryByText("restrictedTitle")).toBeNull()
    expect(screen.queryByText("noKycManage")).toBeNull()
  })

  it("renders the access-denied card only after the permission check has settled with denied", () => {
    permissionStatusMock.mockReturnValue({
      status: "denied",
      granted: false,
      isLoading: false,
      isError: false,
    })

    render(renderPage())

    expect(screen.getByText("restrictedTitle")).toBeDefined()
    expect(screen.getByText("noKycManage")).toBeDefined()

    // Loading sentinel must be gone
    expect(screen.queryByTestId("permission-loading")).toBeNull()
  })

  it("does not render the access-denied card when permission is granted", async () => {
    permissionStatusMock.mockReturnValue({
      status: "granted",
      granted: true,
      isLoading: false,
      isError: false,
    })

    render(renderPage())

    // Wait a tick for the initial useEffect to run; assertion is the
    // negative — restricted copy must never appear.
    await waitFor(() => {
      expect(screen.queryByText("restrictedTitle")).toBeNull()
    })
    expect(screen.queryByTestId("permission-loading")).toBeNull()
  })
})
