/**
 * AppRouter tests — ADMIN-PERF-01.
 *
 * The router lazy-loads every authenticated page. We mock each
 * feature's page module with a fast-resolving stub so the test
 * exercises the Suspense boundary contract:
 *
 *   - the skeleton appears while the chunk is "loading"
 *   - the page content lands once the lazy import resolves
 *   - the route map matches the production paths
 *
 * MemoryRouter is not used because AppRouter ships its own
 * BrowserRouter — instead we navigate via window.history and rely
 * on react-router's reactivity.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

// Stub each lazy page so its dynamic import resolves quickly.
vi.mock("@/features/dashboard/components/dashboard-page", () => ({
  DashboardPage: () => <div data-testid="dashboard-page">DashboardPage</div>,
}))
vi.mock("@/features/users/components/users-page", () => ({
  UsersPage: () => <div data-testid="users-page">UsersPage</div>,
}))
vi.mock("@/features/users/components/user-detail-page", () => ({
  UserDetailPage: () => <div data-testid="user-detail-page">UserDetailPage</div>,
}))
vi.mock("@/features/moderation/components/moderation-page", () => ({
  ModerationPage: () => <div data-testid="moderation-page">ModerationPage</div>,
}))
vi.mock("@/features/conversations/components/conversations-page", () => ({
  ConversationsPage: () => <div data-testid="conversations-page" />,
}))
vi.mock("@/features/conversations/components/conversation-detail-page", () => ({
  ConversationDetailPage: () => <div data-testid="conversation-detail-page" />,
}))
vi.mock("@/features/jobs/components/jobs-page", () => ({
  JobsPage: () => <div data-testid="jobs-page" />,
}))
vi.mock("@/features/jobs/components/job-detail-page", () => ({
  JobDetailPage: () => <div data-testid="job-detail-page" />,
}))
vi.mock("@/features/reviews/components/reviews-page", () => ({
  ReviewsPage: () => <div data-testid="reviews-page" />,
}))
vi.mock("@/features/reviews/components/review-detail-page", () => ({
  ReviewDetailPage: () => <div data-testid="review-detail-page" />,
}))
vi.mock("@/features/media/components/media-page", () => ({
  MediaPage: () => <div data-testid="media-page" />,
}))
vi.mock("@/features/media/components/media-detail-page", () => ({
  MediaDetailPage: () => <div data-testid="media-detail-page" />,
}))
vi.mock("@/features/disputes/components/disputes-page", () => ({
  DisputesPage: () => <div data-testid="disputes-page" />,
}))
vi.mock("@/features/disputes/components/dispute-detail-page", () => ({
  DisputeDetailPage: () => <div data-testid="dispute-detail-page" />,
}))
vi.mock("@/features/invoices/components/invoices-page", () => ({
  InvoicesPage: () => <div data-testid="invoices-page" />,
}))

// AdminLayout wraps with sidebar + outlet — stub it so the test only
// exercises the route resolution.
vi.mock("@/shared/components/layouts/admin-layout", async () => {
  const { Outlet } = await import("react-router-dom")
  return {
    AdminLayout: () => <Outlet />,
  }
})

vi.mock("@/features/auth/components/login-form", () => ({
  LoginPage: () => <div data-testid="login-page">LoginPage</div>,
}))

import { AppRouter } from "../router"

function renderRouter(path: string) {
  window.history.replaceState({}, "", path)
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={qc}>
      <AppRouter />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  // Reset history between tests.
  window.history.replaceState({}, "", "/")
})

afterEach(() => {
  vi.clearAllMocks()
})

describe("AppRouter — ADMIN-PERF-01 lazy routes", () => {
  it("renders DashboardPage on /", async () => {
    renderRouter("/")
    expect(await screen.findByTestId("dashboard-page")).toBeInTheDocument()
  })

  it("renders LoginPage on /login (eager)", async () => {
    renderRouter("/login")
    expect(await screen.findByTestId("login-page")).toBeInTheDocument()
  })

  it("renders UsersPage on /users", async () => {
    renderRouter("/users")
    expect(await screen.findByTestId("users-page")).toBeInTheDocument()
  })

  it("renders UserDetailPage on /users/:id", async () => {
    renderRouter("/users/abc")
    expect(await screen.findByTestId("user-detail-page")).toBeInTheDocument()
  })

  it("renders ModerationPage on /moderation", async () => {
    renderRouter("/moderation")
    expect(await screen.findByTestId("moderation-page")).toBeInTheDocument()
  })

  it("renders JobsPage on /jobs", async () => {
    renderRouter("/jobs")
    expect(await screen.findByTestId("jobs-page")).toBeInTheDocument()
  })

  it("renders InvoicesPage on /invoices", async () => {
    renderRouter("/invoices")
    expect(await screen.findByTestId("invoices-page")).toBeInTheDocument()
  })

  it("renders DisputesPage on /disputes", async () => {
    renderRouter("/disputes")
    expect(await screen.findByTestId("disputes-page")).toBeInTheDocument()
  })

  it("renders ReviewsPage on /reviews", async () => {
    renderRouter("/reviews")
    expect(await screen.findByTestId("reviews-page")).toBeInTheDocument()
  })

  it("renders MediaPage on /media", async () => {
    renderRouter("/media")
    expect(await screen.findByTestId("media-page")).toBeInTheDocument()
  })

  it("renders ConversationsPage on /conversations", async () => {
    renderRouter("/conversations")
    expect(await screen.findByTestId("conversations-page")).toBeInTheDocument()
  })

  it("renders the route skeleton fallback while a chunk is suspended", async () => {
    // Override the dashboard mock with a never-resolving promise to
    // surface the Suspense fallback.
    let resolveLater: (v: unknown) => void = () => {}
    vi.doMock("@/features/dashboard/components/dashboard-page", () => {
      // Trigger an actual lazy promise that the test holds open.
      return new Promise((res) => {
        resolveLater = res as (v: unknown) => void
      })
    })

    // Re-import the router module so the new mock is picked up.
    vi.resetModules()
    const { AppRouter: Lazy } = await import("../router")
    window.history.replaceState({}, "", "/")
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={qc}>
        <Lazy />
      </QueryClientProvider>,
    )

    await waitFor(() => {
      expect(screen.getByRole("status")).toBeInTheDocument()
    })

    // Resolve the chunk so the test cleans up cleanly.
    resolveLater({
      DashboardPage: () => <div data-testid="dashboard-page">D</div>,
    })
  })
})
