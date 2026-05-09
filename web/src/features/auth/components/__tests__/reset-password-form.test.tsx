import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { ResetPasswordForm } from "../reset-password-form"

vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    ...rest
  }: {
    href: string
    children: React.ReactNode
    className?: string
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))

vi.mock("@/features/auth/api/auth-api", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@/features/auth/api/auth-api")>()
  return {
    ...actual,
    resetPassword: vi.fn(),
  }
})

import { resetPassword } from "@/features/auth/api/auth-api"
const mockResetPassword = vi.mocked(resetPassword)

function renderForm(token = "fake-token") {
  return render(
    <NextIntlClientProvider locale="fr" messages={messages}>
      <ResetPasswordForm token={token} />
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("ResetPasswordForm", () => {
  it("renders the new-password and confirm fields when a token is supplied", () => {
    renderForm("fake-token")
    expect(
      screen.getByLabelText(messages.auth.newPassword),
    ).toBeInTheDocument()
    expect(
      screen.getByLabelText(messages.auth.confirmPassword),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: messages.auth.resetPassword }),
    ).toBeInTheDocument()
  })

  it("renders the invalid-link state when the token is empty", () => {
    renderForm("")
    expect(screen.getByRole("alert")).toBeInTheDocument()
    expect(
      screen.getByText(messages.auth.invalidLinkDesc),
    ).toBeInTheDocument()
    const link = screen.getByRole("link", {
      name: new RegExp(messages.common.requestNewLink, "i"),
    })
    expect(link).toHaveAttribute("href", "/forgot-password")
  })

  it("shows a validation error for a too-short password", async () => {
    const user = userEvent.setup()
    renderForm("fake-token")

    await user.type(
      screen.getByLabelText(messages.auth.newPassword),
      "short",
    )
    await user.type(
      screen.getByLabelText(messages.auth.confirmPassword),
      "short",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.resetPassword }),
    )

    await waitFor(() => {
      expect(
        screen.getByText(/at least 10 characters/i),
      ).toBeInTheDocument()
    })
  })

  it("shows a mismatch error when passwords do not match", async () => {
    const user = userEvent.setup()
    renderForm("fake-token")

    await user.type(
      screen.getByLabelText(messages.auth.newPassword),
      "Passw0rd!@#",
    )
    await user.type(
      screen.getByLabelText(messages.auth.confirmPassword),
      "DifferentPass1!",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.resetPassword }),
    )

    await waitFor(() => {
      expect(screen.getByText(/do not match/i)).toBeInTheDocument()
    })
  })

  it("shows the loading copy while submitting", async () => {
    mockResetPassword.mockImplementation(
      () => new Promise<{ message: string }>(() => {}),
    )

    const user = userEvent.setup()
    renderForm("fake-token")

    await user.type(
      screen.getByLabelText(messages.auth.newPassword),
      "Passw0rd!@#",
    )
    await user.type(
      screen.getByLabelText(messages.auth.confirmPassword),
      "Passw0rd!@#",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.resetPassword }),
    )

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: messages.auth.resetting }),
      ).toBeInTheDocument()
    })
  })

  it("renders the success state when the API call resolves", async () => {
    mockResetPassword.mockResolvedValueOnce({ message: "ok" })

    const user = userEvent.setup()
    renderForm("fake-token")

    await user.type(
      screen.getByLabelText(messages.auth.newPassword),
      "Passw0rd!@#",
    )
    await user.type(
      screen.getByLabelText(messages.auth.confirmPassword),
      "Passw0rd!@#",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.resetPassword }),
    )

    await waitFor(() => {
      expect(screen.getByRole("status")).toBeInTheDocument()
    })
    expect(screen.getByText(messages.auth.resetSuccess)).toBeInTheDocument()
    const signInLink = screen.getByRole("link", {
      name: messages.common.signIn,
    })
    expect(signInLink).toHaveAttribute("href", "/login")
  })

  it("renders an inline error when the API throws", async () => {
    mockResetPassword.mockRejectedValueOnce(new Error("Token expired"))

    const user = userEvent.setup()
    renderForm("fake-token")

    await user.type(
      screen.getByLabelText(messages.auth.newPassword),
      "Passw0rd!@#",
    )
    await user.type(
      screen.getByLabelText(messages.auth.confirmPassword),
      "Passw0rd!@#",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.resetPassword }),
    )

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("Token expired")
    })
  })

  it("toggles password visibility on the new-password field", async () => {
    const user = userEvent.setup()
    renderForm("fake-token")

    const input = screen.getByLabelText(messages.auth.newPassword)
    expect(input).toHaveAttribute("type", "password")

    const showButtons = screen.getAllByRole("button", {
      name: messages.common.showPassword,
    })
    await user.click(showButtons[0])
    expect(input).toHaveAttribute("type", "text")
  })
})
