import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { TwoFactorToggle } from "../two-factor-toggle"
import { ApiError } from "@/shared/lib/api-client"

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

describe("TwoFactorToggle", () => {
  it("renders the off state with the enable CTA when initialEnabled is false", () => {
    render(<TwoFactorToggle initialEnabled={false} />)
    expect(
      screen.getByRole("button", { name: "enableCta" }),
    ).toBeInTheDocument()
  })

  it("walks through the two-step enable flow on click", async () => {
    mockRequestEnable.mockResolvedValueOnce({
      requires_confirmation: true,
      challenge_id: "challenge-789",
    })
    mockConfirmEnable.mockResolvedValueOnce({ enabled: true })

    const user = userEvent.setup()
    render(<TwoFactorToggle initialEnabled={false} />)

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

  it("surfaces an invalid_code error and clears the input", async () => {
    mockRequestEnable.mockResolvedValueOnce({
      requires_confirmation: true,
      challenge_id: "challenge-789",
    })
    mockConfirmEnable.mockRejectedValueOnce(
      new ApiError(400, "invalid_code", "incorrect"),
    )

    const user = userEvent.setup()
    render(<TwoFactorToggle initialEnabled={false} />)

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

  it("disables 2FA after a successful current_password check", async () => {
    mockDisable.mockResolvedValueOnce({ enabled: false })

    const user = userEvent.setup()
    render(<TwoFactorToggle initialEnabled={true} />)

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
    render(<TwoFactorToggle initialEnabled={true} />)

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
    render(<TwoFactorToggle initialEnabled={false} />)

    await user.click(screen.getByRole("button", { name: "enableCta" }))
    const codeInput = await screen.findByLabelText("codeLabel")
    await user.type(codeInput, "123")
    // Button is disabled when length < 6 — sanity check.
    expect(
      screen.getByRole("button", { name: "confirmEnableCta" }),
    ).toBeDisabled()
    expect(mockConfirmEnable).not.toHaveBeenCalled()
  })
})
