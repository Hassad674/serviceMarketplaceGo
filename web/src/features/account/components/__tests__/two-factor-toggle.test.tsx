import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {
  QueryClient,
  QueryClientProvider,
  useQueryClient,
} from "@tanstack/react-query"
import type { ReactNode } from "react"
import { TwoFactorToggle } from "../two-factor-toggle"
import { ApiError } from "@/shared/lib/api-client"
import type { SessionResponse } from "@/shared/hooks/use-user"

// next-intl: passes the key through. Locale "fr" by default.
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Sonner toast — pulled in via lazy import in the toggle. Mock both
// the top-level export and the default to satisfy whichever shape the
// component uses.
vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock("../../api/two-factor-api", () => ({
  requestEnableTwoFactor: vi.fn(),
  confirmEnableTwoFactor: vi.fn(),
  disableTwoFactor: vi.fn(),
}))

import {
  requestEnableTwoFactor,
  confirmEnableTwoFactor,
  disableTwoFactor,
} from "../../api/two-factor-api"
import { toast } from "sonner"

const mockRequestEnable = vi.mocked(requestEnableTwoFactor)
const mockConfirmEnable = vi.mocked(confirmEnableTwoFactor)
const mockDisable = vi.mocked(disableTwoFactor)
const mockToastSuccess = vi.mocked(toast.success)

beforeEach(() => {
  vi.clearAllMocks()
})

// Build a SessionResponse with the supplied 2FA flag. The TwoFactorToggle
// reads `data.user.two_factor_email_enabled` via useUser(), so any test
// that wants to exercise the "initial state from session cache" branch
// pre-seeds the ["session"] key with this object.
function buildSession(twoFA: boolean | undefined): SessionResponse {
  return {
    user: {
      id: "u1",
      email: "alice@example.com",
      first_name: "Alice",
      last_name: "Doe",
      display_name: "Alice Doe",
      role: "provider",
      referrer_enabled: false,
      email_verified: true,
      kyc_status: "none",
      // Only attach the flag when the test cares — leaving it
      // undefined exercises the "no field" branch (toggle defaults
      // to off, as if the backend hasn't shipped the field yet).
      ...(twoFA === undefined ? {} : { two_factor_email_enabled: twoFA }),
      created_at: "2026-01-01T00:00:00Z",
    },
    organization: null,
  }
}

// Helper to render with a fresh QueryClient + an optional preloaded
// session payload. Returns the client so a test can later trigger
// invalidations and observe a cache mutation.
function renderToggle(opts: {
  initialEnabled?: boolean
  session?: SessionResponse
} = {}) {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, refetchOnMount: false },
    },
  })
  if (opts.session) {
    client.setQueryData(["session"], opts.session)
  }
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
  return {
    ...render(<TwoFactorToggle initialEnabled={opts.initialEnabled} />, {
      wrapper: Wrapper,
    }),
    client,
  }
}

describe("TwoFactorToggle", () => {
  it("renders the off state with the enable CTA when initialEnabled is false", () => {
    renderToggle({ initialEnabled: false })
    expect(
      screen.getByRole("button", { name: "enableCta" }),
    ).toBeInTheDocument()
  })

  // FIX-2FA regression: when /auth/me reports two_factor_email_enabled=true,
  // the toggle MUST render the ON state on first paint — the bug we are
  // fixing was that the toggle stayed OFF after reload even when the DB
  // said the user had 2FA enabled.
  it("derives initial state from the session cache (toggle ON when /me says enabled)", async () => {
    renderToggle({ session: buildSession(true) })
    // Disable CTA is the on-state CTA — its presence proves the toggle
    // resolved its mount-time value from the session cache.
    expect(
      screen.getByRole("button", { name: "disableCta" }),
    ).toBeInTheDocument()
  })

  it("defaults to OFF when the session is missing or the flag is undefined", () => {
    renderToggle({ session: buildSession(undefined) })
    expect(
      screen.getByRole("button", { name: "enableCta" }),
    ).toBeInTheDocument()
  })

  it("syncs the toggle when the session cache transitions from undefined to true", async () => {
    const { client } = renderToggle({ initialEnabled: false })
    // Mount with no session — toggle starts OFF.
    expect(
      screen.getByRole("button", { name: "enableCta" }),
    ).toBeInTheDocument()
    // Simulate /auth/me landing AFTER mount.
    client.setQueryData(["session"], buildSession(true))
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "disableCta" }),
      ).toBeInTheDocument()
    })
  })

  it("walks through the two-step enable flow on click", async () => {
    mockRequestEnable.mockResolvedValueOnce({
      requires_confirmation: true,
      challenge_id: "challenge-789",
    })
    mockConfirmEnable.mockResolvedValueOnce({ enabled: true })

    const user = userEvent.setup()
    renderToggle({ initialEnabled: false })

    await user.click(screen.getByRole("button", { name: "enableCta" }))

    await waitFor(() => {
      expect(mockRequestEnable).toHaveBeenCalledTimes(1)
    })

    // Step 2: code field appears, user types in code, confirms.
    const codeInput = await screen.findByLabelText("codeLabel")
    await user.type(codeInput, "987654")
    await user.click(
      screen.getByRole("button", { name: "confirmEnableCta" }),
    )

    await waitFor(() => {
      expect(mockConfirmEnable).toHaveBeenCalledWith("987654")
    })
    await waitFor(() => {
      expect(mockToastSuccess).toHaveBeenCalledWith("toasts.enabled")
    })
    // Local state flipped to enabled; the disable CTA is now
    // visible.
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "disableCta" }),
      ).toBeInTheDocument()
    })
  })

  // FIX-2FA: a successful enable invalidates the session query so the
  // sidebar / account header / any other useUser() consumer refreshes
  // its 2FA badge on next render — otherwise the badge would lag behind
  // the toggle until the next page navigation.
  it("invalidates the session cache after a successful enable", async () => {
    mockRequestEnable.mockResolvedValueOnce({
      requires_confirmation: true,
      challenge_id: "challenge-x",
    })
    mockConfirmEnable.mockResolvedValueOnce({ enabled: true })

    const { client } = renderToggle({ initialEnabled: false })
    const invalidateSpy = vi.spyOn(client, "invalidateQueries")

    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "enableCta" }))
    const codeInput = await screen.findByLabelText("codeLabel")
    await user.type(codeInput, "123456")
    await user.click(
      screen.getByRole("button", { name: "confirmEnableCta" }),
    )

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["session"] })
    })
  })

  it("invalidates the session cache after a successful disable", async () => {
    mockDisable.mockResolvedValueOnce({ enabled: false })
    const { client } = renderToggle({
      session: buildSession(true),
    })
    const invalidateSpy = vi.spyOn(client, "invalidateQueries")

    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "disableCta" }))
    const passwordInput = await screen.findByLabelText("currentPasswordLabel")
    await user.type(passwordInput, "Passw0rd!")
    await user.click(
      screen.getByRole("button", { name: "confirmDisableCta" }),
    )

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["session"] })
    })
  })

  it("surfaces an invalid_code error and clears the input", async () => {
    mockRequestEnable.mockResolvedValueOnce({
      requires_confirmation: true,
      challenge_id: "challenge-789",
    })
    mockConfirmEnable.mockRejectedValueOnce(
      new ApiError(400, "invalid_code", "incorrect"),
    )

    const user = userEvent.setup()
    renderToggle({ initialEnabled: false })

    await user.click(screen.getByRole("button", { name: "enableCta" }))
    const codeInput = (await screen.findByLabelText(
      "codeLabel",
    )) as HTMLInputElement
    await user.type(codeInput, "123456")
    await user.click(
      screen.getByRole("button", { name: "confirmEnableCta" }),
    )

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("errors.invalidCode")
    })
    expect(codeInput.value).toBe("")
  })

  it("surfaces a challenge_expired error using the dedicated key", async () => {
    mockRequestEnable.mockResolvedValueOnce({
      requires_confirmation: true,
      challenge_id: "challenge-789",
    })
    mockConfirmEnable.mockRejectedValueOnce(
      new ApiError(400, "challenge_expired", "expired"),
    )

    const user = userEvent.setup()
    renderToggle({ initialEnabled: false })

    await user.click(screen.getByRole("button", { name: "enableCta" }))
    const codeInput = await screen.findByLabelText("codeLabel")
    await user.type(codeInput, "123456")
    await user.click(
      screen.getByRole("button", { name: "confirmEnableCta" }),
    )

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        "errors.challengeExpired",
      )
    })
  })

  it("surfaces a too_many_attempts error", async () => {
    mockRequestEnable.mockResolvedValueOnce({
      requires_confirmation: true,
      challenge_id: "challenge-789",
    })
    mockConfirmEnable.mockRejectedValueOnce(
      new ApiError(429, "too_many_attempts", "throttled"),
    )

    const user = userEvent.setup()
    renderToggle({ initialEnabled: false })

    await user.click(screen.getByRole("button", { name: "enableCta" }))
    const codeInput = await screen.findByLabelText("codeLabel")
    await user.type(codeInput, "999999")
    await user.click(
      screen.getByRole("button", { name: "confirmEnableCta" }),
    )

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        "errors.tooManyAttempts",
      )
    })
  })

  it("disables 2FA after a successful current_password check", async () => {
    mockDisable.mockResolvedValueOnce({ enabled: false })

    const user = userEvent.setup()
    renderToggle({ initialEnabled: true })

    // Toggle is in the on state, click disable.
    await user.click(screen.getByRole("button", { name: "disableCta" }))
    const passwordInput = await screen.findByLabelText("currentPasswordLabel")
    await user.type(passwordInput, "Passw0rd!")
    await user.click(
      screen.getByRole("button", { name: "confirmDisableCta" }),
    )

    await waitFor(() => {
      expect(mockDisable).toHaveBeenCalledWith({
        current_password: "Passw0rd!",
      })
    })
    await waitFor(() => {
      expect(mockToastSuccess).toHaveBeenCalledWith("toasts.disabled")
    })
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "enableCta" }),
      ).toBeInTheDocument()
    })
  })

  it("rejects the disable flow when the password is wrong", async () => {
    mockDisable.mockRejectedValueOnce(
      new ApiError(401, "invalid_credentials", "wrong"),
    )

    const user = userEvent.setup()
    renderToggle({ initialEnabled: true })

    await user.click(screen.getByRole("button", { name: "disableCta" }))
    const passwordInput = (await screen.findByLabelText(
      "currentPasswordLabel",
    )) as HTMLInputElement
    await user.type(passwordInput, "wrong")
    await user.click(
      screen.getByRole("button", { name: "confirmDisableCta" }),
    )

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        "errors.invalidCredentials",
      )
    })
    expect(passwordInput.value).toBe("")
  })

  it("validates the code length before hitting the API", async () => {
    mockRequestEnable.mockResolvedValueOnce({
      requires_confirmation: true,
      challenge_id: "challenge-1",
    })

    const user = userEvent.setup()
    renderToggle({ initialEnabled: false })

    await user.click(screen.getByRole("button", { name: "enableCta" }))
    const codeInput = await screen.findByLabelText("codeLabel")
    await user.type(codeInput, "123")
    // Button is disabled when length < 6 — sanity check.
    expect(
      screen.getByRole("button", { name: "confirmEnableCta" }),
    ).toBeDisabled()
    expect(mockConfirmEnable).not.toHaveBeenCalled()
  })

  it("disables the enable CTA while a request is in flight", async () => {
    let resolveRequest: ((value: { requires_confirmation: true; challenge_id: string }) => void) | undefined
    mockRequestEnable.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveRequest = resolve
        }),
    )

    const user = userEvent.setup()
    renderToggle({ initialEnabled: false })

    const enableButton = screen.getByRole("button", { name: "enableCta" })
    await user.click(enableButton)
    // While the request is pending the same CTA button stays mounted
    // but its `disabled` attribute is set — a second click is a no-op
    // (the React handler bails on `busy === true`).
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "enableCta" }),
      ).toBeDisabled()
    })
    // The label switches to the "saving" key to signal the in-flight
    // state to the user.
    expect(screen.getByText("saving")).toBeInTheDocument()
    expect(mockRequestEnable).toHaveBeenCalledTimes(1)

    resolveRequest?.({ requires_confirmation: true, challenge_id: "x" })
    await screen.findByLabelText("codeLabel")
  })
})
