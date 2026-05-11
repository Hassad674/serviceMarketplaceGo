import { describe, expect, it, vi, beforeEach, afterEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { NextIntlClientProvider } from "next-intl"
import { SessionsList } from "../sessions-list"
import * as sessionsApi from "../../api/sessions-api"

// Minimal fr.json subset used by the SessionsList component.
const messages = {
  account: {
    security: {
      retry: "Réessayer",
      sessions: {
        title: "Sessions actives",
        subtitle:
          "Voici les appareils qui ont accès à ton compte. Révoque toute session que tu ne reconnais pas.",
        currentBadge: "Cette session",
        revoke: "Révoquer",
        revoking: "Révocation...",
        revokeOthers: "Tout révoquer sauf cette session",
        revokeOthersConfirm: "Confirmer ?",
        empty: "Aucune session active à afficher.",
        error: "Impossible de charger les sessions. Réessaie dans un instant.",
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

describe("SessionsList", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("renders the Malt-style row with device label, location and revoke button", async () => {
    vi.spyOn(sessionsApi, "listSessions").mockResolvedValue({
      data: [
        {
          id: "11111111-1111-1111-1111-111111111111",
          device_label: "Ordinateur de bureau (Chrome)",
          browser: "Chrome",
          os: "Windows",
          city: "Paris",
          country_code: "FR",
          ip_anonymized: "203.0.113.0/24",
          login_method: "password",
          created_at: "2026-05-11T08:48:46Z",
          last_used_at: "2026-05-11T08:48:46Z",
          expires_at: "2026-05-25T08:48:46Z",
          is_current: false,
        },
      ],
    })

    renderWith(<SessionsList />)

    await screen.findByText("Ordinateur de bureau (Chrome)")
    expect(screen.getByText(/Paris/)).toBeInTheDocument()
    expect(screen.getByText("203.0.113.0/24")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Révoquer" })).toBeInTheDocument()
  })

  it("renders the 'Cette session' badge on the current row and hides its Révoquer button", async () => {
    vi.spyOn(sessionsApi, "listSessions").mockResolvedValue({
      data: [
        {
          id: "22222222-2222-2222-2222-222222222222",
          device_label: "iPhone (Safari)",
          os: "iOS",
          city: "Lyon",
          country_code: "FR",
          ip_anonymized: "198.51.100.0/24",
          login_method: "password",
          created_at: "2026-05-11T08:00:00Z",
          last_used_at: "2026-05-11T08:30:00Z",
          expires_at: "2026-05-25T08:00:00Z",
          is_current: true,
        },
      ],
    })

    renderWith(<SessionsList />)

    await screen.findByText("iPhone (Safari)")
    expect(screen.getByText(/Cette session/i)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "Révoquer" })).not.toBeInTheDocument()
  })

  it("revokes a session on click and removes the row optimistically", async () => {
    vi.spyOn(sessionsApi, "listSessions").mockResolvedValue({
      data: [
        {
          id: "33333333-3333-3333-3333-333333333333",
          device_label: "Android (Chrome)",
          os: "Android",
          city: "",
          country_code: "",
          ip_anonymized: "192.0.2.0/24",
          login_method: "password",
          created_at: "2026-05-10T08:00:00Z",
          last_used_at: "2026-05-10T08:00:00Z",
          expires_at: "2026-05-25T08:00:00Z",
          is_current: false,
        },
      ],
    })
    const revokeSpy = vi
      .spyOn(sessionsApi, "revokeSession")
      .mockResolvedValue(undefined)

    renderWith(<SessionsList />)
    await screen.findByText("Android (Chrome)")
    await userEvent.click(screen.getByRole("button", { name: "Révoquer" }))

    expect(revokeSpy).toHaveBeenCalledWith("33333333-3333-3333-3333-333333333333")
    await waitFor(() =>
      expect(screen.queryByText("Android (Chrome)")).not.toBeInTheDocument(),
    )
  })

  it("renders the empty state when no sessions are returned", async () => {
    vi.spyOn(sessionsApi, "listSessions").mockResolvedValue({ data: [] })
    renderWith(<SessionsList />)
    await screen.findByText("Aucune session active à afficher.")
  })

  it("renders the error state on fetch failure", async () => {
    vi.spyOn(sessionsApi, "listSessions").mockRejectedValue(new Error("nope"))
    renderWith(<SessionsList />)
    await screen.findByText(
      "Impossible de charger les sessions. Réessaie dans un instant.",
    )
  })
})
