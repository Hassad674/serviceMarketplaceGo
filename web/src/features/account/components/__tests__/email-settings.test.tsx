import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, waitFor, act } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { ApiError } from "@/shared/lib/api-client"
import { EmailSettings } from "../email-settings"

// next-intl: pass the key through as the rendered text.
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// useUser stub — return a stable account email.
vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({ data: { id: "u1", email: "current@example.com" } }),
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
vi.mock("../../hooks/use-change-email", () => ({
  useChangeEmail: () => ({
    isPending,
    mutate: (body: unknown, opts: { onSuccess: () => void; onError: (err: unknown) => void }) => {
      mutateMock(body)
      lastOnSuccess = opts.onSuccess
      lastOnError = opts.onError
    },
  }),
}))

// Stub window.location to capture redirects.
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

describe("EmailSettings", () => {
  it("renders the current email and the form fields", () => {
    render(<EmailSettings />)
    expect(screen.getByText("current@example.com")).toBeInTheDocument()
    expect(
      screen.getByLabelText("currentPassword"),
    ).toBeInTheDocument()
    expect(screen.getByLabelText("newEmail")).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "changeEmailCta" }),
    ).toBeInTheDocument()
  })

  it("validates required password and email format on submit", async () => {
    const user = userEvent.setup()
    render(<EmailSettings />)

    await user.click(screen.getByRole("button", { name: "changeEmailCta" }))

    await waitFor(() => {
      expect(
        screen.getByText("errors.passwordRequired"),
      ).toBeInTheDocument()
    })
    expect(mutateMock).not.toHaveBeenCalled()
  })

  it("rejects invalid email format", async () => {
    const user = userEvent.setup()
    render(<EmailSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(screen.getByLabelText("newEmail"), "not-an-email")
    await user.click(screen.getByRole("button", { name: "changeEmailCta" }))

    await waitFor(() => {
      expect(screen.getByText("errors.invalidEmail")).toBeInTheDocument()
    })
    expect(mutateMock).not.toHaveBeenCalled()
  })

  it("submits valid values and on success toasts + redirects to /login", async () => {
    const user = userEvent.setup()
    render(<EmailSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(screen.getByLabelText("newEmail"), "new@example.com")
    await user.click(screen.getByRole("button", { name: "changeEmailCta" }))

    await waitFor(() => {
      expect(mutateMock).toHaveBeenCalledWith({
        current_password: "OldPass1!aaa",
        new_email: "new@example.com",
      })
    })

    // Trigger the mutation success path.
    await act(async () => {
      lastOnSuccess?.()
    })

    expect(toastSuccess).toHaveBeenCalledWith("emailChangedSuccess")
    expect(window.location.href).toBe("/login")
  })

  it("maps invalid_credentials to an inline error on the password field", async () => {
    const user = userEvent.setup()
    render(<EmailSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(screen.getByLabelText("newEmail"), "new@example.com")
    await user.click(screen.getByRole("button", { name: "changeEmailCta" }))

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
    // Password field should be cleared.
    const passwordInput = screen.getByLabelText(
      "currentPassword",
    ) as HTMLInputElement
    expect(passwordInput.value).toBe("")
  })

  it("maps email_already_exists to an inline error on the email field", async () => {
    const user = userEvent.setup()
    render(<EmailSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(screen.getByLabelText("newEmail"), "taken@example.com")
    await user.click(screen.getByRole("button", { name: "changeEmailCta" }))

    await waitFor(() => expect(mutateMock).toHaveBeenCalled())

    await act(async () => {
      lastOnError?.(
        new ApiError(409, "email_already_exists", "taken", null),
      )
    })

    await waitFor(() => {
      expect(
        screen.getByText("errors.emailAlreadyExists"),
      ).toBeInTheDocument()
    })
    expect(window.location.href).toBe("")
  })

  it("falls back to a toast for unmapped error codes", async () => {
    const user = userEvent.setup()
    render(<EmailSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(screen.getByLabelText("newEmail"), "new@example.com")
    await user.click(screen.getByRole("button", { name: "changeEmailCta" }))

    await waitFor(() => expect(mutateMock).toHaveBeenCalled())

    await act(async () => {
      lastOnError?.(new ApiError(500, "unknown_error", "boom", null))
    })

    await waitFor(() => {
      expect(toastError).toHaveBeenCalledWith("errors.generic")
    })
  })

  it("toasts a generic error for non-ApiError failures", async () => {
    const user = userEvent.setup()
    render(<EmailSettings />)

    await user.type(
      screen.getByLabelText("currentPassword"),
      "OldPass1!aaa",
    )
    await user.type(screen.getByLabelText("newEmail"), "new@example.com")
    await user.click(screen.getByRole("button", { name: "changeEmailCta" }))

    await waitFor(() => expect(mutateMock).toHaveBeenCalled())

    await act(async () => {
      lastOnError?.(new Error("network"))
    })

    await waitFor(() => {
      expect(toastError).toHaveBeenCalledWith("errors.generic")
    })
  })
})
