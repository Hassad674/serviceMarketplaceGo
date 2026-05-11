/**
 * SessionsList — bulk revoke "others" button.
 *
 * Covers the gap left by sessions-list.test.tsx: when there is at
 * least one non-current session, the "Tout révoquer sauf cette
 * session" button is visible, and clicking it (after window.confirm)
 * fires `revokeOtherSessions` and drops every non-current row from
 * the cache. When every row is the current session, the button is
 * hidden — there is nothing else to revoke.
 *
 * Regression: the bulk-revoke action MUST be the single way to drop
 * other devices from the session list — the per-row "Révoquer" is
 * hidden on the current row by design. If the bulk button regresses,
 * users cannot log out other tabs/devices from the security panel.
 */
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { NextIntlClientProvider } from "next-intl"
import { SessionsList } from "../sessions-list"
import * as sessionsApi from "../../api/sessions-api"

const messages = {
  account: {
    security: {
      retry: "Réessayer",
      sessions: {
        title: "Sessions actives",
        subtitle: "Voici les appareils.",
        currentBadge: "Cette session",
        revoke: "Révoquer",
        revoking: "Révocation...",
        revokeOthers: "Tout révoquer sauf cette session",
        revokeOthersConfirm: "Confirmer ?",
        empty: "Aucune session active à afficher.",
        error: "Erreur",
        unknownLocation: "Localisation inconnue",
      },
    },
  },
}

function renderWith(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <NextIntlClientProvider locale="fr" messages={messages as never}>
        {ui}
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

function buildSession(overrides: Partial<sessionsApi.Session> = {}): sessionsApi.Session {
  return {
    id: "id-1",
    device_label: "Mac (Chrome)",
    browser: "Chrome",
    os: "macOS",
    city: "Paris",
    country_code: "FR",
    ip_anonymized: "203.0.113.0/24",
    login_method: "password",
    created_at: "2026-05-11T08:00:00Z",
    last_used_at: "2026-05-11T08:00:00Z",
    expires_at: "2026-05-25T08:00:00Z",
    is_current: false,
    ...overrides,
  }
}

describe("SessionsList — bulk revoke (Tout révoquer sauf cette session)", () => {
  let confirmSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    vi.restoreAllMocks()
    confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    confirmSpy.mockRestore()
  })

  it("renders the bulk button when at least one non-current session exists", async () => {
    vi.spyOn(sessionsApi, "listSessions").mockResolvedValue({
      data: [
        buildSession({ id: "a", is_current: true }),
        buildSession({ id: "b", is_current: false }),
      ],
    })
    renderWith(<SessionsList />)
    await screen.findByRole("button", { name: "Tout révoquer sauf cette session" })
  })

  it("hides the bulk button when every row is the current session", async () => {
    vi.spyOn(sessionsApi, "listSessions").mockResolvedValue({
      data: [buildSession({ id: "a", is_current: true })],
    })
    renderWith(<SessionsList />)
    // Wait for at least one row to render so the absence assertion is
    // not a false negative against an empty initial loader.
    await screen.findByText("Mac (Chrome)")
    expect(
      screen.queryByRole("button", {
        name: "Tout révoquer sauf cette session",
      }),
    ).toBeNull()
  })

  it("hides the bulk button when the session list is empty", async () => {
    vi.spyOn(sessionsApi, "listSessions").mockResolvedValue({ data: [] })
    renderWith(<SessionsList />)
    await screen.findByText("Aucune session active à afficher.")
    expect(
      screen.queryByRole("button", {
        name: "Tout révoquer sauf cette session",
      }),
    ).toBeNull()
  })

  it("calls revokeOtherSessions on confirm and refetches the list", async () => {
    const listSpy = vi
      .spyOn(sessionsApi, "listSessions")
      .mockResolvedValueOnce({
        data: [
          buildSession({ id: "current", is_current: true }),
          buildSession({ id: "other-1", is_current: false }),
          buildSession({ id: "other-2", is_current: false }),
        ],
      })
      // After bulk revoke, the cache invalidation refetches — only the
      // current row remains.
      .mockResolvedValueOnce({
        data: [buildSession({ id: "current", is_current: true })],
      })
    const bulkSpy = vi
      .spyOn(sessionsApi, "revokeOtherSessions")
      .mockResolvedValue(undefined)

    renderWith(<SessionsList />)
    const button = await screen.findByRole("button", {
      name: "Tout révoquer sauf cette session",
    })
    await userEvent.click(button)

    expect(confirmSpy).toHaveBeenCalled()
    expect(bulkSpy).toHaveBeenCalledTimes(1)
    // listSessions called twice: initial load + post-invalidation refetch.
    await waitFor(() => expect(listSpy).toHaveBeenCalledTimes(2))
  })

  it("does NOT call revokeOtherSessions when confirm is dismissed", async () => {
    confirmSpy.mockReturnValue(false)
    vi.spyOn(sessionsApi, "listSessions").mockResolvedValue({
      data: [
        buildSession({ id: "current", is_current: true }),
        buildSession({ id: "other-1", is_current: false }),
      ],
    })
    const bulkSpy = vi
      .spyOn(sessionsApi, "revokeOtherSessions")
      .mockResolvedValue(undefined)

    renderWith(<SessionsList />)
    const button = await screen.findByRole("button", {
      name: "Tout révoquer sauf cette session",
    })
    await userEvent.click(button)

    expect(confirmSpy).toHaveBeenCalled()
    expect(bulkSpy).not.toHaveBeenCalled()
  })
})
