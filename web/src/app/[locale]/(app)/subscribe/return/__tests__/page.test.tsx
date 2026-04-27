import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactElement } from "react"
import SubscribeReturnPage from "../page"

const mockSearchParams = new Map<string, string>()
vi.mock("next/navigation", () => ({
  useSearchParams: () => ({
    get: (key: string) => mockSearchParams.get(key) ?? null,
  }),
}))

vi.mock("next/link", () => ({
  default: ({ children, ...rest }: { children: React.ReactNode } & Record<string, unknown>) => (
    <a {...rest}>{children}</a>
  ),
}))

const getMySubscriptionMock = vi.fn()
vi.mock("@/features/subscription/api/subscription-api", () => ({
  getMySubscription: () => getMySubscriptionMock(),
}))

function renderPage(): ReactElement {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return (
    <QueryClientProvider client={client}>
      <SubscribeReturnPage />
    </QueryClientProvider>
  )
}

const fakeSubscription = {
  id: "sub_1",
  plan: "freelance" as const,
  billing_cycle: "monthly" as const,
  status: "active" as const,
  current_period_start: "2026-04-01T00:00:00Z",
  current_period_end: "2026-05-01T00:00:00Z",
  cancel_at_period_end: false,
  started_at: "2026-04-01T00:00:00Z",
}

describe("SubscribeReturnPage", () => {
  beforeEach(() => {
    getMySubscriptionMock.mockReset()
    mockSearchParams.clear()
  })

  it("renders pending state at first paint", () => {
    getMySubscriptionMock.mockReturnValue(new Promise(() => {})) // never resolves
    render(renderPage())
    expect(screen.getByText(/Activation en cours/i)).toBeDefined()
  })

  it("flips to the success state once the first poll lands an active subscription", async () => {
    getMySubscriptionMock.mockResolvedValue(fakeSubscription)
    render(renderPage())
    await waitFor(() => {
      expect(screen.getByText(/Premium activé/i)).toBeDefined()
    })
  })

  it("hides the dashboard CTA when return_to=mobile is set", async () => {
    mockSearchParams.set("return_to", "mobile")
    getMySubscriptionMock.mockResolvedValue(fakeSubscription)
    render(renderPage())
    await waitFor(() => {
      expect(screen.getByText(/Premium activé/i)).toBeDefined()
    })
    expect(screen.queryByRole("link", { name: /tableau de bord/i })).toBeNull()
  })
})
