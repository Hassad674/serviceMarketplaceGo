/**
 * PostHogProvider — login/logout identify + group flow.
 *
 * Mounts the provider with a mocked useSession() and asserts that:
 *   - on resolved session WITH consent, posthog.identify is called
 *     with the user id + role and the org group is attached.
 *   - on resolved-no-session WITH consent, posthog.reset is called.
 *   - WITHOUT consent, the SDK is never initialised — pause-before-
 *     consent invariant (Phase A.2).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { render, act } from "@testing-library/react"

const sdkMock = vi.hoisted(() => ({
  init: vi.fn(),
  identify: vi.fn(),
  group: vi.fn(),
  reset: vi.fn(),
  capture: vi.fn(),
  has_opted_out_capturing: vi.fn(() => false),
  opt_in_capturing: vi.fn(),
  opt_out_capturing: vi.fn(),
  debug: vi.fn(),
}))
vi.mock("posthog-js", () => ({
  default: sdkMock,
  ...sdkMock,
}))

const sessionState = vi.hoisted(() => ({
  data: null as
    | null
    | {
        user: {
          id: string
          role: string
          email_verified: boolean
          referrer_enabled: boolean
        }
        organization: { id: string; type: string; member_role: string } | null
      },
}))

vi.mock("@/shared/hooks/use-user", () => ({
  useSession: () => ({ data: sessionState.data }),
}))

beforeEach(() => {
  Object.values(sdkMock).forEach((fn) => {
    if (typeof fn === "function" && "mockClear" in fn) fn.mockClear()
  })
  process.env.NEXT_PUBLIC_POSTHOG_KEY = "phc_test_key"
  window.localStorage.clear()
  vi.resetModules()
})

afterEach(() => {
  sessionState.data = null
  window.localStorage.clear()
})

/** Pre-set the analytics consent so the provider initialises the SDK. */
function grantConsent() {
  window.localStorage.setItem("marketplace.analytics.consent", "accepted")
}

describe("PostHogProvider", () => {
  it("identifies the user and attaches the org group when logged in (with consent)", async () => {
    grantConsent()
    sessionState.data = {
      user: {
        id: "user-7",
        role: "agency",
        email_verified: true,
        referrer_enabled: false,
      },
      organization: { id: "org-42", type: "agency", member_role: "owner" },
    }

    const { PostHogProvider } = await import("../posthog-provider")
    render(<PostHogProvider />)

    expect(sdkMock.init).toHaveBeenCalledTimes(1)
    expect(sdkMock.identify).toHaveBeenCalledWith("user-7", {
      role: "agency",
      email_verified: true,
      referrer_enabled: false,
    })
    expect(sdkMock.group).toHaveBeenCalledWith("organization", "org-42", {
      type: "agency",
      member_role: "owner",
    })
  })

  it("resets when there is no session (logged out + consent granted)", async () => {
    grantConsent()
    sessionState.data = null
    const { PostHogProvider } = await import("../posthog-provider")
    render(<PostHogProvider />)
    // Reset is only called once SDK is initialised. With consent
    // granted the provider boots the SDK on mount.
    expect(sdkMock.reset).toHaveBeenCalled()
  })

  it("does NOT initialise the SDK before consent is granted (RGPD pause-before-consent)", async () => {
    // No consent in localStorage. Mount the provider with a session
    // that would normally trigger identify().
    sessionState.data = {
      user: {
        id: "user-7",
        role: "agency",
        email_verified: true,
        referrer_enabled: false,
      },
      organization: { id: "org-42", type: "agency", member_role: "owner" },
    }
    const { PostHogProvider } = await import("../posthog-provider")
    render(<PostHogProvider />)
    expect(sdkMock.init).not.toHaveBeenCalled()
    expect(sdkMock.identify).not.toHaveBeenCalled()
    expect(sdkMock.group).not.toHaveBeenCalled()
  })

  it("boots the SDK after a runtime consent flip (analytics:consent-changed)", async () => {
    sessionState.data = {
      user: {
        id: "user-7",
        role: "agency",
        email_verified: true,
        referrer_enabled: false,
      },
      organization: null,
    }
    const { PostHogProvider } = await import("../posthog-provider")
    render(<PostHogProvider />)
    expect(sdkMock.init).not.toHaveBeenCalled()
    // Simulate the CMP flipping analytics on.
    grantConsent()
    await act(async () => {
      window.dispatchEvent(new CustomEvent("analytics:consent-changed"))
    })
    expect(sdkMock.init).toHaveBeenCalledTimes(1)
    expect(sdkMock.identify).toHaveBeenCalledWith("user-7", expect.any(Object))
  })
})
