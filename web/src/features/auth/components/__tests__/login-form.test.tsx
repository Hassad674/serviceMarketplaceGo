import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { LoginForm } from "../login-form"

// Mock next-intl navigation (Link, useRouter)
const mockPush = vi.fn()
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
  useRouter: () => ({
    push: mockPush,
    replace: vi.fn(),
    back: vi.fn(),
    prefetch: vi.fn(),
  }),
}))

// Mock the login API function. We keep the real AuthApiError class so
// the component's `err instanceof AuthApiError` check keeps working
// when the mocked login throws.
vi.mock("@/features/auth/api/auth-api", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@/features/auth/api/auth-api")>()
  return {
    ...actual,
    login: vi.fn(),
  }
})

import { login } from "@/features/auth/api/auth-api"
const mockLogin = vi.mocked(login)

function renderLoginForm() {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <LoginForm />
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("LoginForm", () => {
  it("renders email and password inputs", () => {
    renderLoginForm()

    expect(screen.getByLabelText(messages.auth.email)).toBeInTheDocument()
    expect(screen.getByLabelText(messages.auth.password)).toBeInTheDocument()
  })

  it("renders sign in button", () => {
    renderLoginForm()

    expect(
      screen.getByRole("button", { name: messages.auth.loginTitle }),
    ).toBeInTheDocument()
  })

  it("renders forgot password link", () => {
    renderLoginForm()

    const link = screen.getByRole("link", {
      name: messages.auth.forgotPassword,
    })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute("href", "/forgot-password")
  })

  it("renders create account link", () => {
    renderLoginForm()

    const link = screen.getByRole("link", {
      name: messages.common.createAccount,
    })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute("href", "/register")
  })

  it("shows validation error for empty email submission", async () => {
    const user = userEvent.setup()
    renderLoginForm()

    // Fill password but leave email empty, then submit
    const passwordInput = screen.getByLabelText(messages.auth.password)
    await user.type(passwordInput, "Passw0rd!")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    // Zod validation: email is required
    await waitFor(() => {
      expect(screen.getByText(/invalid email/i)).toBeInTheDocument()
    })
  })

  it("shows validation error for short password", async () => {
    const user = userEvent.setup()
    renderLoginForm()

    const emailInput = screen.getByLabelText(messages.auth.email)
    const passwordInput = screen.getByLabelText(messages.auth.password)

    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "short")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    await waitFor(() => {
      expect(
        screen.getByText(/password must contain at least 8 characters/i),
      ).toBeInTheDocument()
    })
  })

  it("toggles password visibility", async () => {
    const user = userEvent.setup()
    renderLoginForm()

    const passwordInput = screen.getByLabelText(messages.auth.password)
    expect(passwordInput).toHaveAttribute("type", "password")

    // Click the show password button
    const toggleButton = screen.getByRole("button", {
      name: messages.common.showPassword,
    })
    await user.click(toggleButton)

    expect(passwordInput).toHaveAttribute("type", "text")

    // Click the hide password button
    const hideButton = screen.getByRole("button", {
      name: messages.common.hidePassword,
    })
    await user.click(hideButton)

    expect(passwordInput).toHaveAttribute("type", "password")
  })

  it("shows loading state when submitting", async () => {
    // Make login hang (never resolves during this test)
    mockLogin.mockImplementation(
      () => new Promise<never>(() => {}),
    )

    const user = userEvent.setup()
    renderLoginForm()

    const emailInput = screen.getByLabelText(messages.auth.email)
    const passwordInput = screen.getByLabelText(messages.auth.password)

    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "Passw0rd!")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: messages.auth.signingIn }),
      ).toBeInTheDocument()
    })
  })

  it("shows server error on failed login", async () => {
    mockLogin.mockRejectedValueOnce(new Error("Invalid credentials"))

    const user = userEvent.setup()
    renderLoginForm()

    const emailInput = screen.getByLabelText(messages.auth.email)
    const passwordInput = screen.getByLabelText(messages.auth.password)

    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "Passw0rd!")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        "Invalid credentials",
      )
    })
  })

  it("navigates to dashboard on successful login", async () => {
    mockLogin.mockResolvedValueOnce({
      id: "1",
      email: "test@example.com",
      first_name: "John",
      last_name: "Doe",
      display_name: "John Doe",
      role: "provider",
      referrer_enabled: false,
      email_verified: true,
      created_at: "2026-01-01",
    })

    const user = userEvent.setup()
    renderLoginForm()

    const emailInput = screen.getByLabelText(messages.auth.email)
    const passwordInput = screen.getByLabelText(messages.auth.password)

    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "Passw0rd!")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/dashboard")
    })
  })
})
