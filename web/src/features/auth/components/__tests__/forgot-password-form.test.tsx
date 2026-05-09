import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { ForgotPasswordForm } from "../forgot-password-form"

// Mock next-intl navigation (Link only — useRouter not needed by
// this form).
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
    forgotPassword: vi.fn(),
  }
})

import { forgotPassword } from "@/features/auth/api/auth-api"
const mockForgotPassword = vi.mocked(forgotPassword)

function renderForm() {
  return render(
    <NextIntlClientProvider locale="fr" messages={messages}>
      <ForgotPasswordForm />
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("ForgotPasswordForm", () => {
  it("renders an email field with the corresponding label", () => {
    renderForm()
    expect(screen.getByLabelText(messages.auth.email)).toBeInTheDocument()
  })

  it("renders the send-link CTA", () => {
    renderForm()
    expect(
      screen.getByRole("button", { name: messages.auth.sendResetLink }),
    ).toBeInTheDocument()
  })

  it("renders a back-to-login link in the footer", () => {
    renderForm()
    const links = screen.getAllByRole("link", {
      name: new RegExp(messages.auth.backToLogin, "i"),
    })
    expect(links.length).toBeGreaterThan(0)
    expect(links[0]).toHaveAttribute("href", "/login")
  })

  it("shows a validation error for an empty email submission", async () => {
    const user = userEvent.setup()
    renderForm()

    // Submit with no input — zod email validation fires.
    await user.click(
      screen.getByRole("button", { name: messages.auth.sendResetLink }),
    )

    // Zod schema message is the source of truth in the unit env;
    // the e2e test (`web/e2e/auth.spec.ts`) asserts the localized
    // "Adresse email invalide" rendered via the running app.
    await waitFor(() => {
      expect(
        screen.getByText(/invalid email|adresse email invalide/i),
      ).toBeInTheDocument()
    })
  })

  it("shows the loading copy while submitting", async () => {
    mockForgotPassword.mockImplementation(
      () => new Promise<{ message: string }>(() => {}),
    )

    const user = userEvent.setup()
    renderForm()

    await user.type(
      screen.getByLabelText(messages.auth.email),
      "test@example.com",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.sendResetLink }),
    )

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: messages.auth.sending }),
      ).toBeInTheDocument()
    })
  })

  it("renders the success state when the API call resolves", async () => {
    mockForgotPassword.mockResolvedValueOnce({ message: "ok" })

    const user = userEvent.setup()
    renderForm()

    await user.type(
      screen.getByLabelText(messages.auth.email),
      "test@example.com",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.sendResetLink }),
    )

    await waitFor(() => {
      expect(screen.getByRole("status")).toBeInTheDocument()
    })
    expect(screen.getByText(messages.common.emailSent)).toBeInTheDocument()
    expect(screen.getByText(messages.auth.resetEmailSent)).toBeInTheDocument()
  })

  it("renders the inline error when the API throws", async () => {
    mockForgotPassword.mockRejectedValueOnce(new Error("Bad gateway"))

    const user = userEvent.setup()
    renderForm()

    await user.type(
      screen.getByLabelText(messages.auth.email),
      "test@example.com",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.sendResetLink }),
    )

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("Bad gateway")
    })
  })

  it("the success state exposes a back-to-login link", async () => {
    mockForgotPassword.mockResolvedValueOnce({ message: "ok" })

    const user = userEvent.setup()
    renderForm()

    await user.type(
      screen.getByLabelText(messages.auth.email),
      "test@example.com",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.sendResetLink }),
    )

    await waitFor(() => {
      const link = screen.getByRole("link", {
        name: new RegExp(messages.auth.backToLogin, "i"),
      })
      expect(link).toHaveAttribute("href", "/login")
    })
  })
})
