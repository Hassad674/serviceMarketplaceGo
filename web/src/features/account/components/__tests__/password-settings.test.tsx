import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, waitFor, act } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { ApiError } from "@/shared/lib/api-client"
import { PasswordSettings } from "../password-settings"

// next-intl: pass the key through as the rendered text.
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// sonner: capture toasts.
const toastSuccess = vi.fn()
const toastError = vi.fn()
vi.mock("sonner", () => ({
  toast: {
    success: (...args: unknown[]) => toastSuccess(...args),
    error: (...args: unknown[]) => toastError(...args),
  },
}))

// Mutation hook: controllable per-test.
const mutateMock = vi.fn()
let lastOnSuccess: (() => void) | null = null
let lastOnError: ((err: unknown) => void) | null = null
let isPending = false
vi.mock("../../hooks/use-change-password", () => ({
  useChangePassword: () => ({
    isPending,
    mutate: (
      body: unknown,
      opts: { onSuccess: () => void; onError: (err: unknown) => void },
    ) => {
      mutateMock(body)
      lastOnSuccess = opts.onSuccess
      lastOnError = opts.onError
    },
  }),
}))

const originalLocation = window.location

beforeEach(() => {
  mutateMock.mockReset()
  toastSuccess.mockReset()
  toastError.mockReset()
  lastOnSuccess = null
  lastOnError = null
  isPending = false
  Object.defineProperty(window, "location", {
    configurable: true,
    value: { ...originalLocation, href: "" },
  })
})

afterEach(() => {
  Object.defineProperty(window, "location", {
    configurable: true,
    value: originalLocation,
  })
})

describe("PasswordSettings", () => {
  it("renders the three password fields and submit button", () => {
    render(<PasswordSettings />)
    expect(
      screen.getByLabelText("currentPassword"),
    ).toBeInTheDocument()
    expect(screen.getByLabelText("newPassword")).toBeInTheDocument()
    expect(
      screen.getByLabelText("confirmPassword"),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "changePasswordCta" }),
    ).toBeInTheDocument()
  })

  it("requires the current password and rejects weak passwords", async () => {
    const user = userEvent.setup()
    render(<PasswordSettings />)

    await user.click(
      screen.getByRole("button", { name: "changePasswordCta" }),
    )

    await waitFor(() => {
      expect(
        screen.getByText("errors.passwordRequired"),
      ).toBeInTheDocument()
    })
    expect(mutateMock).not.toHaveBeenCalled()
  })

  it("rejects when the new password is too weak", async () => {
    const user = userEvent.setup()
    render(<PasswordSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(screen.getByLabelText("newPassword"), "weak")
    await user.type(screen.getByLabelText("confirmPassword"), "weak")
    await user.click(
      screen.getByRole("button", { name: "changePasswordCta" }),
    )

    await waitFor(() => {
      expect(screen.getByText("errors.weakPassword")).toBeInTheDocument()
    })
    expect(mutateMock).not.toHaveBeenCalled()
  })

  it("rejects when confirm does not match new password", async () => {
    const user = userEvent.setup()
    render(<PasswordSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("newPassword"),
      "NewPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("confirmPassword"),
      "DifferentPass1!",
    )
    await user.click(
      screen.getByRole("button", { name: "changePasswordCta" }),
    )

    await waitFor(() => {
      expect(
        screen.getByText("errors.passwordMismatch"),
      ).toBeInTheDocument()
    })
    expect(mutateMock).not.toHaveBeenCalled()
  })

  it("submits valid values and on success toasts + redirects to /login", async () => {
    const user = userEvent.setup()
    render(<PasswordSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("newPassword"),
      "NewPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("confirmPassword"),
      "NewPass1!aaa",
    )
    await user.click(
      screen.getByRole("button", { name: "changePasswordCta" }),
    )

    await waitFor(() => {
      expect(mutateMock).toHaveBeenCalledWith({
        current_password: "OldPass1!aaa",
        new_password: "NewPass1!aaa",
      })
    })

    await act(async () => {
      lastOnSuccess?.()
    })
    expect(toastSuccess).toHaveBeenCalledWith("passwordChangedSuccess")
    expect(window.location.href).toBe("/login")
  })

  it("maps invalid_credentials to an inline error on the current_password field and clears all password fields", async () => {
    const user = userEvent.setup()
    render(<PasswordSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("newPassword"),
      "NewPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("confirmPassword"),
      "NewPass1!aaa",
    )
    await user.click(
      screen.getByRole("button", { name: "changePasswordCta" }),
    )

    await waitFor(() => expect(mutateMock).toHaveBeenCalled())

    await act(async () => {
      lastOnError?.(
        new ApiError(401, "invalid_credentials", "wrong", null),
      )
    })

    await waitFor(() => {
      expect(
        screen.getByText("errors.invalidCredentials"),
      ).toBeInTheDocument()
    })

    expect(
      (screen.getByLabelText("currentPassword") as HTMLInputElement).value,
    ).toBe("")
    expect(
      (screen.getByLabelText("newPassword") as HTMLInputElement).value,
    ).toBe("")
    expect(
      (screen.getByLabelText("confirmPassword") as HTMLInputElement).value,
    ).toBe("")
  })

  it("maps weak_password from backend to the new_password field", async () => {
    const user = userEvent.setup()
    render(<PasswordSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("newPassword"),
      "NewPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("confirmPassword"),
      "NewPass1!aaa",
    )
    await user.click(
      screen.getByRole("button", { name: "changePasswordCta" }),
    )

    await waitFor(() => expect(mutateMock).toHaveBeenCalled())

    await act(async () => {
      lastOnError?.(new ApiError(400, "weak_password", "weak", null))
    })

    await waitFor(() => {
      expect(screen.getByText("errors.weakPassword")).toBeInTheDocument()
    })
  })

  it("maps same_password to the new_password field", async () => {
    const user = userEvent.setup()
    render(<PasswordSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("newPassword"),
      "NewPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("confirmPassword"),
      "NewPass1!aaa",
    )
    await user.click(
      screen.getByRole("button", { name: "changePasswordCta" }),
    )

    await waitFor(() => expect(mutateMock).toHaveBeenCalled())

    await act(async () => {
      lastOnError?.(new ApiError(400, "same_password", "same", null))
    })

    await waitFor(() => {
      expect(screen.getByText("errors.samePassword")).toBeInTheDocument()
    })
  })

  it("toasts a generic error for non-ApiError failures and clears passwords", async () => {
    const user = userEvent.setup()
    render(<PasswordSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("newPassword"),
      "NewPass1!aaa",
    )
    await user.type(
      screen.getByLabelText("confirmPassword"),
      "NewPass1!aaa",
    )
    await user.click(
      screen.getByRole("button", { name: "changePasswordCta" }),
    )

    await waitFor(() => expect(mutateMock).toHaveBeenCalled())

    await act(async () => {
      lastOnError?.(new Error("network"))
    })

    await waitFor(() => {
      expect(toastError).toHaveBeenCalledWith("errors.generic")
    })
    expect(
      (screen.getByLabelText("currentPassword") as HTMLInputElement).value,
    ).toBe("")
  })
})
